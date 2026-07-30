package defaultprofile

import (
	"context"
	"fmt"
	"slices"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
)

// BuildStandardReport は、default profile set の意味・再現性・実行測定を集約する。
func (e *Evaluator) BuildStandardReport(
	ctx context.Context,
	corpus legalquerycorpus.Corpus,
	baselineVersion string,
) (legalqueryeval.StandardReport, error) {
	if e == nil {
		return legalqueryeval.StandardReport{},
			fmt.Errorf("default profile evaluator は nil にできません")
	}
	semantic, reproducibility, err :=
		legalqueryeval.EvaluateSemanticHoldoutReproducibility(
			ctx,
			corpus,
			e.EvaluateWithPlan,
		)
	if err != nil {
		return legalqueryeval.StandardReport{}, err
	}
	derivedObservations, err := legalqueryeval.EvaluateDerivedObservations(
		corpus.Holdout(),
		semantic,
	)
	if err != nil {
		return legalqueryeval.StandardReport{}, err
	}
	execution, err := e.EvaluateExecution(ctx, corpus)
	if err != nil {
		return legalqueryeval.StandardReport{}, err
	}

	profileVersions, err := profileVersionReports(
		e.planning.ProfileMetadata(),
	)
	if err != nil {
		return legalqueryeval.StandardReport{}, err
	}
	holdoutCaseIDs := make([]string, 0, len(corpus.Holdout()))
	for _, semanticCase := range corpus.Holdout() {
		holdoutCaseIDs = append(holdoutCaseIDs, semanticCase.CaseID())
	}
	manifest := corpus.Manifest()
	return legalqueryeval.NewStandardReport(
		legalqueryeval.StandardReportValues{
			CorpusVersion:        manifest.CorpusVersion(),
			HoldoutDigest:        manifest.HoldoutDigest(),
			ProfileSetID:         "default",
			ProfileSetVersion:    e.planning.Profiles().ProfileVersion(),
			RankingVersion:       e.planning.Profiles().RankingVersion(),
			ProfileVersions:      profileVersions,
			BaselineVersion:      baselineVersion,
			DevelopmentCaseCount: len(corpus.Development()),
			HoldoutCaseIDs:       holdoutCaseIDs,
			Semantic:             semantic,
			Execution:            execution,
			Reproducibility:      reproducibility,
			DerivedObservations:  derivedObservations,
		},
	)
}

func profileVersionReports(
	metadata []legalquery.QueryProfileMetadata,
) ([]legalqueryeval.ProfileVersionReport, error) {
	ordered := append([]legalquery.QueryProfileMetadata{}, metadata...)
	slices.SortFunc(
		ordered,
		func(left, right legalquery.QueryProfileMetadata) int {
			switch {
			case left.ProfileID() < right.ProfileID():
				return -1
			case left.ProfileID() > right.ProfileID():
				return 1
			default:
				return 0
			}
		},
	)
	profileVersions := make(
		[]legalqueryeval.ProfileVersionReport,
		0,
		len(ordered),
	)
	for _, profile := range ordered {
		report, reportErr := legalqueryeval.NewProfileVersionReport(
			legalqueryeval.ProfileVersionReportValues{
				ProfileID:      profile.ProfileID(),
				ProfileVersion: profile.ProfileVersion(),
				RankingVersion: profile.RankingVersion(),
			},
		)
		if reportErr != nil {
			return nil, reportErr
		}
		profileVersions = append(profileVersions, report)
	}
	return profileVersions, nil
}
