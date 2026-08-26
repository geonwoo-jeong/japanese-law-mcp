package judicialcitationtrace

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitation"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitationnormalize"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitingcandidatesearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// CoverageNotice は、引用 graph の観測範囲と非評価性を示す固定注意文である。
const CoverageNotice = "裁判所の裁判例検索には、すべての判決等が掲載されているわけではありません。引用関係と件数は、取得して解析できた公表資料で確認した明示的な参照又は検索候補に限られます。被引用候補数は、現在の公式検索範囲で観測した候補数であり、実際の全引用回数ではありません。結果がないことは引用の不存在を示さず、先例性、拘束力、確定性、評価、判例変更又は現在の有効性を判断するものではありません。"

// Port は、1-hop 判例引用 graph を返すアプリケーション境界である。
type Port interface {
	Trace(context.Context, Request) (model.JudicialCitationGraphResult, error)
}

// Service は、詳細、PDF 抽出および候補検索を request scope で組み立てる。
type Service struct {
	detailReader      judicialdecisionread.Port
	citationExtractor judicialcasecitationextract.Port
	candidateSearcher judicialcitingcandidatesearch.Port
	lawAliasResolver  judicialcitationnormalize.ExactLawAliasResolver
}

var _ Port = (*Service)(nil)

// NewService は、三つの capability port と法令別名 resolver を結び付ける。
func NewService(
	detailReader judicialdecisionread.Port,
	citationExtractor judicialcasecitationextract.Port,
	candidateSearcher judicialcitingcandidatesearch.Port,
	lawAliasResolver judicialcitationnormalize.ExactLawAliasResolver,
) (*Service, error) {
	if isNilDependency(detailReader) {
		return nil, fmt.Errorf("judicial-decision.read port は必須です")
	}
	if isNilDependency(citationExtractor) {
		return nil, fmt.Errorf("judicial-decision.case-citation.extract port は必須です")
	}
	if isNilDependency(candidateSearcher) {
		return nil, fmt.Errorf("judicial-decision.citing-candidate.search port は必須です")
	}
	return &Service{
		detailReader:      detailReader,
		citationExtractor: citationExtractor,
		candidateSearcher: candidateSearcher,
		lawAliasResolver:  lawAliasResolver,
	}, nil
}

// Trace は、詳細一回と要求方向だけを順番に実行して一時 graph を返す。
func (s *Service) Trace(
	ctx context.Context,
	request Request,
) (model.JudicialCitationGraphResult, error) {
	if ctx == nil {
		return model.JudicialCitationGraphResult{}, newOperationError(errors.New("context required"))
	}
	if err := request.Validate(); err != nil {
		return model.JudicialCitationGraphResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return model.JudicialCitationGraphResult{}, err
	}

	detailRequest, err := judicialdecisionread.NewRequest(judicialdecisionread.RequestValues{
		Ref: request.Ref(),
	})
	if err != nil {
		return model.JudicialCitationGraphResult{}, err
	}
	root, err := s.detailReader.Read(ctx, detailRequest)
	if err != nil {
		return model.JudicialCitationGraphResult{}, contextOrOperationError(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return model.JudicialCitationGraphResult{}, err
	}
	if err := validateRootResult(root, request.Ref()); err != nil {
		return model.JudicialCitationGraphResult{}, newOperationError(err)
	}

	assembly, err := newGraphAssembly(root)
	if err != nil {
		return model.JudicialCitationGraphResult{}, newOperationError(err)
	}
	issues := []model.JudicialCitationIssue{}
	detailProvenance := lastProvenance(root.Provenance())
	metadataEvidence, err := model.NewJudicialCitationEvidence(model.JudicialCitationEvidenceValues{
		EvidenceLevel: model.JudicialCitationEvidenceLevelOfficialMetadata,
		Provenance:    detailProvenance,
	})
	if err != nil {
		return model.JudicialCitationGraphResult{}, newOperationError(err)
	}

	issues, err = s.addSharedMetadata(
		ctx,
		root.Data(),
		detailProvenance,
		metadataEvidence,
		assembly,
		issues,
	)
	if err != nil {
		return model.JudicialCitationGraphResult{}, err
	}

	outgoing := notRequestedOutcome()
	if wantsOutgoing(request.Direction()) {
		outgoing, issues, err = s.executeOutgoing(ctx, root, assembly, issues)
		if err != nil {
			return model.JudicialCitationGraphResult{}, err
		}
	}
	incoming := notRequestedOutcome()
	if wantsIncoming(request.Direction()) {
		incoming, issues, err = s.executeIncoming(
			ctx,
			root,
			request.IncomingLimit(),
			assembly,
			issues,
		)
		if err != nil {
			return model.JudicialCitationGraphResult{}, err
		}
	}

	if allRequestedDirectionsFailed(request.Direction(), outgoing, incoming) {
		return model.JudicialCitationGraphResult{}, allDirectionsError(request.Direction(), outgoing, incoming)
	}
	coverage, err := newCoverage(request, outgoing, incoming)
	if err != nil {
		return model.JudicialCitationGraphResult{}, newOperationError(err)
	}
	graph, err := assembly.graph(coverage)
	if err != nil {
		return model.JudicialCitationGraphResult{}, newOperationError(err)
	}
	status := model.JudicialCitationResultStatusComplete
	if hasSharedIssue(issues) || !requestedDirectionsComplete(request.Direction(), outgoing, incoming) {
		status = model.JudicialCitationResultStatusPartial
	}
	result, err := model.NewJudicialCitationGraphResult(model.JudicialCitationGraphResultValues{
		Status:         status,
		CoverageNotice: CoverageNotice,
		Graph:          graph,
		Issues:         issues,
	})
	if err != nil {
		return model.JudicialCitationGraphResult{}, newOperationError(err)
	}
	return result, nil
}

func (s *Service) addSharedMetadata(
	ctx context.Context,
	details model.JudicialDecisionDetails,
	provenance model.Provenance,
	evidence model.JudicialCitationEvidence,
	assembly *graphAssembly,
	issues []model.JudicialCitationIssue,
) ([]model.JudicialCitationIssue, error) {
	lower, exists, normalizeErr := judicialcitation.NormalizeLowerCourtDecision(details)
	lowerIssueNeeded := normalizeErr != nil
	if normalizeErr != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	if exists {
		added, err := assembly.addLowerCourt(lower, evidence)
		if err != nil {
			return nil, newOperationError(err)
		}
		if !added {
			lowerIssueNeeded = true
		}
	} else {
		mention, ok, mentionErr := lowerCourtUnresolvedMention(details, provenance)
		if mentionErr != nil {
			lowerIssueNeeded = true
		} else if ok {
			assembly.addUnresolved([]model.JudicialCitationUnresolvedMention{mention})
		}
	}
	if lowerIssueNeeded {
		issue, err := newFixedIssue(
			model.JudicialCitationIssueDirectionShared,
			model.JudicialCitationIssueStageOfficialDetailMetadata,
			"lower_court_metadata_unresolved",
			"原審情報を一意な裁判例参照へ正規化できませんでした。",
		)
		if err != nil {
			return nil, newOperationError(err)
		}
		issues = append(issues, issue)
	}

	provisions, normalizeErr := judicialcitation.NormalizeReferencedProvisions(
		ctx,
		s.lawAliasResolver,
		details,
		provenance,
	)
	if normalizeErr != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		issue, err := newFixedIssue(
			model.JudicialCitationIssueDirectionShared,
			model.JudicialCitationIssueStageLawReferenceResolution,
			"law_reference_resolution_failed",
			"参照法条を一意な法令と条文位置へ正規化できませんでした。",
		)
		if err != nil {
			return nil, newOperationError(err)
		}
		return append(issues, issue), nil
	}
	assembly.addUnresolved(provisions.Unresolved())
	lawLimitReached := false
	for _, reference := range provisions.References() {
		added, err := assembly.addLawReference(reference, evidence)
		if err != nil {
			return nil, newOperationError(err)
		}
		if !added {
			lawLimitReached = true
		}
	}
	if lawLimitReached {
		issue, err := newFixedIssue(
			model.JudicialCitationIssueDirectionShared,
			model.JudicialCitationIssueStageLawReferenceResolution,
			"law_provision_edge_limit_reached",
			"法条関係の処理上限に達したため、決定的な先頭だけを返しました。",
		)
		if err != nil {
			return nil, newOperationError(err)
		}
		issues = append(issues, issue)
	}
	return issues, nil
}

func (s *Service) executeOutgoing(
	ctx context.Context,
	root model.SourcedResource[model.JudicialDecisionDetails],
	assembly *graphAssembly,
	issues []model.JudicialCitationIssue,
) (directionOutcome, []model.JudicialCitationIssue, error) {
	if err := ctx.Err(); err != nil {
		return directionOutcome{}, nil, err
	}
	outcome := requestedOutcome()
	document, exists := firstFullTextDocument(root.Data().Summary().Documents())
	if !exists {
		outcome.status = model.JudicialCitationDirectionStatusUnavailable
		outcome.succeeded = true
		issue, err := newFixedIssue(
			model.JudicialCitationIssueDirectionOutgoing,
			model.JudicialCitationIssueStageOfficialDetailMetadata,
			"full_text_document_unavailable",
			"公式詳細ページに解析対象の全文 PDF が掲載されていません。",
		)
		if err != nil {
			return directionOutcome{}, nil, newOperationError(err)
		}
		return outcome, append(issues, issue), nil
	}
	request, err := judicialcasecitationextract.NewRequest(
		judicialcasecitationextract.RequestValues{Decision: root, Document: document},
	)
	if err != nil {
		return failedOutgoing(outcome, issues, err)
	}
	if err := ctx.Err(); err != nil {
		return directionOutcome{}, nil, err
	}
	result, err := s.citationExtractor.Extract(ctx, request)
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return directionOutcome{}, nil, contextErr
		}
		return failedOutgoing(outcome, issues, err)
	}
	if err := ctx.Err(); err != nil {
		return directionOutcome{}, nil, err
	}
	if err := result.Validate(); err != nil {
		return failedOutgoing(outcome, issues, err)
	}
	if result.DocumentTextStatus() == judicialcasecitationextract.DocumentTextStatusDocumentTextUnavailable {
		outcome.status = model.JudicialCitationDirectionStatusUnavailable
		outcome.succeeded = true
		issue, issueErr := newFixedIssue(
			model.JudicialCitationIssueDirectionOutgoing,
			model.JudicialCitationIssueStageOfficialPDFText,
			"document_text_unavailable",
			"公式 PDF に解析可能な text layer がありません。",
		)
		if issueErr != nil {
			return directionOutcome{}, nil, newOperationError(issueErr)
		}
		return outcome, append(issues, issue), nil
	}
	outcome.status = model.JudicialCitationDirectionStatusComplete
	outcome.methods = append(outcome.methods, model.JudicialCitationMethodOfficialPDFText)
	outcome.succeeded = true
	outcome.truncated = result.Truncated()
	assembly.addUnresolved(result.UnresolvedMentions())
	edgeLimitReached := false
	for _, mention := range result.ConfirmedDecisionMentions() {
		added, addErr := assembly.addOutgoingMention(mention)
		if addErr != nil {
			return directionOutcome{}, nil, newOperationError(addErr)
		}
		if !added {
			edgeLimitReached = true
		}
	}
	if result.Truncated() {
		outcome.status = model.JudicialCitationDirectionStatusPartial
		issue, issueErr := newFixedIssue(
			model.JudicialCitationIssueDirectionOutgoing,
			model.JudicialCitationIssueStageOfficialPDFText,
			"citation_occurrence_limit_reached",
			"判例参照の出現上限に達したため、決定的な先頭だけを返しました。",
		)
		if issueErr != nil {
			return directionOutcome{}, nil, newOperationError(issueErr)
		}
		issues = append(issues, issue)
	}
	if edgeLimitReached {
		outcome.status = model.JudicialCitationDirectionStatusPartial
		outcome.truncated = true
		issue, issueErr := newFixedIssue(
			model.JudicialCitationIssueDirectionOutgoing,
			model.JudicialCitationIssueStageOfficialPDFText,
			"judicial_decision_edge_limit_reached",
			"判例関係の処理上限に達したため、決定的な先頭だけを返しました。",
		)
		if issueErr != nil {
			return directionOutcome{}, nil, newOperationError(issueErr)
		}
		issues = append(issues, issue)
	}
	return outcome, issues, nil
}

func failedOutgoing(
	outcome directionOutcome,
	issues []model.JudicialCitationIssue,
	cause error,
) (directionOutcome, []model.JudicialCitationIssue, error) {
	outcome.failure = cause
	issue, err := newIssueFromError(
		model.JudicialCitationIssueDirectionOutgoing,
		model.JudicialCitationIssueStageOfficialPDFText,
		cause,
	)
	if err != nil {
		return directionOutcome{}, nil, newOperationError(err)
	}
	return outcome, append(issues, issue), nil
}

func (s *Service) executeIncoming(
	ctx context.Context,
	root model.SourcedResource[model.JudicialDecisionDetails],
	limit int,
	assembly *graphAssembly,
	issues []model.JudicialCitationIssue,
) (directionOutcome, []model.JudicialCitationIssue, error) {
	if err := ctx.Err(); err != nil {
		return directionOutcome{}, nil, err
	}
	outcome := requestedOutcome()
	outcome.limit = limit
	outcome.attemptedSearches = plannedCandidateSearches(root.Data())
	request, err := judicialcitingcandidatesearch.NewRequest(
		judicialcitingcandidatesearch.RequestValues{Target: root, Limit: &limit},
	)
	if err != nil {
		return failedIncoming(outcome, issues, err)
	}
	if err := ctx.Err(); err != nil {
		return directionOutcome{}, nil, err
	}
	result, err := s.candidateSearcher.Search(ctx, request)
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return directionOutcome{}, nil, contextErr
		}
		return failedIncoming(outcome, issues, err)
	}
	if err := ctx.Err(); err != nil {
		return directionOutcome{}, nil, err
	}
	if err := result.Validate(); err != nil {
		return failedIncoming(outcome, issues, err)
	}
	outcome.succeeded = true
	outcome.status = model.JudicialCitationDirectionStatusComplete
	if result.Status() == judicialcitingcandidatesearch.ResultStatusPartial {
		outcome.status = model.JudicialCitationDirectionStatusPartial
	}
	coverage := result.Coverage()
	outcome.attemptedSearches = len(coverage.Attempts())
	outcome.completedSearches = completedCandidateSearches(coverage.Attempts())
	if outcome.completedSearches > 0 {
		outcome.methods = append(outcome.methods, model.JudicialCitationMethodOfficialCaseSearch)
	}
	outcome.truncated = coverage.Truncated()
	for _, candidateIssue := range result.Issues() {
		public := candidateIssue.ErrorResult()
		issue, issueErr := model.NewJudicialCitationIssue(model.JudicialCitationIssueValues{
			Direction: model.JudicialCitationIssueDirectionIncoming,
			Stage:     model.JudicialCitationIssueStageOfficialCaseSearch,
			Code:      string(public.Code()),
			Message:   public.Message(),
			Retryable: public.Retryable(),
		})
		if issueErr != nil {
			return directionOutcome{}, nil, newOperationError(issueErr)
		}
		issues = append(issues, issue)
	}
	edgeLimitReached := false
	for _, candidate := range result.Items() {
		added, addErr := assembly.addIncomingCandidate(candidate)
		if addErr != nil {
			return directionOutcome{}, nil, newOperationError(addErr)
		}
		if !added {
			edgeLimitReached = true
		}
	}
	if edgeLimitReached {
		outcome.status = model.JudicialCitationDirectionStatusPartial
		outcome.truncated = true
		issue, issueErr := newFixedIssue(
			model.JudicialCitationIssueDirectionIncoming,
			model.JudicialCitationIssueStageOfficialCaseSearch,
			"judicial_decision_edge_limit_reached",
			"判例関係の処理上限に達したため、決定的な先頭だけを返しました。",
		)
		if issueErr != nil {
			return directionOutcome{}, nil, newOperationError(issueErr)
		}
		issues = append(issues, issue)
	}
	return outcome, issues, nil
}

func failedIncoming(
	outcome directionOutcome,
	issues []model.JudicialCitationIssue,
	cause error,
) (directionOutcome, []model.JudicialCitationIssue, error) {
	outcome.failure = cause
	issue, err := newIssueFromError(
		model.JudicialCitationIssueDirectionIncoming,
		model.JudicialCitationIssueStageOfficialCaseSearch,
		cause,
	)
	if err != nil {
		return directionOutcome{}, nil, newOperationError(err)
	}
	return outcome, append(issues, issue), nil
}

type directionOutcome struct {
	status            model.JudicialCitationDirectionStatus
	methods           []model.JudicialCitationMethod
	truncated         bool
	limit             int
	attemptedSearches int
	completedSearches int
	succeeded         bool
	failure           error
}

func notRequestedOutcome() directionOutcome {
	return directionOutcome{
		status:  model.JudicialCitationDirectionStatusNotRequested,
		methods: []model.JudicialCitationMethod{},
	}
}

func requestedOutcome() directionOutcome {
	return directionOutcome{
		status:  model.JudicialCitationDirectionStatusPartial,
		methods: []model.JudicialCitationMethod{model.JudicialCitationMethodOfficialDetailMetadata},
	}
}

func newCoverage(
	request Request,
	outgoing, incoming directionOutcome,
) (model.JudicialCitationCoverage, error) {
	outgoingCoverage, err := newDirectionCoverage(outgoing, false)
	if err != nil {
		return model.JudicialCitationCoverage{}, err
	}
	incomingCoverage, err := newDirectionCoverage(incoming, true)
	if err != nil {
		return model.JudicialCitationCoverage{}, err
	}
	return model.NewJudicialCitationCoverage(model.JudicialCitationCoverageValues{
		RequestedDirection: request.Direction(),
		HopDepth:           1,
		Outgoing:           outgoingCoverage,
		Incoming:           incomingCoverage,
	})
}

func newDirectionCoverage(
	outcome directionOutcome,
	incoming bool,
) (model.JudicialCitationDirectionCoverage, error) {
	values := model.JudicialCitationDirectionCoverageValues{
		Status:    outcome.status,
		Methods:   slices.Clone(outcome.methods),
		Truncated: outcome.truncated,
	}
	if incoming && outcome.status != model.JudicialCitationDirectionStatusNotRequested {
		values.Limit = &outcome.limit
		values.AttemptedSearches = &outcome.attemptedSearches
		values.CompletedSearches = &outcome.completedSearches
	}
	return model.NewJudicialCitationDirectionCoverage(values)
}

func validateRootResult(
	root model.SourcedResource[model.JudicialDecisionDetails],
	expected model.SourceResourceRef,
) error {
	if err := root.Validate(); err != nil {
		return fmt.Errorf("詳細取得結果が有効ではありません")
	}
	if root.Ref() != expected {
		return fmt.Errorf("詳細取得結果の参照が入力と一致しません")
	}
	return nil
}

func lowerCourtUnresolvedMention(
	details model.JudicialDecisionDetails,
	provenance model.Provenance,
) (model.JudicialCitationUnresolvedMention, bool, error) {
	parts := make([]string, 0, 2)
	if court, exists := details.LowerCourtName(); exists {
		parts = append(parts, court)
	}
	if number, exists := details.LowerCourtCaseNumber(); exists {
		parts = append(parts, number)
	}
	if len(parts) == 0 {
		return model.JudicialCitationUnresolvedMention{}, false, nil
	}
	mention, err := model.NewJudicialCitationUnresolvedMention(
		model.JudicialCitationUnresolvedMentionValues{
			MentionType: model.JudicialCitationMentionTypeDecision,
			MentionText: strings.Join(parts, " "),
			Reason:      model.JudicialCitationUnresolvedReasonInsufficientIdentity,
			Provenance:  provenance,
		},
	)
	if err != nil {
		return model.JudicialCitationUnresolvedMention{}, false, err
	}
	return mention, true, nil
}

func firstFullTextDocument(
	documents []model.JudicialDocumentLink,
) (model.JudicialDocumentLink, bool) {
	for _, document := range documents {
		if document.Kind() == model.JudicialDocumentKindFullText &&
			document.MediaType() == model.JudicialDocumentMediaTypePDF {
			return document, true
		}
	}
	return model.JudicialDocumentLink{}, false
}

func plannedCandidateSearches(details model.JudicialDecisionDetails) int {
	if reporter, exists := details.ReporterCitation(); exists &&
		reporter != details.Summary().CaseNumber() {
		return 2
	}
	return 1
}

func completedCandidateSearches(
	attempts []judicialcitingcandidatesearch.CoverageAttempt,
) int {
	completed := 0
	for _, attempt := range attempts {
		if attempt.Status == judicialcitingcandidatesearch.AttemptStatusComplete {
			completed++
		}
	}
	return completed
}

func wantsOutgoing(direction model.JudicialCitationRequestedDirection) bool {
	return direction != model.JudicialCitationRequestedDirectionIncoming
}

func wantsIncoming(direction model.JudicialCitationRequestedDirection) bool {
	return direction != model.JudicialCitationRequestedDirectionOutgoing
}

func allRequestedDirectionsFailed(
	direction model.JudicialCitationRequestedDirection,
	outgoing, incoming directionOutcome,
) bool {
	switch direction {
	case model.JudicialCitationRequestedDirectionOutgoing:
		return !outgoing.succeeded
	case model.JudicialCitationRequestedDirectionIncoming:
		return !incoming.succeeded
	case model.JudicialCitationRequestedDirectionBoth:
		return !outgoing.succeeded && !incoming.succeeded
	default:
		return true
	}
}

func allDirectionsError(
	direction model.JudicialCitationRequestedDirection,
	outgoing, incoming directionOutcome,
) error {
	if direction == model.JudicialCitationRequestedDirectionOutgoing {
		return newOperationError(outgoing.failure)
	}
	if direction == model.JudicialCitationRequestedDirectionIncoming {
		return newOperationError(incoming.failure)
	}
	if outgoing.failure != nil {
		return newOperationError(outgoing.failure)
	}
	return newOperationError(incoming.failure)
}

func requestedDirectionsComplete(
	direction model.JudicialCitationRequestedDirection,
	outgoing, incoming directionOutcome,
) bool {
	switch direction {
	case model.JudicialCitationRequestedDirectionOutgoing:
		return outgoing.status == model.JudicialCitationDirectionStatusComplete
	case model.JudicialCitationRequestedDirectionIncoming:
		return incoming.status == model.JudicialCitationDirectionStatusComplete
	case model.JudicialCitationRequestedDirectionBoth:
		return outgoing.status == model.JudicialCitationDirectionStatusComplete &&
			incoming.status == model.JudicialCitationDirectionStatusComplete
	default:
		return false
	}
}

func hasSharedIssue(issues []model.JudicialCitationIssue) bool {
	return slices.ContainsFunc(issues, func(issue model.JudicialCitationIssue) bool {
		return issue.Direction() == model.JudicialCitationIssueDirectionShared
	})
}

func contextOrOperationError(ctx context.Context, cause error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return newOperationError(cause)
}

func lastProvenance(values []model.Provenance) model.Provenance {
	return values[len(values)-1]
}

func isNilDependency(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
