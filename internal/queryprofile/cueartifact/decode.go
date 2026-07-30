package cueartifact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

const (
	maximumArtifactBytes = 256 << 10
	maximumJSONDepth     = 16
	maximumJSONValues    = 100000
)

type rawDocument struct {
	schemaVersion int
	profileID     string
	cueSetVersion string
	cues          []rawEntry
}

type rawEntry struct {
	cueID       string
	category    string
	value       string
	intentGroup *string
	signal      *string
	syntaxRole  string
	terms       []string
}

type jsonFrame struct {
	open         json.Delim
	expectingKey bool
	keys         map[string]struct{}
}

// Load は、閉じた schema version 3 の cue 成果物を fail closed で読み込む。
func Load(data []byte) (*Artifact, error) {
	if len(data) == 0 || len(data) > maximumArtifactBytes {
		return nil, fmt.Errorf(
			"cues.json は 1 byte 以上 %d byte 以下でなければなりません",
			maximumArtifactBytes,
		)
	}
	if err := inspectJSON(data); err != nil {
		return nil, err
	}
	document, err := decodeDocument(data)
	if err != nil {
		return nil, err
	}
	return buildArtifact(document)
}

func inspectJSON(data []byte) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("cues.json は有効な UTF-8 でなければなりません")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	frames := make([]jsonFrame, 0, 4)
	rootValues := 0
	valueCount := 0
	rootIsObject := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("cues.json の JSON 構造が有効ではありません: %w", err)
		}
		switch value := token.(type) {
		case json.Delim:
			switch value {
			case '{', '[':
				if err := beginJSONValue(
					&frames,
					&rootValues,
					&valueCount,
				); err != nil {
					return err
				}
				if len(frames) == 0 {
					rootIsObject = value == '{'
				}
				if len(frames)+1 > maximumJSONDepth {
					return fmt.Errorf(
						"cues.json の JSON nesting depth は %d 以下でなければなりません",
						maximumJSONDepth,
					)
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
				return fmt.Errorf("cues.json の JSON delimiter が有効ではありません")
			}
		case string:
			if len(frames) > 0 {
				current := &frames[len(frames)-1]
				if current.open == '{' && current.expectingKey {
					if _, exists := current.keys[value]; exists {
						return fmt.Errorf(
							"cues.json の JSON object に重複 key %q があります",
							value,
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
				&valueCount,
			); err != nil {
				return err
			}
		default:
			if token == nil {
				return fmt.Errorf("cues.json に null は指定できません")
			}
			if err := completeJSONScalar(
				&frames,
				&rootValues,
				&valueCount,
			); err != nil {
				return err
			}
		}
	}
	if len(frames) != 0 || rootValues != 1 {
		return fmt.Errorf("cues.json は一つの JSON object でなければなりません")
	}
	if !rootIsObject {
		return fmt.Errorf("cues.json の最上位は JSON object でなければなりません")
	}
	return nil
}

func beginJSONValue(
	frames *[]jsonFrame,
	rootValues *int,
	valueCount *int,
) error {
	if len(*frames) == 0 {
		*rootValues++
	} else {
		current := &(*frames)[len(*frames)-1]
		if current.open == '{' && current.expectingKey {
			return fmt.Errorf("cues.json の JSON object が有効ではありません")
		}
	}
	return countJSONValue(valueCount)
}

func completeJSONScalar(
	frames *[]jsonFrame,
	rootValues *int,
	valueCount *int,
) error {
	if len(*frames) == 0 {
		*rootValues++
	} else {
		current := &(*frames)[len(*frames)-1]
		if current.open == '{' {
			if current.expectingKey {
				return fmt.Errorf("cues.json の JSON object が有効ではありません")
			}
			current.expectingKey = true
		}
	}
	return countJSONValue(valueCount)
}

func closeJSONContainer(frames *[]jsonFrame, closing json.Delim) error {
	if len(*frames) == 0 {
		return fmt.Errorf("cues.json の JSON container が対応していません")
	}
	current := (*frames)[len(*frames)-1]
	expected := json.Delim('}')
	if current.open == '[' {
		expected = ']'
	}
	if closing != expected ||
		(current.open == '{' && !current.expectingKey) {
		return fmt.Errorf("cues.json の JSON container が対応していません")
	}
	*frames = (*frames)[:len(*frames)-1]
	if len(*frames) > 0 {
		parent := &(*frames)[len(*frames)-1]
		if parent.open == '{' {
			if parent.expectingKey {
				return fmt.Errorf("cues.json の JSON object が有効ではありません")
			}
			parent.expectingKey = true
		}
	}
	return nil
}

func countJSONValue(valueCount *int) error {
	*valueCount++
	if *valueCount > maximumJSONValues {
		return fmt.Errorf(
			"cues.json は %d JSON value 以下でなければなりません",
			maximumJSONValues,
		)
	}
	return nil
}

func decodeDocument(data []byte) (rawDocument, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return rawDocument{}, fmt.Errorf("cues.json を読み込めません: %w", err)
	}
	if err := validateObjectKeys(
		object,
		[]string{"schemaVersion", "profileId", "cueSetVersion", "cues"},
		nil,
		"cues.json",
	); err != nil {
		return rawDocument{}, err
	}
	var document rawDocument
	if err := decodeField(object, "schemaVersion", &document.schemaVersion); err != nil {
		return rawDocument{}, err
	}
	if err := decodeField(object, "profileId", &document.profileID); err != nil {
		return rawDocument{}, err
	}
	if err := decodeField(object, "cueSetVersion", &document.cueSetVersion); err != nil {
		return rawDocument{}, err
	}
	var entries []json.RawMessage
	if err := decodeField(object, "cues", &entries); err != nil {
		return rawDocument{}, err
	}
	document.cues = make([]rawEntry, 0, len(entries))
	for index, entryData := range entries {
		entry, err := decodeEntry(entryData, index)
		if err != nil {
			return rawDocument{}, err
		}
		document.cues = append(document.cues, entry)
	}
	return document, nil
}

func decodeEntry(data json.RawMessage, index int) (rawEntry, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return rawEntry{}, fmt.Errorf("cues[%d] を読み込めません: %w", index, err)
	}
	if err := validateObjectKeys(
		object,
		[]string{"cueId", "category", "value", "syntaxRole", "terms"},
		[]string{"intentGroup", "signal"},
		fmt.Sprintf("cues[%d]", index),
	); err != nil {
		return rawEntry{}, err
	}
	var entry rawEntry
	for name, target := range map[string]*string{
		"cueId":      &entry.cueID,
		"category":   &entry.category,
		"value":      &entry.value,
		"syntaxRole": &entry.syntaxRole,
	} {
		if err := decodeField(object, name, target); err != nil {
			return rawEntry{}, fmt.Errorf("cues[%d]: %w", index, err)
		}
	}
	if err := decodeField(object, "terms", &entry.terms); err != nil {
		return rawEntry{}, fmt.Errorf("cues[%d]: %w", index, err)
	}
	if value, exists := object["intentGroup"]; exists {
		entry.intentGroup = new(string)
		if err := json.Unmarshal(value, entry.intentGroup); err != nil {
			return rawEntry{}, fmt.Errorf(
				"cues[%d].intentGroup を読み込めません: %w",
				index,
				err,
			)
		}
	}
	if value, exists := object["signal"]; exists {
		entry.signal = new(string)
		if err := json.Unmarshal(value, entry.signal); err != nil {
			return rawEntry{}, fmt.Errorf(
				"cues[%d].signal を読み込めません: %w",
				index,
				err,
			)
		}
	}
	return entry, nil
}

func validateObjectKeys(
	object map[string]json.RawMessage,
	required []string,
	optional []string,
	name string,
) error {
	allowed := make(map[string]struct{}, len(required)+len(optional))
	for _, key := range required {
		allowed[key] = struct{}{}
		if _, exists := object[key]; !exists {
			return fmt.Errorf("%s に必須項目 %q がありません", name, key)
		}
	}
	for _, key := range optional {
		allowed[key] = struct{}{}
	}
	for key := range object {
		if _, exists := allowed[key]; !exists {
			return fmt.Errorf("%s に未知項目 %q があります", name, key)
		}
	}
	return nil
}

func decodeField(
	object map[string]json.RawMessage,
	name string,
	target any,
) error {
	if err := json.Unmarshal(object[name], target); err != nil {
		return fmt.Errorf("%s を読み込めません: %w", name, err)
	}
	return nil
}
