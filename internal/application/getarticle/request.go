package getarticle

import (
	"fmt"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/getlaw"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

// RequestValues は、公開 get_article 入力の境界値を保持する。
type RequestValues struct {
	LawID     string
	Provision *model.LawArticleProvision
	Article   string
	Paragraph *int
	AsOf      *model.Date
}

// Request は、正規化済みの公開条文取得入力を不変に保持する。
type Request struct {
	law      getlaw.Request
	location model.LawArticleLocation
}

// NewRequest は、公開入力を法令指定と正規化済み条文位置へ変換する。
func NewRequest(values RequestValues) (Request, error) {
	law, err := getlaw.NewRequest(getlaw.RequestValues{
		LawID: values.LawID,
		AsOf:  values.AsOf,
	})
	if err != nil {
		return Request{}, err
	}
	provision := model.LawArticleProvisionMain
	if values.Provision != nil {
		provision = *values.Provision
	}
	location, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:       provision,
		ArticleNumber:   values.Article,
		ParagraphNumber: values.Paragraph,
	})
	if err != nil {
		return Request{}, err
	}
	return Request{law: law, location: location}, nil
}

// LawID は、正規化済みの公式法令識別子を返す。
func (r Request) LawID() string {
	return r.law.LawID()
}

// AsOf は、検索基準日と有無を返す。
func (r Request) AsOf() (model.Date, bool) {
	return r.law.AsOf()
}

// Location は、正規化済みの条文位置を返す。
func (r Request) Location() model.LawArticleLocation {
	return r.location
}

// Validate は、公開 get_article の入力制約を確認する。
func (r Request) Validate() error {
	if err := r.law.Validate(); err != nil {
		return err
	}
	if err := r.location.Validate(); err != nil {
		return fmt.Errorf("location が有効ではありません: %w", err)
	}
	return nil
}

// UnmarshalJSON は、境界値を介さない直接復元を拒否する。
func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Request は JSON から直接復元できません。NewRequest を使用してください",
	)
}
