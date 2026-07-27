package model

import (
	"encoding/json"
	"fmt"
)

// JudicialPublicationCategory は、裁判所サイトの掲載カテゴリーを正規化した値である。
type JudicialPublicationCategory string

const (
	// JudicialPublicationCategorySupremeCourt は、最高裁判例を表す。
	JudicialPublicationCategorySupremeCourt JudicialPublicationCategory = "supreme_court"
	// JudicialPublicationCategoryHighCourt は、高裁判例を表す。
	JudicialPublicationCategoryHighCourt JudicialPublicationCategory = "high_court"
	// JudicialPublicationCategoryLowerCourt は、下級裁判所の裁判例を表す。
	JudicialPublicationCategoryLowerCourt JudicialPublicationCategory = "lower_court"
	// JudicialPublicationCategoryAdministrative は、行政事件裁判例を表す。
	JudicialPublicationCategoryAdministrative JudicialPublicationCategory = "administrative"
	// JudicialPublicationCategoryLabor は、労働事件裁判例を表す。
	JudicialPublicationCategoryLabor JudicialPublicationCategory = "labor"
	// JudicialPublicationCategoryIntellectualProperty は、知的財産裁判例を表す。
	JudicialPublicationCategoryIntellectualProperty JudicialPublicationCategory = "intellectual_property"
)

// JudicialDecisionSummaryValues は、JudicialDecisionSummary の作成に必要な値を保持する。
type JudicialDecisionSummaryValues struct {
	DecisionID          string
	PublicationCategory JudicialPublicationCategory
	SourceCategoryLabel string
	CaseNumber          string
	CaseName            *string
	DecisionDate        Date
	CourtName           string
	BranchName          *string
	DivisionName        *string
	DecisionType        *string
	Outcome             *string
	DetailURL           string
	Documents           []JudicialDocumentLink
	Source              InformationSource
}

// JudicialDecisionSummary は、一つの公式裁判例の概要を表す。
type JudicialDecisionSummary struct {
	decisionID          string
	publicationCategory JudicialPublicationCategory
	sourceCategoryLabel string
	caseNumber          string
	caseName            *string
	decisionDate        Date
	courtName           string
	branchName          *string
	divisionName        *string
	decisionType        *string
	outcome             *string
	detailURL           string
	documents           []JudicialDocumentLink
	source              InformationSource
}

// NewJudicialDecisionSummary は、入力を複製して検証済みの裁判例概要を返す。
func NewJudicialDecisionSummary(
	values JudicialDecisionSummaryValues,
) (JudicialDecisionSummary, error) {
	summary := JudicialDecisionSummary{
		decisionID:          values.DecisionID,
		publicationCategory: values.PublicationCategory,
		sourceCategoryLabel: values.SourceCategoryLabel,
		caseNumber:          values.CaseNumber,
		caseName:            cloneOptionalString(values.CaseName),
		decisionDate:        values.DecisionDate,
		courtName:           values.CourtName,
		branchName:          cloneOptionalString(values.BranchName),
		divisionName:        cloneOptionalString(values.DivisionName),
		decisionType:        cloneOptionalString(values.DecisionType),
		outcome:             cloneOptionalString(values.Outcome),
		detailURL:           values.DetailURL,
		documents:           cloneJudicialDocumentLinks(values.Documents),
		source:              values.Source,
	}
	if err := summary.Validate(); err != nil {
		return JudicialDecisionSummary{}, err
	}
	return summary, nil
}

// DecisionID は、公式詳細 URL が示す裁判例識別子を返す。
func (s JudicialDecisionSummary) DecisionID() string {
	return s.decisionID
}

// PublicationCategory は、正規化した掲載カテゴリーを返す。
func (s JudicialDecisionSummary) PublicationCategory() JudicialPublicationCategory {
	return s.publicationCategory
}

// SourceCategoryLabel は、情報源が表示した掲載カテゴリー名を返す。
func (s JudicialDecisionSummary) SourceCategoryLabel() string {
	return s.sourceCategoryLabel
}

// CaseNumber は、情報源が表示した事件番号を返す。
func (s JudicialDecisionSummary) CaseNumber() string {
	return s.caseNumber
}

// CaseName は、情報源が表示した事件名と有無を返す。
func (s JudicialDecisionSummary) CaseName() (string, bool) {
	return optionalStringValue(s.caseName)
}

// DecisionDate は、裁判年月日を返す。
func (s JudicialDecisionSummary) DecisionDate() Date {
	return s.decisionDate
}

// CourtName は、裁判所または法廷の名称を返す。
func (s JudicialDecisionSummary) CourtName() string {
	return s.courtName
}

// BranchName は、情報源が独立して示した支部名と有無を返す。
func (s JudicialDecisionSummary) BranchName() (string, bool) {
	return optionalStringValue(s.branchName)
}

// DivisionName は、情報源が独立して示した部または法廷名と有無を返す。
func (s JudicialDecisionSummary) DivisionName() (string, bool) {
	return optionalStringValue(s.divisionName)
}

// DecisionType は、判決、決定その他の裁判種別と有無を返す。
func (s JudicialDecisionSummary) DecisionType() (string, bool) {
	return optionalStringValue(s.decisionType)
}

// Outcome は、情報源が表示した結果と有無を返す。
func (s JudicialDecisionSummary) Outcome() (string, bool) {
	return optionalStringValue(s.outcome)
}

// DetailURL は、裁判所の公式詳細ページを返す。
func (s JudicialDecisionSummary) DetailURL() string {
	return s.detailURL
}

// Documents は、情報源が直接示した公式文書の複製を返す。
func (s JudicialDecisionSummary) Documents() []JudicialDocumentLink {
	return cloneJudicialDocumentLinks(s.documents)
}

// Source は、裁判例情報を取得した情報源を返す。
func (s JudicialDecisionSummary) Source() InformationSource {
	return s.source
}

// Validate は、裁判例概要の必須項目、列挙値、日付、URL および情報源を確認する。
func (s JudicialDecisionSummary) Validate() error {
	switch {
	case s.decisionID == "":
		return fmt.Errorf("裁判例概要の decisionId は必須です")
	case !s.publicationCategory.valid():
		return fmt.Errorf("publicationCategory が有効ではありません")
	case s.sourceCategoryLabel == "":
		return fmt.Errorf("裁判例概要の sourceCategoryLabel は必須です")
	case s.caseNumber == "":
		return fmt.Errorf("裁判例概要の caseNumber は必須です")
	case s.courtName == "":
		return fmt.Errorf("裁判例概要の courtName は必須です")
	}
	if err := validateJudicialOptionalStrings(s.optionalStrings()); err != nil {
		return err
	}
	if err := s.decisionDate.Validate(); err != nil {
		return fmt.Errorf("decisionDate が有効ではありません: %w", err)
	}
	if !isCourtsOfficialHTTPSURL(s.detailURL) {
		return fmt.Errorf(
			"detailUrl は認証情報を含まない https://www.courts.go.jp/ の URL でなければなりません",
		)
	}
	return s.validateDocumentsAndSource()
}

func (s JudicialDecisionSummary) optionalStrings() map[string]*string {
	return map[string]*string{
		"caseName":     s.caseName,
		"branchName":   s.branchName,
		"divisionName": s.divisionName,
		"decisionType": s.decisionType,
		"outcome":      s.outcome,
	}
}

func (s JudicialDecisionSummary) validateDocumentsAndSource() error {
	if s.documents == nil {
		return fmt.Errorf("documents は空配列または値を持つ配列でなければなりません")
	}
	for index, document := range s.documents {
		if err := document.Validate(); err != nil {
			return fmt.Errorf("documents[%d] が有効ではありません: %w", index, err)
		}
	}
	if err := s.source.Validate(); err != nil {
		return fmt.Errorf("source が有効ではありません: %w", err)
	}
	if s.source.Authority() != AuthorityOfficial ||
		!isCourtsOfficialHTTPSURL(s.source.ServiceURL()) {
		return fmt.Errorf("source は裁判所の公式情報源でなければなりません")
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-020 の項目名で裁判例概要を表す。
func (s JudicialDecisionSummary) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		DecisionID          string                      `json:"decisionId"`
		PublicationCategory JudicialPublicationCategory `json:"publicationCategory"`
		SourceCategoryLabel string                      `json:"sourceCategoryLabel"`
		CaseNumber          string                      `json:"caseNumber"`
		CaseName            *string                     `json:"caseName,omitempty"`
		DecisionDate        Date                        `json:"decisionDate"`
		CourtName           string                      `json:"courtName"`
		BranchName          *string                     `json:"branchName,omitempty"`
		DivisionName        *string                     `json:"divisionName,omitempty"`
		DecisionType        *string                     `json:"decisionType,omitempty"`
		Outcome             *string                     `json:"outcome,omitempty"`
		DetailURL           string                      `json:"detailUrl"`
		Documents           []JudicialDocumentLink      `json:"documents"`
		Source              InformationSource           `json:"source"`
	}{
		DecisionID:          s.decisionID,
		PublicationCategory: s.publicationCategory,
		SourceCategoryLabel: s.sourceCategoryLabel,
		CaseNumber:          s.caseNumber,
		CaseName:            cloneOptionalString(s.caseName),
		DecisionDate:        s.decisionDate,
		CourtName:           s.courtName,
		BranchName:          cloneOptionalString(s.branchName),
		DivisionName:        cloneOptionalString(s.divisionName),
		DecisionType:        cloneOptionalString(s.decisionType),
		Outcome:             cloneOptionalString(s.outcome),
		DetailURL:           s.detailURL,
		Documents:           cloneJudicialDocumentLinks(s.documents),
		Source:              s.source,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*JudicialDecisionSummary) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialDecisionSummary は JSON から直接復元できません。" +
			"境界専用の入力型から NewJudicialDecisionSummary を使用してください",
	)
}

func (c JudicialPublicationCategory) valid() bool {
	switch c {
	case JudicialPublicationCategorySupremeCourt,
		JudicialPublicationCategoryHighCourt,
		JudicialPublicationCategoryLowerCourt,
		JudicialPublicationCategoryAdministrative,
		JudicialPublicationCategoryLabor,
		JudicialPublicationCategoryIntellectualProperty:
		return true
	default:
		return false
	}
}

func cloneJudicialDocumentLinks(
	values []JudicialDocumentLink,
) []JudicialDocumentLink {
	if values == nil {
		return nil
	}
	cloned := make([]JudicialDocumentLink, len(values))
	copy(cloned, values)
	return cloned
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func optionalStringValue(value *string) (string, bool) {
	if value == nil {
		return "", false
	}
	return *value, true
}

func validateJudicialOptionalStrings(values map[string]*string) error {
	for name, value := range values {
		if value != nil && *value == "" {
			return fmt.Errorf("%s は存在する場合に空文字列にできません", name)
		}
	}
	return nil
}
