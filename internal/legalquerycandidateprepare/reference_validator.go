package legalquerycandidateprepare

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryadoption"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval/evaluators"
)

// ReferenceValidator は、候補 subtree 外の固定参照を repository から再構築する。
type ReferenceValidator struct {
	repositoryRoot string
}

// ValidateEvaluatorVersion は、履歴 replay を含む exact version の実装有無を確認する。
func (v ReferenceValidator) ValidateEvaluatorVersion(version string) error {
	if !evaluators.IsSupported(version) {
		return fmt.Errorf("evaluatorVersion が閉じた evaluator registry に存在しません")
	}
	return nil
}

// NewReferenceValidator は、root-scoped repository 境界を検証して返す。
func NewReferenceValidator(repositoryRoot string) (ReferenceValidator, error) {
	repository, err := openPrepareRepository(repositoryRoot)
	if err != nil {
		return ReferenceValidator{}, err
	}
	if err := repository.Close(); err != nil {
		return ReferenceValidator{}, err
	}
	return ReferenceValidator{repositoryRoot: repositoryRoot}, nil
}

// ValidateCandidateContent は、source closure を含む manifest 全体を再構築する。
func (v ReferenceValidator) ValidateCandidateContent(
	ctx context.Context,
	raw []byte,
	document legalquerycandidateeval.CandidateContentManifest,
) error {
	if err := checkPrepareContext(ctx); err != nil {
		return err
	}
	if v.repositoryRoot == "" {
		return fmt.Errorf("候補参照 validator が初期化されていません")
	}
	sourceSet, err := BuildSemanticSourceSet(ctx, v.repositoryRoot)
	if err != nil {
		return err
	}
	expected, err := BuildContentManifest(ctx, v.repositoryRoot, sourceSet)
	if err != nil {
		return err
	}
	expectedRaw, err := legalquerycandidateeval.MarshalCanonicalJSON(expected)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(document, expected) || !bytes.Equal(raw, expectedRaw) {
		return fmt.Errorf("candidate content が現在の閉じた参照集合と一致しません")
	}
	return nil
}

// ValidateEvaluationRequest は、holdout fixture を開かず manifest と SOT を照合する。
func (v ReferenceValidator) ValidateEvaluationRequest(
	ctx context.Context,
	_ []byte,
	document legalquerycandidateeval.EvaluationRequest,
) (legalquerycandidateeval.RequestReferenceValidation, error) {
	if err := checkPrepareContext(ctx); err != nil {
		return legalquerycandidateeval.RequestReferenceValidation{}, err
	}
	if v.repositoryRoot == "" {
		return legalquerycandidateeval.RequestReferenceValidation{},
			fmt.Errorf("候補参照 validator が初期化されていません")
	}
	if err := v.ValidateEvaluatorVersion(document.EvaluatorVersion); err != nil {
		return legalquerycandidateeval.RequestReferenceValidation{},
			err
	}
	if document.EvaluatorVersion != evaluators.CurrentVersion {
		return legalquerycandidateeval.RequestReferenceValidation{},
			fmt.Errorf("新しい evaluation request は current evaluatorVersion を必要とします")
	}
	adoption, err := legalqueryadoption.LoadCurrentFromRoot(ctx, v.repositoryRoot)
	if err != nil {
		return legalquerycandidateeval.RequestReferenceValidation{}, err
	}
	if document.BaselineVersion == adoption.BaselineVersion() {
		return legalquerycandidateeval.RequestReferenceValidation{},
			fmt.Errorf("現行 baselineVersion は候補予約に使用できません")
	}
	references, err := BuildRequiredSOTReferences(ctx, v.repositoryRoot)
	if err != nil {
		return legalquerycandidateeval.RequestReferenceValidation{}, err
	}
	if !reflect.DeepEqual(document.RequiredReviewSOTs, references) ||
		document.RequiredReviewSOTSetSHA256 != legalquerycandidateeval.SOTSetSHA256(references) {
		return legalquerycandidateeval.RequestReferenceValidation{},
			fmt.Errorf("evaluation request の SOT 参照が現在の有効集合と一致しません")
	}
	corpus, err := legalquerycorpus.LoadManifest(
		ctx,
		v.repositoryRoot,
		"testdata/legalquery/"+document.CorpusVersion,
	)
	if err != nil {
		return legalquerycandidateeval.RequestReferenceValidation{}, err
	}
	manifest := corpus.Manifest()
	if document.CorpusVersion != manifest.CorpusVersion() ||
		document.CorpusManifestSHA256 != corpus.SHA256() ||
		document.HoldoutDigest != manifest.HoldoutDigest() ||
		!reflect.DeepEqual(
			document.HoldoutLeakageGroupDigests,
			manifest.HoldoutLeakageGroupDigests(),
		) {
		return legalquerycandidateeval.RequestReferenceValidation{},
			fmt.Errorf("evaluation request が corpus manifest と一致しません")
	}
	return legalquerycandidateeval.RequestReferenceValidation{
		CurrentRequiredReviewSOTs: cloneSOTReferences(references),
	}, nil
}
