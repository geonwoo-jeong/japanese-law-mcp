package hanrei

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitingcandidatesearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	citingCandidateParseTimeout = 4 * time.Second
	maximumCitingCandidateNodes = 200000
)

type citingCandidateAdapterDependencies = searchAdapterDependencies

// JudicialCitingCandidateSearchAdapter は、公式検索だけを使う候補検索 adapter である。
type JudicialCitingCandidateSearchAdapter struct {
	dependencies citingCandidateAdapterDependencies
}

var _ judicialcitingcandidatesearch.Port = (*JudicialCitingCandidateSearchAdapter)(nil)

// NewJudicialCitingCandidateSearchAdapter は、固定 origin と共有同時実行枠を使う adapter を返す。
func NewJudicialCitingCandidateSearchAdapter() (*JudicialCitingCandidateSearchAdapter, error) {
	return newJudicialCitingCandidateSearchAdapter(citingCandidateAdapterDependencies{
		doer:         newProductionHTTPClient(),
		now:          time.Now,
		gate:         sharedCourtsHanreiGate,
		parseTimeout: citingCandidateParseTimeout,
	})
}

func newJudicialCitingCandidateSearchAdapter(
	dependencies citingCandidateAdapterDependencies,
) (*JudicialCitingCandidateSearchAdapter, error) {
	if dependencies.doer == nil || dependencies.now == nil ||
		dependencies.gate == nil || dependencies.parseTimeout <= 0 {
		return nil, fmt.Errorf("裁判所の被引用候補検索 adapter の依存関係が不足しています")
	}
	if cap(dependencies.gate) != 1 {
		return nil, fmt.Errorf("裁判所の共有同時実行枠は一件でなければなりません")
	}
	return &JudicialCitingCandidateSearchAdapter{dependencies: dependencies}, nil
}

type citingCandidateSearchPlan struct {
	kind  judicialcitingcandidatesearch.SearchKind
	query string
}

type citingCandidateBudget struct {
	encodedBytes   int
	decodedBytes   int
	nodes          int
	parseRemaining time.Duration
}

// Search は、事件番号と任意の判例集表記を順に最大一回ずつ検索する。
func (a *JudicialCitingCandidateSearchAdapter) Search(
	ctx context.Context,
	request judicialcitingcandidatesearch.Request,
) (judicialcitingcandidatesearch.Result, error) {
	if ctx == nil {
		return judicialcitingcandidatesearch.Result{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return judicialcitingcandidatesearch.Result{}, err
	}
	if err := a.acquire(ctx); err != nil {
		return judicialcitingcandidatesearch.Result{}, err
	}
	defer a.release()
	return a.searchAcquired(ctx, request)
}

func (a *JudicialCitingCandidateSearchAdapter) searchAcquired(
	ctx context.Context,
	request judicialcitingcandidatesearch.Request,
) (judicialcitingcandidatesearch.Result, error) {
	plans := citingCandidatePlans(request.Target().Data())
	budget := citingCandidateBudget{parseRemaining: a.dependencies.parseTimeout}
	attempts := make([]judicialcitingcandidatesearch.Attempt, 0, len(plans))
	issues := make([]judicialcitingcandidatesearch.Issue, 0, len(plans))
	occurrences := make([]judicialcitingcandidatesearch.Candidate, 0)
	observedCount := 0
	sourceTruncated := false
	var firstFailure error
	successes := 0

	for _, plan := range plans {
		if ctx.Err() != nil {
			return judicialcitingcandidatesearch.Result{}, normalizeCitingCandidateError(ctx, ctx.Err())
		}
		mapped, truncated, err := a.executeSearch(ctx, plan.query, &budget)
		if err != nil {
			normalized := normalizeCitingCandidateError(ctx, err)
			if errors.Is(normalized, context.Canceled) {
				return judicialcitingcandidatesearch.Result{}, context.Canceled
			}
			if firstFailure == nil {
				firstFailure = normalized
			}
			var sourceError model.SourceError
			if !errors.As(normalized, &sourceError) {
				return judicialcitingcandidatesearch.Result{}, normalized
			}
			issue, issueErr := judicialcitingcandidatesearch.NewIssue(
				judicialcitingcandidatesearch.IssueValues{
					SearchKind:  plan.kind,
					SourceError: sourceError,
				},
			)
			if issueErr != nil {
				return judicialcitingcandidatesearch.Result{}, fmt.Errorf(
					"候補検索 issue を構成できません: %w",
					issueErr,
				)
			}
			issues = append(issues, issue)
			attempts = append(attempts, judicialcitingcandidatesearch.Attempt{
				SearchKind: plan.kind,
				Status:     judicialcitingcandidatesearch.AttemptStatusFailed,
			})
			continue
		}
		successes++
		observedCount += len(mapped)
		occurrences = append(occurrences, mapped...)
		sourceTruncated = sourceTruncated || truncated
		attempts = append(attempts, judicialcitingcandidatesearch.Attempt{
			SearchKind: plan.kind,
			Status:     judicialcitingcandidatesearch.AttemptStatusComplete,
		})
	}
	if successes == 0 {
		return judicialcitingcandidatesearch.Result{}, firstFailure
	}
	return buildCitingCandidateResult(
		request,
		attempts,
		issues,
		occurrences,
		observedCount,
		sourceTruncated,
	)
}

func citingCandidatePlans(
	details model.JudicialDecisionDetails,
) []citingCandidateSearchPlan {
	caseNumber := details.Summary().CaseNumber()
	plans := []citingCandidateSearchPlan{{
		kind:  judicialcitingcandidatesearch.SearchKindCaseNumber,
		query: caseNumber,
	}}
	if reporterCitation, exists := details.ReporterCitation(); exists && reporterCitation != caseNumber {
		plans = append(plans, citingCandidateSearchPlan{
			kind:  judicialcitingcandidatesearch.SearchKindReporterCitation,
			query: reporterCitation,
		})
	}
	return plans
}

func (a *JudicialCitingCandidateSearchAdapter) executeSearch(
	ctx context.Context,
	query string,
	budget *citingCandidateBudget,
) ([]judicialcitingcandidatesearch.Candidate, bool, error) {
	fetched, consumedEncoded, err := fetchCitingCandidateResponse(
		ctx,
		a.dependencies.doer,
		a.dependencies.now,
		query,
		maximumCitingCandidateResponseBytes-budget.encodedBytes,
	)
	budget.encodedBytes += consumedEncoded
	if err != nil {
		return nil, false, err
	}
	if budget.parseRemaining <= 0 {
		return nil, false, newSearchSourceError(
			model.SourceErrorCodeSourceProcessingLimit,
			"",
		)
	}
	started := time.Now()
	processing, cancel := context.WithTimeout(ctx, budget.parseRemaining)
	defer func() {
		cancel()
		elapsed := time.Since(started)
		if elapsed >= budget.parseRemaining {
			budget.parseRemaining = 0
		} else {
			budget.parseRemaining -= elapsed
		}
	}()
	body, consumedDecoded, err := decodeCitingCandidateResponse(
		ctx,
		processing,
		fetched,
		maximumCitingCandidateDecompressedBytes-budget.decodedBytes,
	)
	budget.decodedBytes += consumedDecoded
	if err != nil {
		return nil, false, err
	}
	response, nodeCount, err := parseSearchResponseWithBudget(
		processing,
		body,
		len(body),
		maximumCitingCandidateNodes-budget.nodes,
		maximumSearchHTMLDepth,
	)
	budget.nodes += nodeCount
	if err != nil {
		return nil, false, err
	}
	return mapCitingCandidateOccurrences(processing, response, fetched.retrievedAt)
}

func buildCitingCandidateResult(
	request judicialcitingcandidatesearch.Request,
	attempts []judicialcitingcandidatesearch.Attempt,
	issues []judicialcitingcandidatesearch.Issue,
	occurrences []judicialcitingcandidatesearch.Candidate,
	observedCount int,
	sourceTruncated bool,
) (judicialcitingcandidatesearch.Result, error) {
	unique := make([]judicialcitingcandidatesearch.Candidate, 0, len(occurrences))
	indexes := make(map[model.SourceResourceRef]int, len(occurrences))
	targetRef := request.Target().Ref()
	for _, occurrence := range occurrences {
		ref := occurrence.Decision().Ref()
		if ref == targetRef {
			continue
		}
		index, exists := indexes[ref]
		if !exists {
			indexes[ref] = len(unique)
			unique = append(unique, occurrence)
			continue
		}
		merged, err := judicialcitingcandidatesearch.NewCandidate(
			judicialcitingcandidatesearch.CandidateValues{
				Decision: unique[index].Decision(),
				Evidence: slices.Concat(unique[index].Evidence(), occurrence.Evidence()),
			},
		)
		if err != nil {
			return judicialcitingcandidatesearch.Result{}, fmt.Errorf(
				"重複候補の evidence を統合できません: %w",
				err,
			)
		}
		unique[index] = merged
	}
	truncated := sourceTruncated || len(unique) > request.Limit()
	items := unique
	if len(items) > request.Limit() {
		items = items[:request.Limit()]
	}
	coverage, err := judicialcitingcandidatesearch.NewCoverage(
		judicialcitingcandidatesearch.CoverageValues{
			Attempts:              attempts,
			ObservedItemCount:     observedCount,
			DedupedCandidateCount: len(unique),
			Truncated:             truncated,
		},
	)
	if err != nil {
		return judicialcitingcandidatesearch.Result{}, fmt.Errorf(
			"候補検索 coverage を構成できません: %w",
			err,
		)
	}
	status := judicialcitingcandidatesearch.ResultStatusComplete
	if len(issues) != 0 {
		status = judicialcitingcandidatesearch.ResultStatusPartial
	}
	return judicialcitingcandidatesearch.NewResult(
		judicialcitingcandidatesearch.ResultValues{
			Status:   status,
			Items:    items,
			Coverage: coverage,
			Issues:   issues,
		},
	)
}

func (a *JudicialCitingCandidateSearchAdapter) acquire(ctx context.Context) error {
	if ctx.Err() != nil {
		return normalizeCitingCandidateError(ctx, ctx.Err())
	}
	select {
	case a.dependencies.gate <- struct{}{}:
		return nil
	default:
		return newCitingCandidateSourceError(model.SourceErrorCodeSourceBusy, "")
	}
}

func (a *JudicialCitingCandidateSearchAdapter) release() {
	<-a.dependencies.gate
}
