package judicialcases

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func TestProfileとSelectorはcorpusV4の裁判例二fixtureを満たす(
	t *testing.T,
) {
	t.Parallel()

	profile := mustProfile(t)
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("preprocessor を構築できません: %v", err)
	}
	profileSet, err := legalquery.NewQueryProfileSet(
		[]legalquery.QueryProfile{profile},
	)
	if err != nil {
		t.Fatalf("profile set を構築できません: %v", err)
	}
	corpus, err := legalquerycorpus.Load(
		context.Background(),
		repositoryRoot(t),
		"testdata/legalquery/corpus-v4",
	)
	if err != nil {
		t.Fatalf("corpus-v4 を読み込めません: %v", err)
	}

	wanted := map[string]struct{}{
		"development-pack-disabled": {},
		"development-pack-enabled":  {},
	}
	found := make(map[string]struct{}, len(wanted))
	for _, semanticCase := range corpus.Development() {
		if _, exists := wanted[semanticCase.CaseID()]; !exists {
			continue
		}
		currentCase := semanticCase
		found[currentCase.CaseID()] = struct{}{}
		t.Run(currentCase.CaseID(), func(t *testing.T) {
			request, err := productRequest(currentCase.Request())
			if err != nil {
				t.Fatalf("product request のエラー = %v", err)
			}
			preprocessed, err := preprocessor.Preprocess(
				context.Background(),
				request,
			)
			if err != nil {
				t.Fatalf("Preprocess() のエラー = %v", err)
			}
			result, err := profileSet.Collect(preprocessed)
			if err != nil {
				t.Fatalf("profile set の集約エラー = %v", err)
			}
			packState, err := legalquery.NewStaticPackState(
				[]string{"judicial-cases"},
				currentCase.EnabledPacks(),
			)
			if err != nil {
				t.Fatalf("pack state のエラー = %v", err)
			}
			plan, err := legalquery.SelectLegalQueryPlan(
				legalquery.SelectorInput{
					ProfileSetResult: result,
					PackState:        packState,
					LimitPerAttempt:  request.LimitPerAttempt(),
				},
			)
			if err != nil {
				t.Fatalf("selector のエラー = %v", err)
			}
			evaluation, err := legalqueryeval.EvaluateSemanticPlanCase(
				currentCase,
				plan,
			)
			if err != nil {
				t.Fatalf("semantic plan 評価のエラー = %v", err)
			}
			if !evaluation.PlanOutcomeMatched() ||
				len(evaluation.Meanings()) != 1 ||
				!evaluation.Meanings()[0].SignatureMatched() {
				t.Fatalf(
					"fixture と不一致です: plan=%#v evaluation=%#v",
					plan,
					evaluation,
				)
			}
			evidenceMatched, evidenceApplicable :=
				evaluation.Meanings()[0].EvidenceAssertion()
			if !evidenceApplicable || !evidenceMatched {
				t.Fatalf(
					"fixture evidence と不一致です: %#v",
					evaluation.Meanings()[0],
				)
			}
		})
	}
	if len(found) != len(wanted) {
		t.Fatalf("裁判例 fixture が不足しています: found=%#v", found)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("repository root を解決できません: %v", err)
	}
	return root
}

func productRequest(
	raw legalquerycorpus.Request,
) (legalquery.Request, error) {
	values := legalquery.RequestValues{Query: raw.Query()}
	if limit, exists := raw.LimitPerAttempt(); exists {
		values.LimitPerAttempt = &limit
	}
	if rawRef, exists := raw.Ref(); exists {
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
			return legalquery.Request{}, err
		}
		ref, err := model.NewSourceResourceRef(
			model.SourceResourceRefValues{
				ProviderID: rawRef.ProviderID(),
				Key:        key,
			},
		)
		if err != nil {
			return legalquery.Request{}, err
		}
		values.Ref = &ref
	}
	return legalquery.NewRequest(values)
}
