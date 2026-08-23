package lawrevisionlist

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/resourceinput"
)

const (
	CapabilityID = "law.revision.list"
	MajorVersion = 1
)

type RequestValues struct {
	LawIDOrNumber string
}

// Request は、法令 ID 又は法令番号を不変に保持する。
type Request struct {
	lawIDOrNumber string
}

func NewRequest(values RequestValues) (Request, error) {
	request := Request{lawIDOrNumber: strings.Trim(values.LawIDOrNumber, " ")}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

func (r Request) LawIDOrNumber() string { return r.lawIDOrNumber }

func (r Request) Validate() error {
	return resourceinput.ValidateLawIdentifiers(
		"lawIdOrNumber",
		"revisionId",
		r.lawIDOrNumber,
		"",
	)
}

func (r Request) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		LawIDOrNumber string `json:"lawIdOrNumber"`
	}{LawIDOrNumber: r.lawIDOrNumber})
}

func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("Request は JSON から直接復元できません。NewRequest を使用してください")
}
