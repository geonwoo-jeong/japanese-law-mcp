package hanreipdf

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestScanExtractedTextExtractsConfirmedAndUnresolvedDecisionMentions(t *testing.T) {
	t.Parallel()

	occurrences, truncated := scanExtractedText(
		"令和6(受)123 を引用し、平成99(受)1 も参照した。",
		1,
		maximumOccurrences,
	)
	if truncated {
		t.Fatal("出現上限未満なのに truncated になりました")
	}
	if len(occurrences) != 2 {
		t.Fatalf("occurrence 数 = %d", len(occurrences))
	}
	if !occurrences[0].confirmed() || occurrences[0].DecisionIdentity != "令和6(受)123" {
		t.Fatalf("confirmed occurrence = %#v", occurrences[0])
	}
	if occurrences[1].confirmed() ||
		occurrences[1].Reason != model.JudicialCitationUnresolvedReasonInsufficientIdentity ||
		occurrences[1].ReferenceText != "平成99(受)1" {
		t.Fatalf("unresolved occurrence = %#v", occurrences[1])
	}
}

func TestBuildExtractResultMapsWorkerOutputToCapabilityResult(t *testing.T) {
	t.Parallel()

	decision, document := judicialcasecitationextracttestRequest(t)
	request, err := judicialcasecitationextract.NewRequest(
		judicialcasecitationextract.RequestValues{
			Decision: decision,
			Document: document,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := buildExtractResult(
		request,
		time.Date(2026, 8, 26, 13, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
		[]byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n%%EOF"),
		workerOutput{
			PageCount:         2,
			ObjectCount:       5,
			DecompressedBytes: 128,
			Occurrences: []workerMention{
				{
					Page:             1,
					ReferenceText:    "令和5(受)42",
					DecisionIdentity: "令和5(受)42",
					Excerpt:          "先行判例として令和5(受)42を引用する。",
				},
				{
					Page:          2,
					ReferenceText: "平成99(受)1",
					Excerpt:       "平成99(受)1も参照した。",
					Reason:        model.JudicialCitationUnresolvedReasonInsufficientIdentity,
				},
			},
			TextUnavailable: false,
			Truncated:       false,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.DocumentTextStatus() != judicialcasecitationextract.DocumentTextStatusAvailable {
		t.Fatalf("documentTextStatus = %q", result.DocumentTextStatus())
	}
	if result.ExaminedPageCount() != 2 || result.OccurrenceCount() != 2 {
		t.Fatalf("page=%d occurrence=%d", result.ExaminedPageCount(), result.OccurrenceCount())
	}
	if len(result.ConfirmedDecisionMentions()) != 1 ||
		result.ConfirmedDecisionMentions()[0].DecisionIdentityText() != "令和5(受)42" {
		t.Fatalf("confirmed = %#v", result.ConfirmedDecisionMentions())
	}
	if len(result.UnresolvedMentions()) != 1 ||
		result.UnresolvedMentions()[0].MentionText() != "平成99(受)1" {
		t.Fatalf("unresolved = %#v", result.UnresolvedMentions())
	}
}

func TestBuildExtractResultAcceptsStrictCourtDateAndReporterIdentities(t *testing.T) {
	t.Parallel()

	request := mustExtractRequest(t)
	result, err := buildExtractResult(
		request,
		time.Date(2026, 8, 26, 13, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
		[]byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n%%EOF"),
		workerOutput{
			PageCount:         2,
			ObjectCount:       5,
			DecompressedBytes: 128,
			Occurrences: []workerMention{
				{
					Page:             1,
					ReferenceText:    "民集59巻7号2087頁",
					DecisionIdentity: "民集59巻7号2087頁",
					Excerpt:          "民集59巻7号2087頁を引用する。",
				},
				{
					Page:             2,
					ReferenceText:    "最高裁判所平成17年9月14日大法廷判決",
					DecisionIdentity: "最高裁判所平成17年9月14日大法廷判決",
					Excerpt:          "最高裁判所平成17年9月14日大法廷判決を参照する。",
				},
			},
			TextUnavailable: false,
			Truncated:       false,
		},
	)
	if err == nil {
		if got := result.ConfirmedDecisionMentions(); len(got) != 2 {
			t.Fatalf("confirmed=%#v", got)
		}
		return
	}
	t.Fatalf("strict identity を受理できていません: %v", err)
}

func TestBuildExtractResultFiltersStrictReporterAndCourtDateSelfReferences(t *testing.T) {
	t.Parallel()

	reporterCitation := "民集第59巻7号2087頁"
	divisionName := "大法廷"
	decisionType := "判決"
	decision, document := judicialcasecitationextracttestRequestWithOptions(
		t,
		extractRequestFixtureOptions{
			sourceID:         sourceID,
			documentURL:      "https://www.courts.go.jp/assets/hanrei/00001.pdf",
			reporterCitation: &reporterCitation,
			divisionName:     &divisionName,
			decisionType:     &decisionType,
		},
	)
	request, err := judicialcasecitationextract.NewRequest(
		judicialcasecitationextract.RequestValues{Decision: decision, Document: document},
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := buildExtractResult(
		request,
		mustRetrievedAt(t),
		[]byte("%PDF-test"),
		workerOutput{
			PageCount:         1,
			ObjectCount:       5,
			DecompressedBytes: 10,
			Occurrences: []workerMention{
				{
					Page:             1,
					ReferenceText:    "民集59巻7号2087頁",
					DecisionIdentity: "民集59巻7号2087頁",
					Excerpt:          "民集59巻7号2087頁",
				},
				{
					Page:             1,
					ReferenceText:    "最高裁判所令和7年1月2日大法廷判決",
					DecisionIdentity: "最高裁判所令和7年1月2日大法廷判決",
					Excerpt:          "最高裁判所令和7年1月2日大法廷判決",
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.OccurrenceCount() != 0 ||
		len(result.ConfirmedDecisionMentions()) != 0 ||
		len(result.UnresolvedMentions()) != 0 {
		t.Fatalf("self references が残りました: %#v", result)
	}
}

func TestBuildExtractResultTreatsTextUnavailableAsSuccessfulDegradation(t *testing.T) {
	t.Parallel()

	decision, document := judicialcasecitationextracttestRequest(t)
	request, err := judicialcasecitationextract.NewRequest(
		judicialcasecitationextract.RequestValues{
			Decision: decision,
			Document: document,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := buildExtractResult(
		request,
		time.Date(2026, 8, 26, 13, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
		[]byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n%%EOF"),
		workerOutput{
			PageCount:         3,
			ObjectCount:       7,
			DecompressedBytes: 256,
			Occurrences:       []workerMention{},
			TextUnavailable:   true,
			Truncated:         false,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.DocumentTextStatus() != judicialcasecitationextract.DocumentTextStatusDocumentTextUnavailable {
		t.Fatalf("documentTextStatus = %q", result.DocumentTextStatus())
	}
	if result.OccurrenceCount() != 0 ||
		len(result.ConfirmedDecisionMentions()) != 0 ||
		len(result.UnresolvedMentions()) != 0 ||
		result.Truncated() {
		t.Fatalf("result = %#v", result)
	}
}

func TestBuildExtractResultFiltersSelfReferenceOccurrence(t *testing.T) {
	t.Parallel()

	result, err := buildExtractResult(
		mustExtractRequest(t),
		mustRetrievedAt(t),
		[]byte("%PDF-test"),
		workerOutput{
			PageCount:         1,
			ObjectCount:       5,
			DecompressedBytes: 10,
			Occurrences: []workerMention{{
				Page:             1,
				ReferenceText:    "令和6(受)123",
				DecisionIdentity: "令和6(受)123",
				Excerpt:          "令和6(受)123",
			}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.OccurrenceCount() != 0 ||
		len(result.ConfirmedDecisionMentions()) != 0 ||
		len(result.UnresolvedMentions()) != 0 ||
		result.Truncated() {
		t.Fatalf("result=%#v", result)
	}
}

func TestBuildExtractResultFailsClosedWhenSelfFilteringWouldBreakTruncationContract(t *testing.T) {
	t.Parallel()

	occurrences := make([]workerMention, maximumOccurrences)
	for index := range occurrences {
		identity := "平成30(受)10"
		if index == 0 {
			identity = "令和6(受)123"
		}
		occurrences[index] = workerMention{
			Page:             1,
			ReferenceText:    identity,
			DecisionIdentity: identity,
			Excerpt:          identity,
		}
	}
	_, err := buildExtractResult(
		mustExtractRequest(t),
		mustRetrievedAt(t),
		[]byte("%PDF-test"),
		workerOutput{
			PageCount:         1,
			ObjectCount:       5,
			DecompressedBytes: 10,
			Occurrences:       occurrences,
			Truncated:         true,
		},
	)
	assertHanreiPDFSourceErrorCode(t, err, model.SourceErrorCodeSourceProcessingLimit)
}

func TestUTF8ExcerptHelpersNeverSplitCharacters(t *testing.T) {
	t.Parallel()

	value := strings.Repeat("あ", 100)
	prefix := utf8Prefix(value, 10)
	suffix := utf8Suffix(value, 10)
	if !utf8.ValidString(prefix) || !utf8.ValidString(suffix) || len(prefix) > 10 || len(suffix) > 10 {
		t.Fatalf("prefix=%q suffix=%q", prefix, suffix)
	}
	if utf8Prefix("短い", 100) != "短い" || utf8Suffix("短い", 100) != "短い" {
		t.Fatal("上限未満の文字列を変更しました")
	}
	longReference := strings.Repeat("あ", 100)
	excerpt := excerptAround(longReference, 0, len(longReference))
	if !utf8.ValidString(excerpt) || len(excerpt) > maximumExcerptBytes {
		t.Fatalf("excerpt bytes=%d", len(excerpt))
	}
}

func TestScanExtractedTextSkipsEraWordWithoutCaseShape(t *testing.T) {
	t.Parallel()

	mentions, truncated := scanExtractedText("令和時代の説明", 1, maximumOccurrences)
	if len(mentions) != 0 || truncated {
		t.Fatalf("mentions=%#v truncated=%t", mentions, truncated)
	}
}

func TestScanExtractedTextExtractsStrictReferenceFormsInSourceOrder(t *testing.T) {
	t.Parallel()

	mentions, truncated := scanExtractedText(
		"民集59巻7号2087頁、最高裁判所平成17年9月14日大法廷判決、令和6(受)123。",
		2,
		maximumOccurrences,
	)
	if truncated || len(mentions) != 3 {
		t.Fatalf("mentions=%#v truncated=%t", mentions, truncated)
	}
	wantReferences := []string{
		"民集59巻7号2087頁",
		"最高裁判所平成17年9月14日大法廷判決",
		"令和6(受)123",
	}
	for index, want := range wantReferences {
		if mentions[index].ReferenceText != want || mentions[index].Page != 2 {
			t.Fatalf("mentions[%d]=%#v want=%q", index, mentions[index], want)
		}
	}
	wantIdentities := []string{
		"民集59巻7号2087頁",
		"最高裁判所平成17年9月14日大法廷判決",
		"令和6(受)123",
	}
	for index, want := range wantIdentities {
		if !mentions[index].confirmed() || mentions[index].DecisionIdentity != want {
			t.Fatalf("mentions[%d]=%#v wantIdentity=%q", index, mentions[index], want)
		}
	}
}

func TestScanExtractedTextDoesNotPromoteLooseCourtDateOrReporterText(t *testing.T) {
	t.Parallel()

	mentions, truncated := scanExtractedText(
		"最高裁判所は平成17年9月14日に説明した。民集の59巻と7号を参照する。",
		1,
		maximumOccurrences,
	)
	if len(mentions) != 0 || truncated {
		t.Fatalf("mentions=%#v truncated=%t", mentions, truncated)
	}
}

func TestScanExtractedTextPreservesDuplicateStrictReporterOccurrences(t *testing.T) {
	t.Parallel()

	mentions, truncated := scanExtractedText(
		"民集59巻7号2087頁を引用し、再び民集第59巻第7号2087頁を引用する。",
		1,
		maximumOccurrences,
	)
	if truncated || len(mentions) != 2 {
		t.Fatalf("mentions=%#v truncated=%t", mentions, truncated)
	}
	for index, mention := range mentions {
		if !mention.confirmed() || mention.DecisionIdentity != "民集59巻7号2087頁" {
			t.Fatalf("mentions[%d]=%#v", index, mention)
		}
	}
}
