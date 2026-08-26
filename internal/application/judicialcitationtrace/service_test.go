package judicialcitationtrace_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitationnormalize"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitationtrace"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitingcandidatesearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestServiceは詳細から共有情報と両方向を順に組み立てる(t *testing.T) {
	root := mustDecisionDetails(t, decisionDetailsOptions{
		fullText:            true,
		reporterCitation:    "民集80巻1号1頁",
		lowerCourtName:      "東京高等裁判所",
		lowerCourtCase:      "令和5年（ネ）第100号",
		referencedProvision: "民法第177条",
	})
	first := mustDecisionMentionWithReference(
		t,
		"昭和五十年（オ）第一号",
		"昭和50年（オ）第1号",
		"page=1",
	)
	second := mustDecisionMentionWithReference(
		t,
		"昭和50年(オ)1",
		"昭和50年（オ）第1号",
		"page=2",
	)
	extractResult := mustExtractResult(t, []model.JudicialCitationDecisionMention{first, second}, false)
	candidateResult := mustCandidateResult(t, []judicialcitingcandidatesearch.Candidate{
		mustCandidate(t, "candidate-1", "2024-01-02", model.JudicialPublicationCategoryHighCourt),
	})

	calls := []string{}
	reader := &fakeDecisionReader{result: root, calls: &calls}
	extractor := &fakeExtractor{result: extractResult, calls: &calls}
	searcher := &fakeCandidateSearcher{result: candidateResult, calls: &calls}
	service := mustService(t, reader, extractor, searcher, mustLawResolver(t, []lawEntry{{"129AC0000000089", "民法"}}))
	request := mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionBoth, nil)

	result, err := service.Trace(context.Background(), request)
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	if strings.Join(calls, ",") != "detail,extract,candidate" {
		t.Fatalf("calls = %v", calls)
	}
	if result.Status() != model.JudicialCitationResultStatusComplete ||
		result.CoverageNotice() != judicialcitationtrace.CoverageNotice {
		t.Fatalf("result status/notice = %q / %q", result.Status(), result.CoverageNotice())
	}
	graph := result.Graph()
	if len(graph.Edges()) != 4 {
		t.Fatalf("edges = %d", len(graph.Edges()))
	}
	wantRelations := []model.JudicialCitationRelationType{
		model.JudicialCitationRelationTypeHasLowerCourtDecision,
		model.JudicialCitationRelationTypeReferencesLawProvision,
		model.JudicialCitationRelationTypeCitesDecision,
		model.JudicialCitationRelationTypePossibleCitesDecision,
	}
	for index, relation := range wantRelations {
		if graph.Edges()[index].RelationType() != relation {
			t.Fatalf("edges[%d].relationType = %q", index, graph.Edges()[index].RelationType())
		}
	}
	if len(graph.Edges()[2].Evidence()) != 2 {
		t.Fatalf("重複引用の evidence = %d", len(graph.Edges()[2].Evidence()))
	}
	if reference, _ := graph.Nodes()[3].ReferenceText(); reference != "昭和五十年（オ）第一号" {
		t.Fatalf("outgoing referenceText = %q", reference)
	}
	if graph.Summary().ConfirmedOutgoingDecisionCount() != 1 ||
		graph.Summary().IncomingCandidateCount() != 1 ||
		graph.Summary().ReferencedProvisionCount() != 1 ||
		graph.Summary().LowerCourtRelationCount() != 1 {
		t.Fatalf("summary = %#v", graph.Summary())
	}
}

func TestServiceは全文PDFがない場合に外向きを成功縮退する(t *testing.T) {
	root := mustDecisionDetails(t, decisionDetailsOptions{})
	calls := []string{}
	service := mustService(
		t,
		&fakeDecisionReader{result: root, calls: &calls},
		&fakeExtractor{calls: &calls},
		&fakeCandidateSearcher{calls: &calls},
		mustLawResolver(t, nil),
	)
	request := mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionOutgoing, nil)

	result, err := service.Trace(context.Background(), request)
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	if strings.Join(calls, ",") != "detail" {
		t.Fatalf("calls = %v", calls)
	}
	coverage := result.Graph().Coverage()
	if result.Status() != model.JudicialCitationResultStatusPartial ||
		coverage.Outgoing().Status() != model.JudicialCitationDirectionStatusUnavailable ||
		len(result.Issues()) != 1 || result.Issues()[0].Code() != "full_text_document_unavailable" {
		t.Fatalf("result = %#v", result)
	}
}

func TestServiceはtextLayer不在を引用なしにしない(t *testing.T) {
	root := mustDecisionDetails(t, decisionDetailsOptions{fullText: true})
	unavailable, err := judicialcasecitationextract.NewResult(
		judicialcasecitationextract.ResultValues{
			ConfirmedDecisionMentions: []model.JudicialCitationDecisionMention{},
			UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
			DocumentTextStatus: judicialcasecitationextract.
				DocumentTextStatusDocumentTextUnavailable,
		},
	)
	if err != nil {
		t.Fatalf("extract result = %v", err)
	}
	service := mustService(
		t,
		&fakeDecisionReader{result: root},
		&fakeExtractor{result: unavailable},
		&fakeCandidateSearcher{},
		mustLawResolver(t, nil),
	)

	result, err := service.Trace(
		context.Background(),
		mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionOutgoing, nil),
	)
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	if result.Graph().Coverage().Outgoing().Status() != model.JudicialCitationDirectionStatusUnavailable ||
		result.Issues()[0].Code() != "document_text_unavailable" {
		t.Fatalf("result = %#v", result)
	}
}

func TestServiceは片方向失敗を安全なissueにして成功方向を保持する(t *testing.T) {
	root := mustDecisionDetails(t, decisionDetailsOptions{fullText: true})
	service := mustService(
		t,
		&fakeDecisionReader{result: root},
		&fakeExtractor{result: mustExtractResult(t, nil, false)},
		&fakeCandidateSearcher{err: errors.New("秘密の検索語とref")},
		mustLawResolver(t, nil),
	)

	result, err := service.Trace(
		context.Background(),
		mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionBoth, nil),
	)
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	if result.Status() != model.JudicialCitationResultStatusPartial ||
		result.Graph().Coverage().Incoming().Status() != model.JudicialCitationDirectionStatusPartial {
		t.Fatalf("result = %#v", result)
	}
	issues := result.Issues()
	if len(issues) != 1 || strings.Contains(issues[0].Message(), "秘密") ||
		issues[0].Code() != "internal_error" {
		t.Fatalf("issues = %#v", issues)
	}
	attempted, _ := result.Graph().Coverage().Incoming().AttemptedSearches()
	completed, _ := result.Graph().Coverage().Incoming().CompletedSearches()
	if attempted != 1 || completed != 0 {
		t.Fatalf("search coverage = %d/%d", completed, attempted)
	}
}

func TestServiceは全要求方向失敗でgraphを返さない(t *testing.T) {
	root := mustDecisionDetails(t, decisionDetailsOptions{fullText: true})
	service := mustService(
		t,
		&fakeDecisionReader{result: root},
		&fakeExtractor{err: mustTraceSourceError(
			t,
			judicialcasecitationextract.CapabilityID,
			model.SourceErrorCodeSourceTimeout,
		)},
		&fakeCandidateSearcher{err: errors.New("秘密検索語")},
		mustLawResolver(t, nil),
	)

	result, err := service.Trace(
		context.Background(),
		mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionBoth, nil),
	)
	if err == nil || result.Validate() == nil {
		t.Fatalf("result/error = %#v / %v", result, err)
	}
	var operation judicialcitationtrace.OperationError
	if !errors.As(err, &operation) || operation.Code() != model.ErrorCodeSourceTimeout ||
		!operation.Retryable() || operation.Result().Validate() != nil {
		t.Fatalf("operation error = %#v", err)
	}
	if strings.Contains(err.Error(), "秘密") {
		t.Fatalf("error が内部値を含みます: %v", err)
	}
}

func TestServiceは候補一検索失敗をincomingPartialへ変換する(t *testing.T) {
	root := mustDecisionDetails(t, decisionDetailsOptions{
		fullText:         true,
		reporterCitation: "民集80巻1号1頁",
	})
	candidate := mustCandidate(t, "candidate-partial", "2025-03-04", model.JudicialPublicationCategoryLabor)
	service := mustService(
		t,
		&fakeDecisionReader{result: root},
		&fakeExtractor{result: mustExtractResult(t, nil, false)},
		&fakeCandidateSearcher{result: mustCandidatePartialResult(
			t,
			[]judicialcitingcandidatesearch.Candidate{candidate},
		)},
		mustLawResolver(t, nil),
	)

	result, err := service.Trace(
		context.Background(),
		mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionBoth, nil),
	)
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	incoming := result.Graph().Coverage().Incoming()
	attempted, _ := incoming.AttemptedSearches()
	completed, _ := incoming.CompletedSearches()
	if result.Status() != model.JudicialCitationResultStatusPartial ||
		incoming.Status() != model.JudicialCitationDirectionStatusPartial ||
		attempted != 2 || completed != 1 ||
		!hasIssueCode(result.Issues(), "source_unavailable") {
		t.Fatalf("result = %#v", result)
	}
}

func TestServiceは候補上限による正常切詰めだけではpartialにしない(t *testing.T) {
	root := mustDecisionDetails(t, decisionDetailsOptions{})
	candidate := mustCandidate(t, "candidate-truncated", "2023-01-02", model.JudicialPublicationCategoryAdministrative)
	service := mustService(
		t,
		&fakeDecisionReader{result: root},
		&fakeExtractor{},
		&fakeCandidateSearcher{result: mustTruncatedCandidateResult(
			t,
			[]judicialcitingcandidatesearch.Candidate{candidate},
		)},
		mustLawResolver(t, nil),
	)

	result, err := service.Trace(
		context.Background(),
		mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionIncoming, intPointer(1)),
	)
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	if result.Status() != model.JudicialCitationResultStatusComplete ||
		result.Graph().Coverage().Incoming().Status() != model.JudicialCitationDirectionStatusComplete ||
		!result.Graph().Coverage().Incoming().Truncated() || len(result.Issues()) != 0 {
		t.Fatalf("result = %#v", result)
	}
}

func TestServiceは依存関係と詳細失敗を安全に検証する(t *testing.T) {
	resolver := mustLawResolver(t, nil)
	var nilReader *fakeDecisionReader
	if _, err := judicialcitationtrace.NewService(
		nilReader,
		&fakeExtractor{},
		&fakeCandidateSearcher{},
		resolver,
	); err == nil {
		t.Fatal("nil reader を受理しました")
	}
	if _, err := judicialcitationtrace.NewService(
		&fakeDecisionReader{},
		(*fakeExtractor)(nil),
		&fakeCandidateSearcher{},
		resolver,
	); err == nil {
		t.Fatal("nil extractor を受理しました")
	}
	if _, err := judicialcitationtrace.NewService(
		&fakeDecisionReader{},
		&fakeExtractor{},
		(*fakeCandidateSearcher)(nil),
		resolver,
	); err == nil {
		t.Fatal("nil searcher を受理しました")
	}

	root := mustDecisionDetails(t, decisionDetailsOptions{})
	calls := []string{}
	service := mustService(
		t,
		&fakeDecisionReader{err: judicialdecisionread.ErrNotFound, calls: &calls},
		&fakeExtractor{calls: &calls},
		&fakeCandidateSearcher{calls: &calls},
		resolver,
	)
	_, err := service.Trace(
		context.Background(),
		mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionOutgoing, nil),
	)
	var operation judicialcitationtrace.OperationError
	if !errors.As(err, &operation) || operation.Code() != model.ErrorCodeNotFound ||
		strings.Join(calls, ",") != "detail" {
		t.Fatalf("error/calls = %#v / %v", err, calls)
	}
	if _, err := service.Trace(nil, mustTraceRequest(
		t,
		root.Ref(),
		model.JudicialCitationRequestedDirectionOutgoing,
		nil,
	)); err == nil {
		t.Fatal("nil context を受理しました")
	}
}

func TestServiceは取消後に後続方向を開始しない(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	root := mustDecisionDetails(t, decisionDetailsOptions{fullText: true})
	calls := []string{}
	service := mustService(
		t,
		&fakeDecisionReader{result: root, calls: &calls, after: cancel},
		&fakeExtractor{calls: &calls},
		&fakeCandidateSearcher{calls: &calls},
		mustLawResolver(t, nil),
	)

	_, err := service.Trace(
		ctx,
		mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionBoth, nil),
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	if strings.Join(calls, ",") != "detail" {
		t.Fatalf("calls = %v", calls)
	}
}

func TestServiceはPDF完了直後の取消で候補検索を開始しない(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	root := mustDecisionDetails(t, decisionDetailsOptions{fullText: true})
	calls := []string{}
	service := mustService(
		t,
		&fakeDecisionReader{result: root, calls: &calls},
		&fakeExtractor{
			result: mustExtractResult(t, nil, false),
			calls:  &calls,
			after:  cancel,
		},
		&fakeCandidateSearcher{calls: &calls},
		mustLawResolver(t, nil),
	)

	_, err := service.Trace(
		ctx,
		mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionBoth, nil),
	)
	if !errors.Is(err, context.Canceled) || strings.Join(calls, ",") != "detail,extract" {
		t.Fatalf("error/calls = %v / %v", err, calls)
	}
}

func TestServiceは判例edge上限で決定的先頭を保持する(t *testing.T) {
	root := mustDecisionDetails(t, decisionDetailsOptions{fullText: true})
	mentions := make([]model.JudicialCitationDecisionMention, 65)
	for index := range mentions {
		identity := fmt.Sprintf("令和1年（受）第%d号", index+1)
		mentions[index] = mustDecisionMention(t, identity, fmt.Sprintf("page=%d", index+1))
	}
	service := mustService(
		t,
		&fakeDecisionReader{result: root},
		&fakeExtractor{result: mustExtractResult(t, mentions, false)},
		&fakeCandidateSearcher{},
		mustLawResolver(t, nil),
	)

	result, err := service.Trace(
		context.Background(),
		mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionOutgoing, nil),
	)
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	if result.Status() != model.JudicialCitationResultStatusPartial ||
		result.Graph().Coverage().Outgoing().Status() != model.JudicialCitationDirectionStatusPartial ||
		len(result.Graph().Edges()) != 64 || !result.Graph().Coverage().Outgoing().Truncated() ||
		!hasIssueCode(result.Issues(), "judicial_decision_edge_limit_reached") {
		t.Fatalf("edges/coverage/issues = %d / %#v / %#v", len(result.Graph().Edges()), result.Graph().Coverage(), result.Issues())
	}
	lastNode := result.Graph().Nodes()[64]
	if reference, _ := lastNode.ReferenceText(); reference != "令和1年（受）第64号" {
		t.Fatalf("last retained node = %q", reference)
	}
}

func TestServiceは外向き後の全判例edge上限をincomingにも適用する(t *testing.T) {
	root := mustDecisionDetails(t, decisionDetailsOptions{
		fullText:         true,
		reporterCitation: "民集80巻1号1頁",
	})
	mentions := make([]model.JudicialCitationDecisionMention, 60)
	for index := range mentions {
		identity := fmt.Sprintf("令和2年（受）第%d号", index+1)
		mentions[index] = mustDecisionMention(t, identity, "page=1")
	}
	candidates := make([]judicialcitingcandidatesearch.Candidate, 5)
	for index := range candidates {
		candidates[index] = mustCandidate(
			t,
			fmt.Sprintf("candidate-cap-%d", index+1),
			"2024-01-02",
			model.JudicialPublicationCategoryHighCourt,
		)
	}
	service := mustService(
		t,
		&fakeDecisionReader{result: root},
		&fakeExtractor{result: mustExtractResult(t, mentions, false)},
		&fakeCandidateSearcher{result: mustCandidateResult(t, candidates)},
		mustLawResolver(t, nil),
	)

	result, err := service.Trace(
		context.Background(),
		mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionBoth, nil),
	)
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	if len(result.Graph().Edges()) != 64 ||
		result.Graph().Summary().IncomingCandidateCount() != 4 ||
		result.Graph().Coverage().Incoming().Status() != model.JudicialCitationDirectionStatusPartial ||
		!result.Graph().Coverage().Incoming().Truncated() ||
		!hasIssueCode(result.Issues(), "judicial_decision_edge_limit_reached") {
		t.Fatalf("result = %#v", result)
	}
}

func TestServiceはPDF出現上限を外向きpartialにする(t *testing.T) {
	root := mustDecisionDetails(t, decisionDetailsOptions{fullText: true})
	mentions := make([]model.JudicialCitationDecisionMention, 256)
	for index := range mentions {
		mentions[index] = mustDecisionMention(t, "令和1年（受）第1号", "page=1")
	}
	service := mustService(
		t,
		&fakeDecisionReader{result: root},
		&fakeExtractor{result: mustExtractResult(t, mentions, true)},
		&fakeCandidateSearcher{},
		mustLawResolver(t, nil),
	)

	result, err := service.Trace(
		context.Background(),
		mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionOutgoing, nil),
	)
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	if result.Graph().Coverage().Outgoing().Status() != model.JudicialCitationDirectionStatusPartial ||
		!hasIssueCode(result.Issues(), "citation_occurrence_limit_reached") {
		t.Fatalf("result = %#v", result)
	}
}

func TestServiceは法条上限をsharedだけのpartialとして示す(t *testing.T) {
	entries := make([]lawEntry, 33)
	segments := make([]string, 33)
	for index := range entries {
		entries[index] = lawEntry{
			id:    fmt.Sprintf("law-%d", index+1),
			title: fmt.Sprintf("試験法律%d", index+1),
		}
		segments[index] = entries[index].title + "第1条"
	}
	segments[0] = entries[0].title + "附則第1条第2項"
	root := mustDecisionDetails(t, decisionDetailsOptions{
		fullText:            true,
		referencedProvision: strings.Join(segments, "、"),
	})
	service := mustService(
		t,
		&fakeDecisionReader{result: root},
		&fakeExtractor{result: mustExtractResult(t, nil, false)},
		&fakeCandidateSearcher{},
		mustLawResolver(t, entries),
	)

	result, err := service.Trace(
		context.Background(),
		mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionOutgoing, nil),
	)
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	if result.Status() != model.JudicialCitationResultStatusPartial ||
		result.Graph().Coverage().Outgoing().Status() != model.JudicialCitationDirectionStatusComplete ||
		result.Graph().Coverage().Outgoing().Truncated() ||
		result.Graph().Summary().ReferencedProvisionCount() != 32 ||
		!hasIssueCode(result.Issues(), "law_provision_edge_limit_reached") {
		t.Fatalf("result = %#v", result)
	}
}

func TestServiceは不完全な原審情報を未解決言及に保つ(t *testing.T) {
	root := mustDecisionDetails(t, decisionDetailsOptions{
		fullText:       true,
		lowerCourtName: "東京高等裁判所",
	})
	service := mustService(
		t,
		&fakeDecisionReader{result: root},
		&fakeExtractor{result: mustExtractResult(t, nil, false)},
		&fakeCandidateSearcher{},
		mustLawResolver(t, nil),
	)

	result, err := service.Trace(
		context.Background(),
		mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionOutgoing, nil),
	)
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	mentions := result.Graph().UnresolvedMentions()
	if len(mentions) != 1 || mentions[0].Reason() != model.JudicialCitationUnresolvedReasonInsufficientIdentity {
		t.Fatalf("mentions = %#v", mentions)
	}
}

func TestServiceは共有正規化失敗をsharedIssueにする(t *testing.T) {
	root := mustDecisionDetails(t, decisionDetailsOptions{
		fullText:            true,
		lowerCourtName:      "東京高等裁判所",
		lowerCourtCase:      "令和5年（ネ）第100号 補足",
		referencedProvision: "民法第1条",
	})
	service := mustService(
		t,
		&fakeDecisionReader{result: root},
		&fakeExtractor{result: mustExtractResult(t, nil, false)},
		&fakeCandidateSearcher{},
		judicialcitationnormalize.ExactLawAliasResolver{},
	)

	result, err := service.Trace(
		context.Background(),
		mustTraceRequest(t, root.Ref(), model.JudicialCitationRequestedDirectionOutgoing, nil),
	)
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	if result.Status() != model.JudicialCitationResultStatusPartial ||
		!hasIssueCode(result.Issues(), "lower_court_metadata_unresolved") ||
		!hasIssueCode(result.Issues(), "law_reference_resolution_failed") {
		t.Fatalf("issues = %#v", result.Issues())
	}
}

func hasIssueCode(issues []model.JudicialCitationIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code() == code {
			return true
		}
	}
	return false
}

type fakeDecisionReader struct {
	result model.SourcedResource[model.JudicialDecisionDetails]
	err    error
	calls  *[]string
	after  func()
}

func (f *fakeDecisionReader) Read(
	_ context.Context,
	_ judicialdecisionread.Request,
) (model.SourcedResource[model.JudicialDecisionDetails], error) {
	if f.calls != nil {
		*f.calls = append(*f.calls, "detail")
	}
	if f.after != nil {
		f.after()
	}
	return f.result, f.err
}

type fakeExtractor struct {
	result judicialcasecitationextract.Result
	err    error
	calls  *[]string
	after  func()
}

func (f *fakeExtractor) Extract(
	_ context.Context,
	_ judicialcasecitationextract.Request,
) (judicialcasecitationextract.Result, error) {
	if f.calls != nil {
		*f.calls = append(*f.calls, "extract")
	}
	if f.after != nil {
		f.after()
	}
	return f.result, f.err
}

type fakeCandidateSearcher struct {
	result judicialcitingcandidatesearch.Result
	err    error
	calls  *[]string
}

func (f *fakeCandidateSearcher) Search(
	_ context.Context,
	_ judicialcitingcandidatesearch.Request,
) (judicialcitingcandidatesearch.Result, error) {
	if f.calls != nil {
		*f.calls = append(*f.calls, "candidate")
	}
	return f.result, f.err
}
