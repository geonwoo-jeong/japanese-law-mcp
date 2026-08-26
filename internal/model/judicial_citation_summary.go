package model

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
)

// JudicialCitationSummaryValues は、引用 graph 要約の作成値を保持する。
type JudicialCitationSummaryValues struct {
	ConfirmedOutgoingDecisionCount  int
	IncomingCandidateCount          int
	ReferencedProvisionCount        int
	LowerCourtRelationCount         int
	UnresolvedMentionCount          int
	IncomingObservedYearBuckets     []JudicialCitationYearBucket
	IncomingObservedCategoryBuckets []JudicialCitationCategoryBucket
}

// JudicialCitationSummary は、関係件数と被引用候補の観測分布を表す。
type JudicialCitationSummary struct {
	confirmedOutgoingDecisionCount  int
	incomingCandidateCount          int
	referencedProvisionCount        int
	lowerCourtRelationCount         int
	unresolvedMentionCount          int
	incomingObservedYearBuckets     []JudicialCitationYearBucket
	incomingObservedCategoryBuckets []JudicialCitationCategoryBucket
}

func NewJudicialCitationSummary(values JudicialCitationSummaryValues) (JudicialCitationSummary, error) {
	summary := JudicialCitationSummary{
		confirmedOutgoingDecisionCount:  values.ConfirmedOutgoingDecisionCount,
		incomingCandidateCount:          values.IncomingCandidateCount,
		referencedProvisionCount:        values.ReferencedProvisionCount,
		lowerCourtRelationCount:         values.LowerCourtRelationCount,
		unresolvedMentionCount:          values.UnresolvedMentionCount,
		incomingObservedYearBuckets:     slices.Clone(values.IncomingObservedYearBuckets),
		incomingObservedCategoryBuckets: slices.Clone(values.IncomingObservedCategoryBuckets),
	}
	if err := summary.Validate(); err != nil {
		return JudicialCitationSummary{}, err
	}
	return summary, nil
}

func (s JudicialCitationSummary) ConfirmedOutgoingDecisionCount() int {
	return s.confirmedOutgoingDecisionCount
}
func (s JudicialCitationSummary) IncomingCandidateCount() int {
	return s.incomingCandidateCount
}
func (s JudicialCitationSummary) ReferencedProvisionCount() int {
	return s.referencedProvisionCount
}
func (s JudicialCitationSummary) LowerCourtRelationCount() int {
	return s.lowerCourtRelationCount
}
func (s JudicialCitationSummary) UnresolvedMentionCount() int {
	return s.unresolvedMentionCount
}
func (s JudicialCitationSummary) IncomingObservedYearBuckets() []JudicialCitationYearBucket {
	return slices.Clone(s.incomingObservedYearBuckets)
}
func (s JudicialCitationSummary) IncomingObservedCategoryBuckets() []JudicialCitationCategoryBucket {
	return slices.Clone(s.incomingObservedCategoryBuckets)
}

func (s JudicialCitationSummary) Validate() error {
	for _, value := range []int{
		s.confirmedOutgoingDecisionCount,
		s.incomingCandidateCount,
		s.referencedProvisionCount,
		s.lowerCourtRelationCount,
		s.unresolvedMentionCount,
	} {
		if value < 0 {
			return fmt.Errorf("summary の件数は 0 以上でなければなりません")
		}
	}
	if s.incomingObservedYearBuckets == nil || s.incomingObservedCategoryBuckets == nil {
		return fmt.Errorf("summary の bucket 配列は nil にできません")
	}
	if err := validateJudicialCitationYearBuckets(s.incomingObservedYearBuckets); err != nil {
		return err
	}
	if err := validateJudicialCitationCategoryBuckets(s.incomingObservedCategoryBuckets); err != nil {
		return err
	}
	if sumJudicialCitationYearBuckets(s.incomingObservedYearBuckets) != s.incomingCandidateCount {
		return fmt.Errorf("年別 bucket の合計は incomingCandidateCount と一致しなければなりません")
	}
	if sumJudicialCitationCategoryBuckets(s.incomingObservedCategoryBuckets) != s.incomingCandidateCount {
		return fmt.Errorf("掲載カテゴリ別 bucket の合計は incomingCandidateCount と一致しなければなりません")
	}
	return nil
}

func (s JudicialCitationSummary) validateAgainst(
	edges []JudicialCitationEdge,
	mentions []JudicialCitationUnresolvedMention,
	nodes map[string]JudicialCitationNode,
) error {
	if err := s.Validate(); err != nil {
		return err
	}
	actual := judicialCitationSummaryCounts{unresolved: len(mentions)}
	years := make(map[int]int)
	categories := make(map[JudicialPublicationCategory]int)
	for _, edge := range edges {
		switch edge.RelationType() {
		case JudicialCitationRelationTypeCitesDecision:
			actual.outgoing++
		case JudicialCitationRelationTypePossibleCitesDecision:
			actual.incoming++
			summary, exists := nodes[edge.FromNodeID()].DecisionSummary()
			if !exists {
				return fmt.Errorf("被引用候補に decisionSummary がありません")
			}
			year, err := strconv.Atoi(summary.DecisionDate().String()[:4])
			if err != nil {
				return fmt.Errorf("被引用候補の裁判年を取得できません: %w", err)
			}
			years[year]++
			categories[summary.PublicationCategory()]++
		case JudicialCitationRelationTypeReferencesLawProvision:
			actual.provisions++
		case JudicialCitationRelationTypeHasLowerCourtDecision:
			actual.lowerCourt++
		}
	}
	if actual.outgoing != s.confirmedOutgoingDecisionCount ||
		actual.incoming != s.incomingCandidateCount ||
		actual.provisions != s.referencedProvisionCount ||
		actual.lowerCourt != s.lowerCourtRelationCount ||
		actual.unresolved != s.unresolvedMentionCount {
		return fmt.Errorf("relation 又は未解決言及の件数が一致しません")
	}
	if !equalJudicialCitationYearBuckets(s.incomingObservedYearBuckets, years) {
		return fmt.Errorf("incomingObservedYearBuckets が候補ノードと一致しません")
	}
	if !equalJudicialCitationCategoryBuckets(s.incomingObservedCategoryBuckets, categories) {
		return fmt.Errorf("incomingObservedCategoryBuckets が候補ノードと一致しません")
	}
	return nil
}

type judicialCitationSummaryCounts struct {
	outgoing   int
	incoming   int
	provisions int
	lowerCourt int
	unresolved int
}

func (s JudicialCitationSummary) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ConfirmedOutgoingDecisionCount  int                              `json:"confirmedOutgoingDecisionCount"`
		IncomingCandidateCount          int                              `json:"incomingCandidateCount"`
		ReferencedProvisionCount        int                              `json:"referencedProvisionCount"`
		LowerCourtRelationCount         int                              `json:"lowerCourtRelationCount"`
		UnresolvedMentionCount          int                              `json:"unresolvedMentionCount"`
		IncomingObservedYearBuckets     []JudicialCitationYearBucket     `json:"incomingObservedYearBuckets"`
		IncomingObservedCategoryBuckets []JudicialCitationCategoryBucket `json:"incomingObservedCategoryBuckets"`
	}{
		s.confirmedOutgoingDecisionCount,
		s.incomingCandidateCount,
		s.referencedProvisionCount,
		s.lowerCourtRelationCount,
		s.unresolvedMentionCount,
		slices.Clone(s.incomingObservedYearBuckets),
		slices.Clone(s.incomingObservedCategoryBuckets),
	})
}

func (*JudicialCitationSummary) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialCitationSummary は JSON から直接復元できません。境界専用の入力型から NewJudicialCitationSummary を使用してください",
	)
}

// JudicialCitationYearBucket は、被引用候補の裁判年別件数を表す。
type JudicialCitationYearBucket struct {
	year  int
	count int
}

func NewJudicialCitationYearBucket(year, count int) (JudicialCitationYearBucket, error) {
	bucket := JudicialCitationYearBucket{year: year, count: count}
	if err := bucket.Validate(); err != nil {
		return JudicialCitationYearBucket{}, err
	}
	return bucket, nil
}

func (b JudicialCitationYearBucket) Year() int  { return b.year }
func (b JudicialCitationYearBucket) Count() int { return b.count }
func (b JudicialCitationYearBucket) Validate() error {
	if b.year < 1 || b.count < 1 {
		return fmt.Errorf("year bucket の year と count は 1 以上でなければなりません")
	}
	return nil
}
func (b JudicialCitationYearBucket) MarshalJSON() ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Year  int `json:"year"`
		Count int `json:"count"`
	}{b.year, b.count})
}
func (*JudicialCitationYearBucket) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("JudicialCitationYearBucket は JSON から直接復元できません")
}

// JudicialCitationCategoryBucket は、被引用候補の掲載カテゴリ別件数を表す。
type JudicialCitationCategoryBucket struct {
	publicationCategory JudicialPublicationCategory
	count               int
}

func NewJudicialCitationCategoryBucket(
	publicationCategory JudicialPublicationCategory,
	count int,
) (JudicialCitationCategoryBucket, error) {
	bucket := JudicialCitationCategoryBucket{publicationCategory: publicationCategory, count: count}
	if err := bucket.Validate(); err != nil {
		return JudicialCitationCategoryBucket{}, err
	}
	return bucket, nil
}

func (b JudicialCitationCategoryBucket) PublicationCategory() JudicialPublicationCategory {
	return b.publicationCategory
}
func (b JudicialCitationCategoryBucket) Count() int { return b.count }
func (b JudicialCitationCategoryBucket) Validate() error {
	if !b.publicationCategory.valid() || b.count < 1 {
		return fmt.Errorf("category bucket の掲載カテゴリは有効で count は 1 以上でなければなりません")
	}
	return nil
}
func (b JudicialCitationCategoryBucket) MarshalJSON() ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		PublicationCategory JudicialPublicationCategory `json:"publicationCategory"`
		Count               int                         `json:"count"`
	}{b.publicationCategory, b.count})
}
func (*JudicialCitationCategoryBucket) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("JudicialCitationCategoryBucket は JSON から直接復元できません")
}

func validateJudicialCitationYearBuckets(values []JudicialCitationYearBucket) error {
	lastYear := 0
	for index, bucket := range values {
		if err := bucket.Validate(); err != nil {
			return fmt.Errorf("incomingObservedYearBuckets[%d] が有効ではありません: %w", index, err)
		}
		if index > 0 && bucket.Year() <= lastYear {
			return fmt.Errorf("incomingObservedYearBuckets は年の昇順でなければなりません")
		}
		lastYear = bucket.Year()
	}
	return nil
}

func validateJudicialCitationCategoryBuckets(values []JudicialCitationCategoryBucket) error {
	lastOrder := -1
	for index, bucket := range values {
		if err := bucket.Validate(); err != nil {
			return fmt.Errorf("incomingObservedCategoryBuckets[%d] が有効ではありません: %w", index, err)
		}
		order := judicialPublicationCategoryOrder(bucket.PublicationCategory())
		if order <= lastOrder {
			return fmt.Errorf("incomingObservedCategoryBuckets は規定順でなければなりません")
		}
		lastOrder = order
	}
	return nil
}

func sumJudicialCitationYearBuckets(values []JudicialCitationYearBucket) int {
	total := 0
	for _, bucket := range values {
		total += bucket.Count()
	}
	return total
}

func sumJudicialCitationCategoryBuckets(values []JudicialCitationCategoryBucket) int {
	total := 0
	for _, bucket := range values {
		total += bucket.Count()
	}
	return total
}

func equalJudicialCitationYearBuckets(values []JudicialCitationYearBucket, counts map[int]int) bool {
	years := make([]int, 0, len(counts))
	for year := range counts {
		years = append(years, year)
	}
	sort.Ints(years)
	if len(values) != len(years) {
		return false
	}
	for index, year := range years {
		if values[index].Year() != year || values[index].Count() != counts[year] {
			return false
		}
	}
	return true
}

func equalJudicialCitationCategoryBuckets(
	values []JudicialCitationCategoryBucket,
	counts map[JudicialPublicationCategory]int,
) bool {
	ordered := judicialPublicationCategories()
	expected := make([]JudicialCitationCategoryBucket, 0, len(counts))
	for _, category := range ordered {
		if count := counts[category]; count > 0 {
			expected = append(expected, JudicialCitationCategoryBucket{category, count})
		}
	}
	return slices.Equal(values, expected)
}

func judicialPublicationCategoryOrder(value JudicialPublicationCategory) int {
	return slices.Index(judicialPublicationCategories(), value)
}

func judicialPublicationCategories() []JudicialPublicationCategory {
	return []JudicialPublicationCategory{
		JudicialPublicationCategorySupremeCourt,
		JudicialPublicationCategoryHighCourt,
		JudicialPublicationCategoryLowerCourt,
		JudicialPublicationCategoryAdministrative,
		JudicialPublicationCategoryLabor,
		JudicialPublicationCategoryIntellectualProperty,
	}
}
