package legalqueryplanning

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	nextProfileSetDevelopmentOnlyCalibrationID = "next-profile-set-development-only-calibration"
	nextCalibratedCoreVersion                  = "core-2026-07-31-36"
	nextCalibratedJudicialVersion              = "judicial-cases-2026-07-31-12"
	nextCalibratedProfileSetVersion            = "profile-set-sha256-" +
		"5107d2ab16dd1668fe316c34d153b4eed3990ac6555db80f1617630529fe7c6c"
)

func TestNextProfileSetはDevelopmentだけで校正版を固定する(t *testing.T) {
	t.Parallel()

	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf(
			"%s: repository path を解決できません: %v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			err,
		)
	}
	firstDependencies, firstDevelopment := loadStage43CalibrationRun(
		t,
		repository,
	)
	secondDependencies, secondDevelopment := loadStage43CalibrationRun(
		t,
		repository,
	)
	if firstDevelopment.CorpusVersion() != "corpus-v10" ||
		firstDevelopment.SchemaVersion() != 2 ||
		len(firstDevelopment.Cases()) != 43 ||
		firstDevelopment.ContentDigest() != secondDevelopment.ContentDigest() {
		t.Fatalf(
			"%s: development identity = %q/%d/%q/%d",
			nextProfileSetDevelopmentOnlyCalibrationID,
			firstDevelopment.CorpusVersion(),
			firstDevelopment.SchemaVersion(),
			firstDevelopment.ContentDigest(),
			len(firstDevelopment.Cases()),
		)
	}
	assertStage43DevelopmentAssertionIDs(t, firstDevelopment.Cases())
	assertStage43DevelopmentAssertionIDs(t, secondDevelopment.Cases())
	assertStage43CalibratedVersions(t, firstDependencies)
	assertStage43CalibratedVersions(t, secondDependencies)
	assertStage43CalibrationFingerprint(
		t,
		firstDependencies,
		firstDevelopment,
		secondDependencies,
		secondDevelopment,
	)
}

func loadStage43CalibrationRun(
	t *testing.T,
	repository string,
) (Dependencies, legalquerycorpus.DevelopmentCorpus) {
	t.Helper()

	developmentOnlyRepository := stage43DevelopmentOnlyRepository(t, repository)
	development, err := legalquerycorpus.LoadDevelopment(
		t.Context(),
		developmentOnlyRepository,
		"testdata/legalquery/corpus-v10/development",
	)
	if err != nil {
		t.Fatalf(
			"%s: development を読み込めません: %v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			err,
		)
	}
	dependencies, err := buildStage36NextDependencies()
	if err != nil {
		t.Fatalf(
			"%s: next profile set を構成できません: %v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			err,
		)
	}
	return dependencies, development
}

func assertStage43CalibratedVersions(t *testing.T, dependencies Dependencies) {
	t.Helper()

	metadata := dependencies.ProfileMetadata()
	if len(metadata) != 2 ||
		metadata[0].ProfileID() != "core" ||
		metadata[0].ProfileVersion() != nextCalibratedCoreVersion ||
		metadata[0].CueSetVersion() != "core-cues-2026-07-31-16" ||
		metadata[1].ProfileID() != JudicialCasesPackID ||
		metadata[1].ProfileVersion() != nextCalibratedJudicialVersion ||
		metadata[1].CueSetVersion() !=
			"judicial-cases-cues-2026-07-31-5" ||
		dependencies.Profiles().RankingVersion() != nextRankingVersion ||
		dependencies.Profiles().ProfileVersion() !=
			nextCalibratedProfileSetVersion {
		t.Fatalf(
			"%s: 校正版 identity が固定値と一致しません: profileSetVersion=%q",
			nextProfileSetDevelopmentOnlyCalibrationID,
			dependencies.Profiles().ProfileVersion(),
		)
	}
	for _, value := range metadata {
		assertStage43CalibratedPolicy(t, value)
	}
}

func assertStage43DevelopmentAssertionIDs(
	t *testing.T,
	cases []legalquerycorpus.SemanticCase,
) {
	t.Helper()

	want := []string{
		"shared-terminal-five-distinct-meanings",
		"shared-terminal-five-with-repeated-meaning",
		"shared-terminal-four-distinct-meanings",
		"shared-terminal-nonunique-maximal-sequence",
		"shared-terminal-repeated-meaning-different-span",
		"shared-terminal-repeated-separator",
		"shared-terminal-same-span-multiple-meanings",
		"shared-terminal-tail-connector-chain",
		"shared-terminal-unknown-separator",
		"shared-terminal-valid-maximal-no-subsequence",
		"shared-terminal-valid-separator",
	}
	var got []string
	for _, semanticCase := range cases {
		got = append(got, semanticCase.DevelopmentAssertionIDs()...)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf(
			"%s: development assertion IDs = %#v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			got,
		)
	}
}

func assertStage43CalibratedPolicy(
	t *testing.T,
	metadata legalquery.QueryProfileMetadata,
) {
	t.Helper()

	score := metadata.Score()
	selection := metadata.Selection()
	margin, marginExists := selection.BranchRetentionMargin()
	if metadata.SchemaVersion() != 2 ||
		metadata.RankingVersion() != nextRankingVersion ||
		score.Minimum() != 0 ||
		score.Maximum() != 405 ||
		score.HighConfidenceAt() != 130 ||
		score.MediumConfidenceAt() != 80 ||
		selection.SingleThreshold() != 85 ||
		selection.MinimumExecutionThreshold() != 80 ||
		selection.SingleMargin() != 25 ||
		selection.HedgeMargin() != 10 ||
		!marginExists || margin != 12 ||
		!slices.Equal(metadata.TieBreak(), []legalquery.QueryTieBreak{
			legalquery.QueryTieBreakEvidenceSet,
			legalquery.QueryTieBreakStepCount,
			legalquery.QueryTieBreakMeaningSignature,
			legalquery.QueryTieBreakSourcePosition,
		}) {
		t.Fatalf(
			"%s: profile %q の校正 policy が固定値と一致しません",
			nextProfileSetDevelopmentOnlyCalibrationID,
			metadata.ProfileID(),
		)
	}
	wantWeights := []struct {
		code   legalquery.EvidenceCode
		weight int
	}{
		{legalquery.EvidenceOfficialIdentifier, 90},
		{legalquery.EvidenceStructuredReference, 80},
		{legalquery.EvidenceExplicitTask, 60},
		{legalquery.EvidenceExplicitResource, 50},
		{legalquery.EvidenceOfficialAlias, 40},
		{legalquery.EvidenceLegalConcept, 35},
		{legalquery.EvidenceMorphologicalContext, 25},
		{legalquery.EvidenceUniqueTypoCorrection, 15},
		{legalquery.EvidenceGeneralTerm, 10},
	}
	weights := score.EvidenceWeights()
	if len(weights) != len(wantWeights) {
		t.Fatalf(
			"%s: profile %q の weight 件数 = %d",
			nextProfileSetDevelopmentOnlyCalibrationID,
			metadata.ProfileID(),
			len(weights),
		)
	}
	for index, want := range wantWeights {
		if weights[index].Code() != want.code ||
			weights[index].Weight() != want.weight {
			t.Fatalf(
				"%s: profile %q の weight[%d] = %s/%d",
				nextProfileSetDevelopmentOnlyCalibrationID,
				metadata.ProfileID(),
				index,
				weights[index].Code(),
				weights[index].Weight(),
			)
		}
	}
}

func evaluateStage43DevelopmentCase(
	ctx context.Context,
	dependencies Dependencies,
	semanticCase legalquerycorpus.SemanticCase,
) (
	legalquery.LegalQueryPlan,
	error,
) {
	request, err := stage43ProductRequest(semanticCase.Request())
	if err != nil {
		var argumentError legalquery.ArgumentError
		if !errors.As(err, &argumentError) {
			return legalquery.LegalQueryPlan{}, err
		}
		expected, isRequestError := semanticCase.Expected().(legalquerycorpus.ExpectedRequestError)
		if !isRequestError {
			return legalquery.LegalQueryPlan{}, err
		}
		if expected.ErrorCode() != argumentError.Code() ||
			expected.Field() != legalquerycorpus.RequestErrorField(
				argumentError.Field(),
			) {
			return legalquery.LegalQueryPlan{}, fmt.Errorf(
				"request_error の code/field が一致しません",
			)
		}
		return legalquery.LegalQueryPlan{}, nil
	}
	if semanticCase.Expected().Kind() ==
		legalquerycorpus.SemanticExpectedKindRequestError {
		return legalquery.LegalQueryPlan{}, fmt.Errorf(
			"request_error を期待する入力が request に受理されました",
		)
	}
	preprocessed, err := dependencies.Preprocessor().Preprocess(ctx, request)
	if err != nil {
		return legalquery.LegalQueryPlan{}, err
	}
	result, err := dependencies.Profiles().Collect(preprocessed)
	if err != nil {
		return legalquery.LegalQueryPlan{}, err
	}
	packState, err := legalquery.NewStaticPackState(
		[]string{JudicialCasesPackID},
		semanticCase.EnabledPacks(),
	)
	if err != nil {
		return legalquery.LegalQueryPlan{}, err
	}
	plan, err := legalquery.SelectLegalQueryPlan(legalquery.SelectorInput{
		ProfileSetResult: result,
		PackState:        packState,
		LimitPerAttempt:  request.LimitPerAttempt(),
	})
	if err != nil {
		return legalquery.LegalQueryPlan{}, err
	}
	return plan, nil
}

func stage43ProductRequest(raw legalquerycorpus.Request) (legalquery.Request, error) {
	limit := stage43RawRequestLimit(raw)
	base, err := legalquery.NewRequest(legalquery.RequestValues{
		Query:           raw.Query(),
		LimitPerAttempt: limit,
	})
	if err != nil {
		return legalquery.Request{}, err
	}
	rawRef, exists := raw.Ref()
	if !exists {
		return base, nil
	}
	ref, err := stage43ProductRef(rawRef)
	if err != nil {
		return legalquery.Request{}, stage43InvalidRefError()
	}
	return legalquery.NewRequest(legalquery.RequestValues{
		Query:           raw.Query(),
		Ref:             &ref,
		LimitPerAttempt: limit,
	})
}

func stage43RawRequestLimit(request legalquerycorpus.Request) *int {
	limit, exists := request.LimitPerAttempt()
	if !exists {
		return nil
	}
	return &limit
}

func stage43ProductRef(
	rawRef legalquerycorpus.RequestRef,
) (model.SourceResourceRef, error) {
	keyValues := model.SourceResourceKeyValues{
		SourceID:     rawRef.Key().SourceID(),
		ResourceType: rawRef.Key().ResourceType(),
		ResourceID:   rawRef.Key().ResourceID(),
	}
	if versionID, versioned := rawRef.Key().VersionID(); versioned {
		keyValues.VersionID = versionID
	}
	key, err := model.NewSourceResourceKey(keyValues)
	if err != nil {
		return model.SourceResourceRef{}, err
	}
	return model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: rawRef.ProviderID(),
		Key:        key,
	})
}

func stage43InvalidRefError() error {
	err, buildErr := legalquery.NewArgumentError(
		"ref",
		"は有効な SourceResourceRef でなければなりません",
	)
	if buildErr != nil {
		return fmt.Errorf("ref の入力エラーを構築できません: %w", buildErr)
	}
	return err
}

func assertStage43DevelopmentCaseIsRunnable(
	t *testing.T,
	semanticCase legalquerycorpus.SemanticCase,
	plan legalquery.LegalQueryPlan,
) {
	t.Helper()

	if semanticCase.Expected().Kind() !=
		legalquerycorpus.SemanticExpectedKindPlan {
		return
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf(
			"%s: caseId %q の plan が無効です: %v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			semanticCase.CaseID(),
			err,
		)
	}
}

func TestNextProfileSetはDevelopmentOnly入力だけで再現する(t *testing.T) {
	t.Parallel()

	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf(
			"%s: repository path を解決できません: %v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			err,
		)
	}
	developmentOnlyRepository := stage43DevelopmentOnlyRepository(
		t,
		repository,
	)
	entries := stage43DirectoryEntries(
		t,
		filepath.Join(
			developmentOnlyRepository,
			"testdata",
			"legalquery",
			"corpus-v10",
		),
	)
	if !slices.Equal(entries, []string{"development"}) {
		t.Fatalf(
			"%s: copied corpus-v10 entries = %#v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			entries,
		)
	}
	if _, err := legalquerycorpus.LoadDevelopment(
		context.Background(),
		developmentOnlyRepository,
		"testdata/legalquery/corpus-v10/development",
	); err != nil {
		t.Fatalf(
			"%s: development-only repository を読めません: %v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			err,
		)
	}
}

func TestStage43DevelopmentOnlyCopyはSymlinkを拒否する(t *testing.T) {
	t.Parallel()

	source := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte("{}"), 0o600); err != nil {
		t.Fatalf(
			"%s: symlink target を作成できません: %v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			err,
		)
	}
	if err := os.Symlink(outside, filepath.Join(source, "linked.json")); err != nil {
		t.Fatalf(
			"%s: symlink fixture を作成できません: %v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			err,
		)
	}
	if err := stage43CopyTreeFiles(source, t.TempDir()); err == nil {
		t.Fatalf(
			"%s: development-only copy が symlink を受理しました",
			nextProfileSetDevelopmentOnlyCalibrationID,
		)
	}
}

func stage43DevelopmentOnlyRepository(
	t *testing.T,
	repositoryRoot string,
) string {
	t.Helper()

	root := t.TempDir()
	stage43CopyTree(
		t,
		filepath.Join(repositoryRoot, "testdata", "legalquery", "schemas"),
		filepath.Join(root, "testdata", "legalquery", "schemas"),
	)
	stage43CopyTree(
		t,
		filepath.Join(
			repositoryRoot,
			"testdata",
			"legalquery",
			"corpus-v10",
			"development",
		),
		filepath.Join(
			root,
			"testdata",
			"legalquery",
			"corpus-v10",
			"development",
		),
	)
	return root
}

func stage43CopyTree(t *testing.T, source string, target string) {
	t.Helper()

	if err := stage43CopyTreeFiles(source, target); err != nil {
		t.Fatalf(
			"%s: development-only repository を作成できません: %v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			err,
		)
	}
}

func stage43CopyTreeFiles(source string, target string) error {
	return filepath.WalkDir(
		source,
		func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("development-only source に symlink があります")
			}
			relative, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			destination := filepath.Join(target, relative)
			if entry.IsDir() {
				return os.MkdirAll(destination, 0o755)
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf(
					"development-only source に通常 file 以外があります",
				)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(destination, data, 0o644)
		},
	)
}

func stage43DirectoryEntries(t *testing.T, directory string) []string {
	t.Helper()

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf(
			"%s: directory を列挙できません: %v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			err,
		)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	slices.Sort(names)
	return names
}
