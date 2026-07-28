package querypreprocess_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/nlp/kagome"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

const (
	civilLawID       = "129AC0000000089"
	civilRevisionID  = "129AC0000000089_20260624_508AC0000000045"
	civilLawNumber   = "明治二十九年法律第八十九号"
	penalLawID       = "140AC0000000045"
	penalRevisionID  = "140AC0000000045_20260521_507AC0000000039"
	penalLawNumber   = "明治四十年法律第四十五号"
	laborLawID       = "419AC0000000128"
	laborRevisionID  = "419AC0000000128_20200401_430AC0000000071"
	laborLawNumber   = "平成十九年法律第百二十八号"
	testCueProfileID = "test-core"
)

func testLawEntries() []lawnamelexicon.Entry {
	return []lawnamelexicon.Entry{
		{
			ResourceID: civilLawID,
			RevisionID: civilRevisionID,
			LawNumber:  civilLawNumber,
			Canonical:  "民法",
			Terms:      []string{"みんぽう"},
		},
		{
			ResourceID: penalLawID,
			RevisionID: penalRevisionID,
			LawNumber:  penalLawNumber,
			Canonical:  "刑法",
			Terms:      []string{"けいほう"},
		},
		{
			ResourceID: laborLawID,
			RevisionID: laborRevisionID,
			LawNumber:  laborLawNumber,
			Canonical:  "労働契約法",
			Terms:      []string{"労契法", "ろうどうけいやくほう"},
		},
	}
}

func testConceptEntries() []legalconceptlexicon.Entry {
	return []legalconceptlexicon.Entry{
		{
			ConceptID:       "permanent-residence",
			Canonical:       "永住権",
			Terms:           []string{"永住権"},
			ComparisonTerms: []string{"永住権"},
			SourceName:      "試験用の公的情報源",
			SourceURL:       "https://example.go.jp/permanent-residence",
			ConfirmedAt:     "2026-07-28",
			MappingNote:     "試験用の法概念対応",
			SelectionPolicy: legalconceptlexicon.SelectionPolicySingleCandidate,
			Candidates: []legalconceptlexicon.Candidate{
				{
					Task:         legalquery.TaskSearch,
					Resource:     legalquery.ResourceLawProvision,
					InputKind:    legalquery.InputKindLawContentSearch,
					OfficialTerm: "永住許可",
				},
			},
		},
		{
			ConceptID:       "revocation-action",
			Canonical:       "取消訴訟",
			Terms:           []string{"取消訴訟"},
			ComparisonTerms: []string{"取消訴訟"},
			SourceName:      "試験用の公的情報源",
			SourceURL:       "https://example.go.jp/revocation-action",
			ConfirmedAt:     "2026-07-28",
			MappingNote:     "試験用の法概念対応",
			SelectionPolicy: legalconceptlexicon.SelectionPolicySingleCandidate,
			Candidates: []legalconceptlexicon.Candidate{
				{
					Task:         legalquery.TaskSearch,
					Resource:     legalquery.ResourceLawProvision,
					InputKind:    legalquery.InputKindLawContentSearch,
					OfficialTerm: "取消訴訟",
				},
			},
		},
	}
}

func testCueEntries() []legalquery.CueVocabularyEntry {
	return []legalquery.CueVocabularyEntry{
		{
			ProfileID: testCueProfileID,
			CueID:     "task-read",
			Terms:     []string{"読む", "読んで"},
		},
		{
			ProfileID: testCueProfileID,
			CueID:     "task-search",
			Terms:     []string{"検索", "探して"},
		},
	}
}

func mustNewPreprocessor(
	t *testing.T,
	laws []lawnamelexicon.Entry,
	concepts []legalconceptlexicon.Entry,
	cues []legalquery.CueVocabularyEntry,
) legalquery.QueryPreprocessor {
	t.Helper()

	terms := make([]string, 0)
	for _, law := range laws {
		terms = append(terms, law.Canonical)
		terms = append(terms, law.Terms...)
	}
	for _, concept := range concepts {
		terms = append(terms, concept.Terms...)
	}
	for _, cue := range cues {
		terms = append(terms, cue.Terms...)
	}
	slices.Sort(terms)
	terms = slices.Compact(terms)
	if len(terms) == 0 {
		terms = []string{"試験"}
	}

	analyzer, err := kagome.NewAnalyzer(terms)
	if err != nil {
		t.Fatalf("試験用の形態素解析器を作成できません: %v", err)
	}
	preprocessor, err := querypreprocess.New(querypreprocess.Values{
		Analyzer:      analyzer,
		LawNames:      laws,
		LegalConcepts: concepts,
		Cues:          cues,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-025: 前処理器を作成できません: %v", err)
	}
	return preprocessor
}

func mustDefaultPreprocessor(t *testing.T) legalquery.QueryPreprocessor {
	t.Helper()

	return mustNewPreprocessor(
		t,
		testLawEntries(),
		testConceptEntries(),
		testCueEntries(),
	)
}

func mustRequest(t *testing.T, query string) legalquery.Request {
	t.Helper()

	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: query,
	})
	if err != nil {
		t.Fatalf("試験用の照会を作成できません: %v", err)
	}
	return request
}

func mustRequestWithRef(
	t *testing.T,
	query string,
	ref model.SourceResourceRef,
) legalquery.Request {
	t.Helper()

	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: query,
		Ref:   &ref,
	})
	if err != nil {
		t.Fatalf("試験用の参照付き照会を作成できません: %v", err)
	}
	return request
}

func mustLawRef(t *testing.T) model.SourceResourceRef {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "egov",
		ResourceType: "law",
		ResourceID:   civilLawID,
		VersionID:    civilRevisionID,
	})
	if err != nil {
		t.Fatalf("試験用の資源キーを作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "egov-v2",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用の資源参照を作成できません: %v", err)
	}
	return ref
}

func assertSpan(
	t *testing.T,
	query string,
	span legalquery.QuerySpan,
	surface string,
	startByte int,
) {
	t.Helper()

	endByte := startByte + len(surface)
	if span.StartByte() != startByte || span.EndByte() != endByte {
		t.Fatalf(
			"SOT-MODEL-025: %q の span = [%d,%d), want [%d,%d)",
			surface,
			span.StartByte(),
			span.EndByte(),
			startByte,
			endByte,
		)
	}
	if query[span.StartByte():span.EndByte()] != surface {
		t.Fatalf(
			"SOT-MODEL-025: span の原文 = %q, want %q",
			query[span.StartByte():span.EndByte()],
			surface,
		)
	}
}

type preprocessSnapshot struct {
	query       string
	comparison  string
	laws        []string
	concepts    []string
	cues        []string
	identifiers []string
	dates       []string
	articles    []string
	paragraphs  []string
	queryTerms  []string
}

func snapshotResult(result legalquery.PreprocessResult) preprocessSnapshot {
	snapshot := preprocessSnapshot{
		query:      result.Query(),
		comparison: result.ComparisonKey(),
	}
	for _, mention := range result.LawNameMentions() {
		snapshot.laws = append(snapshot.laws, fmt.Sprintf(
			"%d:%d:%s:%s:%s",
			mention.Span().StartByte(),
			mention.Span().EndByte(),
			mention.LawID(),
			mention.Canonical(),
			mention.MatchKind(),
		))
	}
	for _, mention := range result.LegalConceptMentions() {
		snapshot.concepts = append(snapshot.concepts, fmt.Sprintf(
			"%d:%d:%s:%s",
			mention.Span().StartByte(),
			mention.Span().EndByte(),
			mention.ConceptID(),
			mention.MatchKind(),
		))
	}
	for _, mention := range result.CueMentions() {
		snapshot.cues = append(snapshot.cues, fmt.Sprintf(
			"%d:%d:%s:%s:%s",
			mention.Span().StartByte(),
			mention.Span().EndByte(),
			mention.ProfileID(),
			mention.CueID(),
			mention.MatchKind(),
		))
	}
	for _, mention := range result.IdentifierMentions() {
		snapshot.identifiers = append(snapshot.identifiers, fmt.Sprintf(
			"%d:%d:%s:%s",
			mention.Span().StartByte(),
			mention.Span().EndByte(),
			mention.Kind(),
			mention.LawID(),
		))
	}
	for _, mention := range result.DateMentions() {
		snapshot.dates = append(snapshot.dates, fmt.Sprintf(
			"%d:%d:%s",
			mention.Span().StartByte(),
			mention.Span().EndByte(),
			mention.Date().String(),
		))
	}
	for _, mention := range result.ArticleMentions() {
		snapshot.articles = append(snapshot.articles, fmt.Sprintf(
			"%d:%d:%s:%s",
			mention.Span().StartByte(),
			mention.Span().EndByte(),
			mention.Provision(),
			mention.ArticleNumber(),
		))
	}
	for _, mention := range result.ParagraphMentions() {
		snapshot.paragraphs = append(snapshot.paragraphs, fmt.Sprintf(
			"%d:%d:%d",
			mention.Span().StartByte(),
			mention.Span().EndByte(),
			mention.ParagraphNumber(),
		))
	}
	for _, mention := range result.QueryTermMentions() {
		snapshot.queryTerms = append(snapshot.queryTerms, fmt.Sprintf(
			"%d:%d:%s:%s",
			mention.Span().StartByte(),
			mention.Span().EndByte(),
			mention.Kind(),
			mention.Surface(),
		))
	}
	return snapshot
}
