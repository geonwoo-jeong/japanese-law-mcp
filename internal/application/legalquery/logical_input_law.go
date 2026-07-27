package legalquery

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/resourceinput"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// LawSearchIntentV1Values は、法令名検索の logical input 値を保持する。
type LawSearchIntentV1Values struct {
	Query string
	AsOf  *model.Date
}

// LawSearchIntentV1 は、law.search@1 へ変換できる provider 非依存条件である。
type LawSearchIntentV1 struct {
	query string
	asOf  *model.Date
}

// NewLawSearchIntentV1 は、法令検索能力と同じ共通制約で検証した条件を返す。
func NewLawSearchIntentV1(
	values LawSearchIntentV1Values,
) (LawSearchIntentV1, error) {
	request, err := lawsearch.NewRequest(lawsearch.RequestValues{
		Query: values.Query,
		AsOf:  values.AsOf,
	})
	if err != nil {
		return LawSearchIntentV1{}, fmt.Errorf("law search intent が有効ではありません: %w", err)
	}
	intent := LawSearchIntentV1{query: request.Query()}
	if asOf, exists := request.AsOf(); exists {
		intent.asOf = &asOf
	}
	return intent, nil
}

// Query は、正規化済みの法令検索語を返す。
func (i LawSearchIntentV1) Query() string {
	return i.query
}

// AsOf は、検索基準日と有無を返す。
func (i LawSearchIntentV1) AsOf() (model.Date, bool) {
	return optionalDate(i.asOf)
}

// InputKind は、law_search variant を返す。
func (i LawSearchIntentV1) InputKind() LogicalInputKind {
	return InputKindLawSearch
}

// Validate は、law.search@1 へ lossless に変換できることを検証する。
func (i LawSearchIntentV1) Validate() error {
	_, err := lawsearch.NewRequest(lawsearch.RequestValues{
		Query: i.query,
		AsOf:  i.asOf,
	})
	return err
}

// UnmarshalJSON は、constructor を介さない直接復元を拒否する。
func (*LawSearchIntentV1) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LawSearchIntentV1 は JSON から直接復元できません。NewLawSearchIntentV1 を使用してください",
	)
}

func (i LawSearchIntentV1) clone() LawSearchIntentV1 {
	return LawSearchIntentV1{
		query: i.query,
		asOf:  cloneOptionalDate(i.asOf),
	}
}

func (LawSearchIntentV1) legalQueryLogicalInput() {}

// LawContentSearchIntentV1Values は、法令本文検索の logical input 値を保持する。
type LawContentSearchIntentV1Values struct {
	AllTerms     []string
	AnyTerms     []string
	ExcludeTerms []string
	AsOf         *model.Date
}

// LawContentSearchIntentV1 は、構造化された provider 非依存の本文検索条件である。
type LawContentSearchIntentV1 struct {
	allTerms     []string
	anyTerms     []string
	excludeTerms []string
	asOf         *model.Date
}

// NewLawContentSearchIntentV1 は、本文検索能力と同じ共通制約で検証した条件を返す。
func NewLawContentSearchIntentV1(
	values LawContentSearchIntentV1Values,
) (LawContentSearchIntentV1, error) {
	request, err := lawcontentsearch.NewRequest(lawcontentsearch.RequestValues{
		AllTerms:     values.AllTerms,
		AnyTerms:     values.AnyTerms,
		ExcludeTerms: values.ExcludeTerms,
		AsOf:         values.AsOf,
	})
	if err != nil {
		return LawContentSearchIntentV1{}, fmt.Errorf(
			"law content search intent が有効ではありません: %w",
			err,
		)
	}
	intent := LawContentSearchIntentV1{
		allTerms:     request.AllTerms(),
		anyTerms:     request.AnyTerms(),
		excludeTerms: request.ExcludeTerms(),
	}
	if asOf, exists := request.AsOf(); exists {
		intent.asOf = &asOf
	}
	return intent, nil
}

// AllTerms は、すべてを含む検索語の複製を返す。
func (i LawContentSearchIntentV1) AllTerms() []string {
	return append([]string{}, i.allTerms...)
}

// AnyTerms は、いずれかを含む検索語の複製を返す。
func (i LawContentSearchIntentV1) AnyTerms() []string {
	return append([]string{}, i.anyTerms...)
}

// ExcludeTerms は、除外する検索語の複製を返す。
func (i LawContentSearchIntentV1) ExcludeTerms() []string {
	return append([]string{}, i.excludeTerms...)
}

// AsOf は、検索基準日と有無を返す。
func (i LawContentSearchIntentV1) AsOf() (model.Date, bool) {
	return optionalDate(i.asOf)
}

// InputKind は、law_content_search variant を返す。
func (i LawContentSearchIntentV1) InputKind() LogicalInputKind {
	return InputKindLawContentSearch
}

// Validate は、law.content.search@1 へ lossless に変換できることを検証する。
func (i LawContentSearchIntentV1) Validate() error {
	_, err := lawcontentsearch.NewRequest(lawcontentsearch.RequestValues{
		AllTerms:     i.allTerms,
		AnyTerms:     i.anyTerms,
		ExcludeTerms: i.excludeTerms,
		AsOf:         i.asOf,
	})
	return err
}

// UnmarshalJSON は、constructor を介さない直接復元を拒否する。
func (*LawContentSearchIntentV1) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LawContentSearchIntentV1 は JSON から直接復元できません。NewLawContentSearchIntentV1 を使用してください",
	)
}

func (i LawContentSearchIntentV1) clone() LawContentSearchIntentV1 {
	return LawContentSearchIntentV1{
		allTerms:     append([]string{}, i.allTerms...),
		anyTerms:     append([]string{}, i.anyTerms...),
		excludeTerms: append([]string{}, i.excludeTerms...),
		asOf:         cloneOptionalDate(i.asOf),
	}
}

func (LawContentSearchIntentV1) legalQueryLogicalInput() {}

// LawReadIntentV1Values は、法令本文読取りの logical input 値を保持する。
type LawReadIntentV1Values struct {
	LawID      string
	RevisionID string
	AsOf       *model.Date
	Ref        *model.SourceResourceRef
}

// LawReadIntentV1 は、法令 ID または入力 ref による法令読取り条件である。
type LawReadIntentV1 struct {
	lawID      string
	revisionID string
	asOf       *model.Date
	ref        *model.SourceResourceRef
}

// NewLawReadIntentV1 は、対象の排他性を検証した法令読取り条件を返す。
func NewLawReadIntentV1(values LawReadIntentV1Values) (LawReadIntentV1, error) {
	intent := LawReadIntentV1{
		lawID:      values.LawID,
		revisionID: values.RevisionID,
		asOf:       cloneOptionalDate(values.AsOf),
		ref:        cloneOptionalResourceRef(values.Ref),
	}
	if err := intent.Validate(); err != nil {
		return LawReadIntentV1{}, err
	}
	return intent, nil
}

// LawID は、route 選択後に参照へ変換する法令 ID と有無を返す。
func (i LawReadIntentV1) LawID() (string, bool) {
	return i.lawID, i.lawID != ""
}

// RevisionID は、正確な法令リビジョン ID と有無を返す。
func (i LawReadIntentV1) RevisionID() (string, bool) {
	return i.revisionID, i.revisionID != ""
}

// AsOf は、法令リビジョンの選択基準日と有無を返す。
func (i LawReadIntentV1) AsOf() (model.Date, bool) {
	return optionalDate(i.asOf)
}

// Ref は、入力で受け取った法令参照と有無を返す。
func (i LawReadIntentV1) Ref() (model.SourceResourceRef, bool) {
	return optionalResourceRef(i.ref)
}

// InputKind は、law_read variant を返す。
func (i LawReadIntentV1) InputKind() LogicalInputKind {
	return InputKindLawRead
}

// Validate は、ID 形または ref 形のいずれか一方であることを検証する。
func (i LawReadIntentV1) Validate() error {
	if i.ref != nil {
		return i.validateRefForm()
	}
	if err := resourceinput.ValidateLawIdentifiers(
		"lawId",
		"revisionId",
		i.lawID,
		i.revisionID,
	); err != nil {
		return err
	}
	if i.revisionID != "" && i.asOf != nil {
		return fmt.Errorf("revisionId と asOf は同時に指定できません")
	}
	return validateOptionalLogicalDate(i.asOf)
}

func (i LawReadIntentV1) validateRefForm() error {
	if i.lawID != "" || i.revisionID != "" || i.asOf != nil {
		return fmt.Errorf("ref と lawId、revisionId または asOf は同時に指定できません")
	}
	return resourceinput.ValidateLawRef("ref", *i.ref)
}

// UnmarshalJSON は、constructor を介さない直接復元を拒否する。
func (*LawReadIntentV1) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LawReadIntentV1 は JSON から直接復元できません。NewLawReadIntentV1 を使用してください",
	)
}

func (i LawReadIntentV1) clone() LawReadIntentV1 {
	return LawReadIntentV1{
		lawID:      i.lawID,
		revisionID: i.revisionID,
		asOf:       cloneOptionalDate(i.asOf),
		ref:        cloneOptionalResourceRef(i.ref),
	}
}

func (LawReadIntentV1) legalQueryLogicalInput() {}

// LawArticleReadIntentV1Values は、法令条文読取りの logical input 値を保持する。
type LawArticleReadIntentV1Values struct {
	LawID    string
	Ref      *model.SourceResourceRef
	Location model.LawArticleLocation
	AsOf     *model.Date
}

// LawArticleReadIntentV1 は、法令と条文位置を表す provider 非依存条件である。
type LawArticleReadIntentV1 struct {
	lawID    string
	ref      *model.SourceResourceRef
	location model.LawArticleLocation
	asOf     *model.Date
}

// NewLawArticleReadIntentV1 は、対象と位置を検証した条文読取り条件を返す。
func NewLawArticleReadIntentV1(
	values LawArticleReadIntentV1Values,
) (LawArticleReadIntentV1, error) {
	intent := LawArticleReadIntentV1{
		lawID:    values.LawID,
		ref:      cloneOptionalResourceRef(values.Ref),
		location: values.Location,
		asOf:     cloneOptionalDate(values.AsOf),
	}
	if err := intent.Validate(); err != nil {
		return LawArticleReadIntentV1{}, err
	}
	return intent, nil
}

// LawID は、route 選択後に参照へ変換する法令 ID と有無を返す。
func (i LawArticleReadIntentV1) LawID() (string, bool) {
	return i.lawID, i.lawID != ""
}

// Ref は、入力で受け取った法令参照と有無を返す。
func (i LawArticleReadIntentV1) Ref() (model.SourceResourceRef, bool) {
	return optionalResourceRef(i.ref)
}

// Location は、取得対象の条または項を返す。
func (i LawArticleReadIntentV1) Location() model.LawArticleLocation {
	return i.location
}

// AsOf は、法令リビジョンの選択基準日と有無を返す。
func (i LawArticleReadIntentV1) AsOf() (model.Date, bool) {
	return optionalDate(i.asOf)
}

// InputKind は、law_article_read variant を返す。
func (i LawArticleReadIntentV1) InputKind() LogicalInputKind {
	return InputKindLawArticleRead
}

// Validate は、対象の排他性、条文位置および日付を検証する。
func (i LawArticleReadIntentV1) Validate() error {
	if (i.lawID == "") == (i.ref == nil) {
		return fmt.Errorf("lawId または ref のどちらか一方だけが必要です")
	}
	if i.ref != nil {
		if err := resourceinput.ValidateLawRef("ref", *i.ref); err != nil {
			return err
		}
		if _, versioned := i.ref.Key().VersionID(); versioned && i.asOf != nil {
			return fmt.Errorf("版を含む ref と asOf は同時に指定できません")
		}
	} else {
		if err := resourceinput.ValidateLawIdentifiers(
			"lawId",
			"revisionId",
			i.lawID,
			"",
		); err != nil {
			return err
		}
	}
	if err := i.location.Validate(); err != nil {
		return fmt.Errorf("location が有効ではありません: %w", err)
	}
	return validateOptionalLogicalDate(i.asOf)
}

// UnmarshalJSON は、constructor を介さない直接復元を拒否する。
func (*LawArticleReadIntentV1) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LawArticleReadIntentV1 は JSON から直接復元できません。NewLawArticleReadIntentV1 を使用してください",
	)
}

func (i LawArticleReadIntentV1) clone() LawArticleReadIntentV1 {
	return LawArticleReadIntentV1{
		lawID:    i.lawID,
		ref:      cloneOptionalResourceRef(i.ref),
		location: i.location,
		asOf:     cloneOptionalDate(i.asOf),
	}
}

func (LawArticleReadIntentV1) legalQueryLogicalInput() {}

// LawUpdateListIntentV1Values は、法令更新一覧の logical input 値を保持する。
type LawUpdateListIntentV1Values struct {
	Date model.Date
}

// LawUpdateListIntentV1 は、一つの暦日を対象にする更新一覧条件である。
type LawUpdateListIntentV1 struct {
	date model.Date
}

// NewLawUpdateListIntentV1 は、検証済みの更新一覧条件を返す。
func NewLawUpdateListIntentV1(
	values LawUpdateListIntentV1Values,
) (LawUpdateListIntentV1, error) {
	request, err := lawupdatelist.NewRequest(
		lawupdatelist.RequestValues{Date: values.Date},
	)
	if err != nil {
		return LawUpdateListIntentV1{}, fmt.Errorf(
			"law update list intent が有効ではありません: %w",
			err,
		)
	}
	return LawUpdateListIntentV1{date: request.Date()}, nil
}

// Date は、更新一覧の対象日を返す。
func (i LawUpdateListIntentV1) Date() model.Date {
	return i.date
}

// InputKind は、law_updates variant を返す。
func (i LawUpdateListIntentV1) InputKind() LogicalInputKind {
	return InputKindLawUpdates
}

// Validate は、law.update.list@1 へ変換できる暦日を検証する。
func (i LawUpdateListIntentV1) Validate() error {
	_, err := lawupdatelist.NewRequest(
		lawupdatelist.RequestValues{Date: i.date},
	)
	return err
}

// UnmarshalJSON は、constructor を介さない直接復元を拒否する。
func (*LawUpdateListIntentV1) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LawUpdateListIntentV1 は JSON から直接復元できません。NewLawUpdateListIntentV1 を使用してください",
	)
}

func (LawUpdateListIntentV1) legalQueryLogicalInput() {}

func validateOptionalLogicalDate(value *model.Date) error {
	if value == nil {
		return nil
	}
	if err := value.Validate(); err != nil {
		return fmt.Errorf("asOf が有効ではありません: %w", err)
	}
	return nil
}
