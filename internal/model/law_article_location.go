package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

const maximumLawArticleNumberBytes = 64

// LawArticleProvision は、条文が属する本則または原始附則を表す。
type LawArticleProvision string

const (
	// LawArticleProvisionMain は、本則を表す。
	LawArticleProvisionMain LawArticleProvision = "main"
	// LawArticleProvisionSupplementary は、原始附則を表す。
	LawArticleProvisionSupplementary LawArticleProvision = "supplementary"
)

// LawArticleLocationValues は、LawArticleLocation の作成に必要な値を保持する。
type LawArticleLocationValues struct {
	Provision       LawArticleProvision
	ArticleNumber   string
	ParagraphNumber *int
}

// LawArticleLocation は、表現方法に依存しない法令の条または項の位置を保持する。
type LawArticleLocation struct {
	provision       LawArticleProvision
	articleNumber   string
	paragraphNumber *int
}

// NewLawArticleLocation は、入力を複製して検証済みの LawArticleLocation を返す。
func NewLawArticleLocation(values LawArticleLocationValues) (LawArticleLocation, error) {
	location := LawArticleLocation{
		provision:     values.Provision,
		articleNumber: values.ArticleNumber,
	}
	if values.ParagraphNumber != nil {
		paragraphNumber := *values.ParagraphNumber
		location.paragraphNumber = &paragraphNumber
	}
	if err := location.Validate(); err != nil {
		return LawArticleLocation{}, err
	}
	return location, nil
}

// Provision は、本則または原始附則の区分を返す。
func (l LawArticleLocation) Provision() LawArticleProvision {
	return l.provision
}

// ArticleNumber は、正規化済みの条番号を返す。
func (l LawArticleLocation) ArticleNumber() string {
	return l.articleNumber
}

// ParagraphNumber は、項番号と有無を返す。
func (l LawArticleLocation) ParagraphNumber() (int, bool) {
	if l.paragraphNumber == nil {
		return 0, false
	}
	return *l.paragraphNumber, true
}

// Validate は、条文位置の区分、条番号および項番号を確認する。
func (l LawArticleLocation) Validate() error {
	switch l.provision {
	case LawArticleProvisionMain, LawArticleProvisionSupplementary:
	default:
		return fmt.Errorf("provision は main または supplementary でなければなりません")
	}
	if err := validateLawArticleNumber(l.articleNumber); err != nil {
		return err
	}
	if l.paragraphNumber != nil && *l.paragraphNumber < 1 {
		return fmt.Errorf("paragraphNumber は 1 以上でなければなりません")
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-018 の項目名で条文位置を表す。
func (l LawArticleLocation) MarshalJSON() ([]byte, error) {
	if err := l.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Provision       LawArticleProvision `json:"provision"`
		ArticleNumber   string              `json:"articleNumber"`
		ParagraphNumber *int                `json:"paragraphNumber,omitempty"`
	}{
		Provision:       l.provision,
		ArticleNumber:   l.articleNumber,
		ParagraphNumber: cloneOptionalInt(l.paragraphNumber),
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*LawArticleLocation) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LawArticleLocation は JSON から直接復元できません。境界専用の入力型から NewLawArticleLocation を使用してください",
	)
}

func validateLawArticleNumber(value string) error {
	if err := validateLawArticleNumberBoundary(value); err != nil {
		return err
	}
	if err := validateLawArticleNumberSegments(value); err != nil {
		return err
	}
	return nil
}

func validateLawVersionArticleNumber(value string) error {
	if err := validateLawArticleNumberBoundary(value); err != nil {
		return err
	}

	separator := strings.IndexByte(value, ':')
	if separator < 0 {
		return validateLawArticleNumberSegments(value)
	}
	if separator == 0 || separator == len(value)-1 ||
		strings.IndexByte(value[separator+1:], ':') >= 0 {
		return fmt.Errorf("articleNumber の条範囲は二つの条番号を一つの : で連結しなければなりません")
	}

	start := value[:separator]
	end := value[separator+1:]
	if validateLawArticleNumberSegments(start) != nil ||
		validateLawArticleNumberSegments(end) != nil {
		return fmt.Errorf("articleNumber の条範囲の両端は正の十進整数を _ で連結した正規形でなければなりません")
	}
	if compareCanonicalLawArticleNumbers(start, end) >= 0 {
		return fmt.Errorf("articleNumber の条範囲は開始条番号が終了条番号より前でなければなりません")
	}
	return nil
}

func validateLawArticleNumberBoundary(value string) error {
	switch {
	case value == "":
		return fmt.Errorf("articleNumber は必須です")
	case !utf8.ValidString(value):
		return fmt.Errorf("articleNumber は有効な UTF-8 でなければなりません")
	case len(value) > maximumLawArticleNumberBytes:
		return fmt.Errorf("articleNumber は UTF-8 で 64 byte 以下でなければなりません")
	}
	return nil
}

func validateLawArticleNumberSegments(value string) error {
	segmentStart := true
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character == '_' {
			if segmentStart {
				return fmt.Errorf("articleNumber は正の十進整数を _ で連結した正規形でなければなりません")
			}
			segmentStart = true
			continue
		}
		if character < '0' || character > '9' || (segmentStart && character == '0') {
			return fmt.Errorf("articleNumber は正の十進整数を _ で連結した正規形でなければなりません")
		}
		segmentStart = false
	}
	if segmentStart {
		return fmt.Errorf("articleNumber は正の十進整数を _ で連結した正規形でなければなりません")
	}
	return nil
}

func compareCanonicalLawArticleNumbers(left, right string) int {
	leftSegments := strings.Split(left, "_")
	rightSegments := strings.Split(right, "_")
	sharedLength := min(len(leftSegments), len(rightSegments))
	for index := 0; index < sharedLength; index++ {
		if len(leftSegments[index]) != len(rightSegments[index]) {
			return len(leftSegments[index]) - len(rightSegments[index])
		}
		if leftSegments[index] < rightSegments[index] {
			return -1
		}
		if leftSegments[index] > rightSegments[index] {
			return 1
		}
	}
	return len(leftSegments) - len(rightSegments)
}

func cloneOptionalInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
