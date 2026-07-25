package model

import (
	"encoding/json"
	"fmt"
	"time"
)

const dateLayout = "2006-01-02"

// Date は、暦日を YYYY-MM-DD 形式で保持する。
type Date struct {
	value string
}

// NewDate は、実在する暦日だけを受け入れる Date を返す。
func NewDate(value string) (Date, error) {
	date := Date{value: value}
	if err := date.Validate(); err != nil {
		return Date{}, err
	}
	return date, nil
}

// String は、YYYY-MM-DD 形式の日付を返す。
func (d Date) String() string {
	return d.value
}

// IsZero は、日付が設定されていないかを返す。
func (d Date) IsZero() bool {
	return d.value == ""
}

// Validate は、日付が SOT-MODEL-009 の形式と暦日に適合するか確認する。
func (d Date) Validate() error {
	parsed, err := time.Parse(dateLayout, d.value)
	if err != nil || parsed.Format(dateLayout) != d.value {
		return fmt.Errorf("日付は実在する YYYY-MM-DD 形式でなければなりません")
	}
	return nil
}

// MarshalJSON は、日付を JSON 文字列として表す。
func (d Date) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(d.value)
}
