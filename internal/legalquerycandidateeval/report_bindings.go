package legalquerycandidateeval

import (
	"fmt"
	"os"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
)

type loadedReport struct {
	raw    []byte
	digest string
}

func validateTrackedReportBindings(
	repository *legalqueryartifact.Repository,
	candidateRoot *legalqueryartifact.Repository,
	requests map[string]loadedArtifact[EvaluationRequest],
	results map[string]loadedArtifact[EvaluationResult],
	layout preparationRootLayout,
) (map[string][]byte, error) {
	failedReports, err := loadFailedReports(candidateRoot, layout)
	if err != nil {
		return nil, err
	}
	baselines, err := loadBaselineVersionReports(repository)
	if err != nil {
		return nil, err
	}
	reports := make(map[string][]byte, len(results))
	usedFailed := make(map[string]struct{}, len(failedReports))
	for _, evaluationID := range sortedKeys(requests) {
		request := requests[evaluationID].document
		result, consumed := results[evaluationID]
		failed, hasFailed := failedReports[evaluationID]
		baseline, hasBaseline := baselines[request.BaselineVersion]
		if !consumed {
			if hasFailed || hasBaseline {
				return nil, fmt.Errorf("未評価 request に予約済み report があります")
			}
			continue
		}
		switch result.document.Outcome {
		case EvaluationOutcomePassed:
			if hasFailed || !hasBaseline || baseline.digest != result.document.ReportSHA256 {
				return nil, fmt.Errorf("passed result の baseline binding が一致しません")
			}
			reports[evaluationID] = append([]byte(nil), baseline.raw...)
		case EvaluationOutcomeFailed:
			if hasBaseline || !hasFailed || failed.digest != result.document.ReportSHA256 {
				return nil, fmt.Errorf("failed result の report binding が一致しません")
			}
			usedFailed[evaluationID] = struct{}{}
			reports[evaluationID] = append([]byte(nil), failed.raw...)
		default:
			return nil, fmt.Errorf("tracked result の outcome が不正です")
		}
	}
	if len(usedFailed) != len(failedReports) {
		return nil, fmt.Errorf("result から参照されない failed report があります")
	}
	return reports, nil
}

func loadFailedReports(
	root *legalqueryartifact.Repository,
	layout preparationRootLayout,
) (map[string]loadedReport, error) {
	if !layout.failedReportsPresent {
		return map[string]loadedReport{}, nil
	}
	directory, err := root.OpenChild("failed-reports")
	if err != nil {
		return nil, fmt.Errorf("failed report directory を開けません: %w", err)
	}
	defer func() { _ = directory.Close() }()
	entries, err := directory.ReadDirectory(maximumFailedReportFiles, maximumFailedReportTotal)
	if err != nil {
		return nil, fmt.Errorf("failed report directory を列挙できません: %w", err)
	}
	reports := make(map[string]loadedReport, len(entries))
	for _, entry := range entries {
		evaluationID, err := validateArtifactEntry(entry, evaluationIDPattern, maximumEvaluationReportBytes)
		if err != nil {
			return nil, fmt.Errorf("failed report: %w", err)
		}
		raw, err := directory.ReadRegular(entry.Name(), maximumEvaluationReportBytes)
		if err != nil {
			return nil, fmt.Errorf("failed report を読めません: %w", err)
		}
		reports[evaluationID] = loadedReport{raw: raw, digest: RawSHA256(raw)}
	}
	return reports, nil
}

func loadBaselineVersionReports(
	repository *legalqueryartifact.Repository,
) (map[string]loadedReport, error) {
	testdata, err := repository.OpenChild("testdata")
	if err != nil {
		return nil, fmt.Errorf("baseline testdata directory を開けません: %w", err)
	}
	defer func() { _ = testdata.Close() }()
	legalquery, err := testdata.OpenChild("legalquery")
	if err != nil {
		return nil, fmt.Errorf("baseline legalquery directory を開けません: %w", err)
	}
	defer func() { _ = legalquery.Close() }()
	baselines, err := legalquery.OpenChild("baselines")
	if err != nil {
		return nil, fmt.Errorf("baseline directory を開けません: %w", err)
	}
	defer func() { _ = baselines.Close() }()
	versions, err := baselines.OpenChild("versions")
	if err != nil {
		return nil, fmt.Errorf("baseline versions directory を開けません: %w", err)
	}
	defer func() { _ = versions.Close() }()
	entries, err := versions.ReadDirectory(maximumResultFiles, maximumFailedReportTotal)
	if err != nil {
		return nil, fmt.Errorf("baseline versions directory を列挙できません: %w", err)
	}
	reports := make(map[string]loadedReport, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		info := entry.Info()
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() ||
			info.Size() < 1 || info.Size() > maximumEvaluationReportBytes ||
			!strings.HasSuffix(name, ".json") {
			return nil, fmt.Errorf("baseline version entry が通常 JSON file ではありません")
		}
		version := strings.TrimSuffix(name, ".json")
		if !baselineVersionPattern.MatchString(version) {
			return nil, fmt.Errorf("baseline version file 名が不正です")
		}
		raw, err := versions.ReadRegular(name, maximumEvaluationReportBytes)
		if err != nil {
			return nil, fmt.Errorf("baseline version report を読めません: %w", err)
		}
		reports[version] = loadedReport{raw: raw, digest: RawSHA256(raw)}
	}
	return reports, nil
}
