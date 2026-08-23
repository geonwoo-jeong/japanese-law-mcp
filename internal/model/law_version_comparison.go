package model

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// LawVersionComparisonScope は、版間比較で採用した法令構造の範囲を表す。
type LawVersionComparisonScope string

const (
	LawVersionComparisonScopeMainAndOriginalSupplementaryArticles LawVersionComparisonScope = "main_and_original_supplementary_articles"
)

// LawVersionComparisonValues は、二版比較結果を構築する値を保持する。
type LawVersionComparisonValues struct {
	LawID              string
	Scope              LawVersionComparisonScope
	Before             LawVersionSnapshot
	After              LawVersionSnapshot
	BeforeArticleCount int
	AfterArticleCount  int
	AddedCount         int
	RemovedCount       int
	ModifiedCount      int
	UnchangedCount     int
	TotalCount         int
	Items              []LawVersionChange
}

// LawVersionComparison は、一つの法令について確定した二版の条差分を表す。
type LawVersionComparison struct {
	lawID              string
	scope              LawVersionComparisonScope
	before             LawVersionSnapshot
	after              LawVersionSnapshot
	beforeArticleCount int
	afterArticleCount  int
	addedCount         int
	removedCount       int
	modifiedCount      int
	unchangedCount     int
	totalCount         int
	items              []LawVersionChange
}

// NewLawVersionComparison は、入力を複製して検証済みの二版比較結果を返す。
func NewLawVersionComparison(values LawVersionComparisonValues) (LawVersionComparison, error) {
	comparison := LawVersionComparison{
		lawID:              values.LawID,
		scope:              values.Scope,
		before:             values.Before,
		after:              values.After,
		beforeArticleCount: values.BeforeArticleCount,
		afterArticleCount:  values.AfterArticleCount,
		addedCount:         values.AddedCount,
		removedCount:       values.RemovedCount,
		modifiedCount:      values.ModifiedCount,
		unchangedCount:     values.UnchangedCount,
		totalCount:         values.TotalCount,
		items:              cloneLawVersionChanges(values.Items),
	}
	if err := comparison.Validate(); err != nil {
		return LawVersionComparison{}, err
	}
	return comparison, nil
}

func (c LawVersionComparison) LawID() string                    { return c.lawID }
func (c LawVersionComparison) Scope() LawVersionComparisonScope { return c.scope }
func (c LawVersionComparison) Before() LawVersionSnapshot       { return c.before }
func (c LawVersionComparison) After() LawVersionSnapshot        { return c.after }
func (c LawVersionComparison) BeforeArticleCount() int          { return c.beforeArticleCount }
func (c LawVersionComparison) AfterArticleCount() int           { return c.afterArticleCount }
func (c LawVersionComparison) AddedCount() int                  { return c.addedCount }
func (c LawVersionComparison) RemovedCount() int                { return c.removedCount }
func (c LawVersionComparison) ModifiedCount() int               { return c.modifiedCount }
func (c LawVersionComparison) UnchangedCount() int              { return c.unchangedCount }
func (c LawVersionComparison) TotalCount() int                  { return c.totalCount }
func (c LawVersionComparison) Items() []LawVersionChange {
	return cloneLawVersionChanges(c.items)
}

// Validate は、出典、件数、変更項目及び二版の同一法令性を検証する。
func (c LawVersionComparison) Validate() error {
	if c.lawID == "" || !utf8.ValidString(c.lawID) {
		return fmt.Errorf("lawId は有効な UTF-8 の必須項目です")
	}
	if c.scope != LawVersionComparisonScopeMainAndOriginalSupplementaryArticles {
		return fmt.Errorf("scope が定義されていません")
	}
	if err := c.before.Validate(); err != nil {
		return fmt.Errorf("before が有効ではありません: %w", err)
	}
	if err := c.after.Validate(); err != nil {
		return fmt.Errorf("after が有効ではありません: %w", err)
	}
	beforeLaw := c.before.Law()
	afterLaw := c.after.Law()
	if beforeLaw.LawID() != c.lawID || afterLaw.LawID() != c.lawID {
		return fmt.Errorf("before と after は lawId と同じ法令でなければなりません")
	}
	if beforeLaw.Source() != afterLaw.Source() {
		return fmt.Errorf("before と after は同じ情報源でなければなりません")
	}
	if c.beforeArticleCount < 0 || c.afterArticleCount < 0 ||
		c.addedCount < 0 || c.removedCount < 0 || c.modifiedCount < 0 ||
		c.unchangedCount < 0 || c.totalCount < 0 {
		return fmt.Errorf("件数は 0 以上でなければなりません")
	}
	if c.beforeArticleCount != c.removedCount+c.modifiedCount+c.unchangedCount {
		return fmt.Errorf("beforeArticleCount の件数式が一致しません")
	}
	if c.afterArticleCount != c.addedCount+c.modifiedCount+c.unchangedCount {
		return fmt.Errorf("afterArticleCount の件数式が一致しません")
	}
	if c.totalCount != c.addedCount+c.removedCount+c.modifiedCount ||
		c.totalCount != len(c.items) {
		return fmt.Errorf("totalCount が変更件数又は items と一致しません")
	}
	return c.validateItems()
}

func (c LawVersionComparison) validateItems() error {
	counts := map[LawVersionChangeKind]int{}
	beforeIdentities := make(map[string]struct{}, len(c.items))
	afterIdentities := make(map[string]struct{}, len(c.items))
	lastAfterOrder := 0
	lastRemovedOrder := 0
	removedStarted := false
	for index, item := range c.items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("items[%d] が有効ではありません: %w", index, err)
		}
		counts[item.ChangeKind()]++
		if before, exists := item.Before(); exists {
			if err := validateComparisonArticleSide(before, c.lawID, c.before); err != nil {
				return fmt.Errorf("items[%d].before が比較前版と一致しません: %w", index, err)
			}
			key := lawVersionArticleIdentityKey(before.Location())
			if _, duplicate := beforeIdentities[key]; duplicate {
				return fmt.Errorf("items[%d].before の条同一性が重複しています", index)
			}
			beforeIdentities[key] = struct{}{}
			if item.ChangeKind() == LawVersionChangeKindRemoved {
				removedStarted = true
				if before.documentOrder <= lastRemovedOrder {
					return fmt.Errorf("removed は比較前版の文書順でなければなりません")
				}
				lastRemovedOrder = before.documentOrder
			}
		}
		if after, exists := item.After(); exists {
			if removedStarted {
				return fmt.Errorf("added と modified は removed より前でなければなりません")
			}
			if err := validateComparisonArticleSide(after, c.lawID, c.after); err != nil {
				return fmt.Errorf("items[%d].after が比較後版と一致しません: %w", index, err)
			}
			key := lawVersionArticleIdentityKey(after.Location())
			if _, duplicate := afterIdentities[key]; duplicate {
				return fmt.Errorf("items[%d].after の条同一性が重複しています", index)
			}
			afterIdentities[key] = struct{}{}
			if after.documentOrder <= lastAfterOrder {
				return fmt.Errorf("added と modified は比較後版の文書順でなければなりません")
			}
			lastAfterOrder = after.documentOrder
		}
	}
	if counts[LawVersionChangeKindAdded] != c.addedCount ||
		counts[LawVersionChangeKindRemoved] != c.removedCount ||
		counts[LawVersionChangeKindModified] != c.modifiedCount {
		return fmt.Errorf("changeKind ごとの件数が一致しません")
	}
	return nil
}

func validateComparisonArticleSide(
	article LawVersionArticle,
	lawID string,
	snapshot LawVersionSnapshot,
) error {
	citation := article.Citation()
	law := snapshot.Law()
	if citation.LawID() != lawID ||
		citation.RevisionID() != law.RevisionID() ||
		citation.Source() != law.Source() {
		return fmt.Errorf("citation が確定版を指していません")
	}
	return nil
}

func lawVersionArticleIdentityKey(location LawVersionArticleLocation) string {
	return string(location.Provision()) + "\x00" + location.ArticleNumber()
}

// MarshalJSON は、SOT-MODEL-033 の項目名で比較結果を表す。
func (c LawVersionComparison) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		LawID              string                    `json:"lawId"`
		Scope              LawVersionComparisonScope `json:"scope"`
		Before             LawVersionSnapshot        `json:"before"`
		After              LawVersionSnapshot        `json:"after"`
		BeforeArticleCount int                       `json:"beforeArticleCount"`
		AfterArticleCount  int                       `json:"afterArticleCount"`
		AddedCount         int                       `json:"addedCount"`
		RemovedCount       int                       `json:"removedCount"`
		ModifiedCount      int                       `json:"modifiedCount"`
		UnchangedCount     int                       `json:"unchangedCount"`
		TotalCount         int                       `json:"totalCount"`
		Items              []LawVersionChange        `json:"items"`
	}{
		LawID:              c.lawID,
		Scope:              c.scope,
		Before:             c.before,
		After:              c.after,
		BeforeArticleCount: c.beforeArticleCount,
		AfterArticleCount:  c.afterArticleCount,
		AddedCount:         c.addedCount,
		RemovedCount:       c.removedCount,
		ModifiedCount:      c.modifiedCount,
		UnchangedCount:     c.unchangedCount,
		TotalCount:         c.totalCount,
		Items:              cloneLawVersionChanges(c.items),
	})
}

func (*LawVersionComparison) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("LawVersionComparison は JSON から直接復元できません。NewLawVersionComparison を使用してください")
}

func cloneLawVersionChanges(values []LawVersionChange) []LawVersionChange {
	cloned := make([]LawVersionChange, len(values))
	copy(cloned, values)
	return cloned
}
