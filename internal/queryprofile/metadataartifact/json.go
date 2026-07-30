package metadataartifact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

type jsonFrame struct {
	open         json.Delim
	expectingKey bool
	keys         map[string]struct{}
}

func inspectJSON(data []byte) error {
	if !utf8.Valid(data) {
		return fmt.Errorf(
			"profile.json は有効な UTF-8 でなければなりません",
		)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	frames := make([]jsonFrame, 0, 4)
	rootValues := 0
	rootIsObject := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf(
				"profile.json の JSON 構造が有効ではありません: %w",
				err,
			)
		}
		switch value := token.(type) {
		case json.Delim:
			switch value {
			case '{', '[':
				if err := beginJSONValue(&frames, &rootValues); err != nil {
					return err
				}
				if len(frames) == 0 {
					rootIsObject = value == '{'
				}
				frame := jsonFrame{open: value}
				if value == '{' {
					frame.expectingKey = true
					frame.keys = make(map[string]struct{})
				}
				frames = append(frames, frame)
			case '}', ']':
				if err := closeJSONContainer(&frames, value); err != nil {
					return err
				}
			default:
				return fmt.Errorf(
					"profile.json の JSON delimiter が有効ではありません",
				)
			}
		case string:
			if len(frames) > 0 {
				current := &frames[len(frames)-1]
				if current.open == '{' && current.expectingKey {
					if _, exists := current.keys[value]; exists {
						return fmt.Errorf(
							"profile.json の JSON object に重複 key があります",
						)
					}
					current.keys[value] = struct{}{}
					current.expectingKey = false
					continue
				}
			}
			if err := completeJSONScalar(
				&frames,
				&rootValues,
			); err != nil {
				return err
			}
		default:
			if token == nil {
				return fmt.Errorf(
					"profile.json に null は指定できません",
				)
			}
			if err := completeJSONScalar(
				&frames,
				&rootValues,
			); err != nil {
				return err
			}
		}
	}
	if len(frames) != 0 || rootValues != 1 {
		return fmt.Errorf(
			"profile.json は一つの JSON object でなければなりません",
		)
	}
	if !rootIsObject {
		return fmt.Errorf(
			"profile.json の最上位は JSON object でなければなりません",
		)
	}
	return nil
}

func beginJSONValue(
	frames *[]jsonFrame,
	rootValues *int,
) error {
	if len(*frames) == 0 {
		*rootValues++
		return nil
	}
	current := &(*frames)[len(*frames)-1]
	if current.open == '{' && current.expectingKey {
		return fmt.Errorf(
			"profile.json の JSON object が有効ではありません",
		)
	}
	return nil
}

func completeJSONScalar(
	frames *[]jsonFrame,
	rootValues *int,
) error {
	if len(*frames) == 0 {
		*rootValues++
		return nil
	}
	current := &(*frames)[len(*frames)-1]
	if current.open == '{' {
		if current.expectingKey {
			return fmt.Errorf(
				"profile.json の JSON object が有効ではありません",
			)
		}
		current.expectingKey = true
	}
	return nil
}

func closeJSONContainer(
	frames *[]jsonFrame,
	closing json.Delim,
) error {
	if len(*frames) == 0 {
		return fmt.Errorf(
			"profile.json の JSON container が対応していません",
		)
	}
	current := (*frames)[len(*frames)-1]
	expected := json.Delim('}')
	if current.open == '[' {
		expected = ']'
	}
	if closing != expected ||
		(current.open == '{' && !current.expectingKey) {
		return fmt.Errorf(
			"profile.json の JSON container が対応していません",
		)
	}
	*frames = (*frames)[:len(*frames)-1]
	if len(*frames) > 0 {
		parent := &(*frames)[len(*frames)-1]
		if parent.open == '{' {
			if parent.expectingKey {
				return fmt.Errorf(
					"profile.json の JSON object が有効ではありません",
				)
			}
			parent.expectingKey = true
		}
	}
	return nil
}
