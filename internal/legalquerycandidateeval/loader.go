package legalquerycandidateeval

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
)

const (
	maximumManifestFiles      = 4096
	maximumManifestTotalBytes = 64 << 20
	maximumAttestationFiles   = 8192
	maximumAttestationTotal   = 64 << 20
	maximumRequestFiles       = 4096
	maximumRequestTotalBytes  = 32 << 20
	maximumResultFiles        = 4096
	maximumResultTotalBytes   = 32 << 20
	maximumFailedReportFiles  = 4096
	maximumFailedReportTotal  = 256 << 20
)

// ReferenceValidator は subtree 外にある内容固定参照の検証境界である。
// evaluator version は履歴 replay を含む全 request に対して検証する。
// raw は呼出しごとに複製され、実装は保持または変更せずに検証する。
type ReferenceValidator interface {
	ValidateEvaluatorVersion(string) error
	ValidateCandidateContent(context.Context, []byte, CandidateContentManifest) error
	ValidateEvaluationRequest(
		context.Context,
		[]byte,
		EvaluationRequest,
	) (RequestReferenceValidation, error)
}

// RequestReferenceValidation は current SOT index から独立に解決した参照集合を返す。
// validator は request 内の集合を複製せず、repository の index と原 byte から構成する。
type RequestReferenceValidation struct {
	CurrentRequiredReviewSOTs []SOTReference
}

// PreparedCurrent は評価前の current request と内容固定済み参照を保持する。
type PreparedCurrent struct {
	Pointer            PointerDocument
	Request            EvaluationRequest
	CandidateContent   CandidateContentManifest
	ReviewAttestations []ReviewAttestation
}

// LoadPreparedCurrent は result が存在しない評価準備 subtree を全体検証する。
func LoadPreparedCurrent(
	ctx context.Context,
	repositoryRoot string,
	referenceValidator ReferenceValidator,
) (PreparedCurrent, error) {
	if err := checkContext(ctx); err != nil {
		return PreparedCurrent{}, err
	}
	if referenceValidator == nil {
		return PreparedCurrent{}, fmt.Errorf("外部参照 validator が指定されていません")
	}
	repository, err := legalqueryartifact.OpenRepository(repositoryRoot)
	if err != nil {
		return PreparedCurrent{}, fmt.Errorf("candidate evaluation repository を開けません: %w", err)
	}
	defer func() { _ = repository.Close() }()
	root, err := openCandidateEvaluationRoot(repository)
	if err != nil {
		return PreparedCurrent{}, err
	}
	defer func() { _ = root.Close() }()
	return loadPreparedCurrentFromRoot(ctx, root, referenceValidator)
}

func loadPreparedCurrentFromRoot(
	ctx context.Context,
	root *legalqueryartifact.Repository,
	referenceValidator ReferenceValidator,
) (PreparedCurrent, error) {
	layout, err := validateRootEntries(root)
	if err != nil {
		return PreparedCurrent{}, err
	}
	schema, err := loadSchema(root)
	if err != nil {
		return PreparedCurrent{}, err
	}
	pointer, err := loadPointer(ctx, root, schema)
	if err != nil {
		return PreparedCurrent{}, err
	}
	artifacts, err := loadPreparationArtifacts(ctx, root, schema, layout, true)
	if err != nil {
		return PreparedCurrent{}, err
	}
	if err := validatePreparationBindings(artifacts); err != nil {
		return PreparedCurrent{}, err
	}
	if err := validateEvaluatorVersions(artifacts.requests, referenceValidator); err != nil {
		return PreparedCurrent{}, err
	}
	current, exists := artifacts.requests[pointer.EvaluationID]
	if !exists {
		return PreparedCurrent{}, fmt.Errorf("current evaluation request が存在しません")
	}
	if err := validateExternalReferences(ctx, current.document, artifacts, referenceValidator); err != nil {
		return PreparedCurrent{}, err
	}
	return prepareCurrent(pointer, current.document, artifacts), nil
}

func validateEvaluatorVersions(
	requests map[string]loadedArtifact[EvaluationRequest],
	validator ReferenceValidator,
) error {
	for _, evaluationID := range sortedKeys(requests) {
		version := requests[evaluationID].document.EvaluatorVersion
		if err := validator.ValidateEvaluatorVersion(version); err != nil {
			return fmt.Errorf(
				"evaluationId %q の evaluatorVersion が未対応です: %w",
				evaluationID,
				err,
			)
		}
	}
	return nil
}

type preparationArtifacts struct {
	manifests    map[string]loadedArtifact[CandidateContentManifest]
	attestations map[string]loadedArtifact[ReviewAttestation]
	requests     map[string]loadedArtifact[EvaluationRequest]
}

type loadedArtifact[T any] struct {
	document T
	raw      []byte
	digest   string
}

func loadPreparationArtifacts(
	ctx context.Context,
	root *legalqueryartifact.Repository,
	schema SchemaV2,
	layout preparationRootLayout,
	preparationOnly bool,
) (preparationArtifacts, error) {
	manifests, err := loadArtifactDirectory(
		ctx, root, "content-manifests", schema,
		maximumManifestFiles, maximumManifestTotalBytes, maximumManifestBytes,
		candidateContentIDPattern, DecodeCandidateContentManifest,
		func(value CandidateContentManifest) string { return value.CandidateContentID },
	)
	if err != nil {
		return preparationArtifacts{}, err
	}
	attestations, err := loadArtifactDirectory(
		ctx, root, "review-attestations", schema,
		maximumAttestationFiles, maximumAttestationTotal, maximumAttestationBytes,
		reviewAttestationIDPattern, DecodeReviewAttestation,
		func(value ReviewAttestation) string { return value.AttestationID },
	)
	if err != nil {
		return preparationArtifacts{}, err
	}
	requests, err := loadArtifactDirectory(
		ctx, root, "requests", schema,
		maximumRequestFiles, maximumRequestTotalBytes, maximumRequestBytes,
		evaluationIDPattern, DecodeEvaluationRequest,
		func(value EvaluationRequest) string { return value.EvaluationID },
	)
	if err != nil {
		return preparationArtifacts{}, err
	}
	if len(manifests) == 0 || len(attestations) == 0 || len(requests) == 0 {
		return preparationArtifacts{}, fmt.Errorf("candidate evaluation 成果物が空です")
	}
	if preparationOnly {
		if err := requireEmptyHistory(root, layout); err != nil {
			return preparationArtifacts{}, err
		}
	}
	return preparationArtifacts{manifests: manifests, attestations: attestations, requests: requests}, nil
}

func loadArtifactDirectory[T any](
	ctx context.Context,
	root *legalqueryartifact.Repository,
	directoryName string,
	schema SchemaV2,
	maximumFiles int,
	maximumTotalBytes int64,
	maximumFileBytes int,
	idPattern *regexp.Regexp,
	decode func([]byte) (T, error),
	identity func(T) string,
) (map[string]loadedArtifact[T], error) {
	directory, err := root.OpenChild(directoryName)
	if err != nil {
		return nil, fmt.Errorf("%s directory を開けません: %w", directoryName, err)
	}
	defer func() { _ = directory.Close() }()
	entries, err := directory.ReadDirectory(maximumFiles, maximumTotalBytes)
	if err != nil {
		return nil, fmt.Errorf("%s directory を列挙できません: %w", directoryName, err)
	}
	loaded := make(map[string]loadedArtifact[T], len(entries))
	for _, entry := range entries {
		if err := checkContext(ctx); err != nil {
			return nil, err
		}
		fileID, err := validateArtifactEntry(entry, idPattern, maximumFileBytes)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", directoryName, err)
		}
		raw, err := directory.ReadRegular(entry.Name(), int64(maximumFileBytes))
		if err != nil {
			return nil, fmt.Errorf("%s 成果物を読めません: %w", directoryName, err)
		}
		if err := schema.Validate(ctx, raw); err != nil {
			return nil, err
		}
		document, err := decode(raw)
		if err != nil {
			return nil, err
		}
		if identity(document) != fileID {
			return nil, fmt.Errorf("%s の内部 ID が file 名と一致しません", directoryName)
		}
		loaded[fileID] = loadedArtifact[T]{document: document, raw: raw, digest: RawSHA256(raw)}
	}
	return loaded, nil
}

func validateArtifactEntry(
	entry legalqueryartifact.DirectoryEntry,
	idPattern *regexp.Regexp,
	maximumBytes int,
) (string, error) {
	name := entry.Name()
	info := entry.Info()
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() ||
		info.Size() < 1 || info.Size() > int64(maximumBytes) || !strings.HasSuffix(name, ".json") {
		return "", fmt.Errorf("artifact entry が通常 JSON file ではありません")
	}
	id := strings.TrimSuffix(name, ".json")
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("artifact file 名の ID が不正です")
	}
	return id, nil
}

func validatePreparationBindings(artifacts preparationArtifacts) error {
	usedManifests := make(map[string]struct{}, len(artifacts.manifests))
	usedAttestations := make(map[string]struct{}, len(artifacts.attestations))
	baselineVersions := make(map[string]struct{}, len(artifacts.requests))
	for _, requestID := range sortedKeys(artifacts.requests) {
		request := artifacts.requests[requestID].document
		if _, duplicate := baselineVersions[request.BaselineVersion]; duplicate {
			return fmt.Errorf("baselineVersion を複数 request で予約できません")
		}
		baselineVersions[request.BaselineVersion] = struct{}{}
		if err := bindRequest(request, artifacts, usedManifests, usedAttestations); err != nil {
			return err
		}
	}
	if len(usedManifests) != len(artifacts.manifests) ||
		len(usedAttestations) != len(artifacts.attestations) {
		return fmt.Errorf("request から参照されない孤立成果物があります")
	}
	return nil
}

func bindRequest(
	request EvaluationRequest,
	artifacts preparationArtifacts,
	usedManifests map[string]struct{},
	usedAttestations map[string]struct{},
) error {
	manifest, exists := artifacts.manifests[request.CandidateContentID]
	if !exists || manifest.digest != request.CandidateContentManifestSHA256 {
		return fmt.Errorf("request の candidate content binding が一致しません")
	}
	usedManifests[request.CandidateContentID] = struct{}{}
	authorities := make(map[string]struct{}, 2)
	for _, reference := range request.ReviewAttestations {
		attestation, exists := artifacts.attestations[reference.AttestationID]
		if !exists || attestation.digest != reference.AttestationSHA256 {
			return fmt.Errorf("request の review attestation binding が一致しません")
		}
		if err := bindAttestation(request, reference, attestation.document); err != nil {
			return err
		}
		if _, duplicate := authorities[attestation.document.ReviewerAuthorityID]; duplicate {
			return fmt.Errorf("二 scope の reviewerAuthorityId は異なる必要があります")
		}
		authorities[attestation.document.ReviewerAuthorityID] = struct{}{}
		usedAttestations[reference.AttestationID] = struct{}{}
	}
	return nil
}

func bindAttestation(
	request EvaluationRequest,
	reference ReviewAttestationReference,
	attestation ReviewAttestation,
) error {
	if attestation.ReviewScope != reference.ReviewScope ||
		attestation.CandidateContentID != request.CandidateContentID ||
		attestation.CandidateContentManifestSHA256 != request.CandidateContentManifestSHA256 ||
		attestation.RubricVersion != request.ReviewRubricVersion ||
		attestation.RubricSHA256 != request.ReviewRubricSHA256 ||
		attestation.ReviewedSOTSetSHA256 != request.RequiredReviewSOTSetSHA256 ||
		!slices.Equal(attestation.ReviewedSOTs, request.RequiredReviewSOTs) {
		return fmt.Errorf("attestation と request の内容 binding が一致しません")
	}
	return nil
}

func validateExternalReferences(
	ctx context.Context,
	currentRequest EvaluationRequest,
	artifacts preparationArtifacts,
	validator ReferenceValidator,
) error {
	currentManifest, exists := artifacts.manifests[currentRequest.CandidateContentID]
	if !exists {
		return fmt.Errorf("current request の candidate content が存在しません")
	}
	manifestDocument, err := DecodeCandidateContentManifest(currentManifest.raw)
	if err != nil {
		return fmt.Errorf("検証済み candidate content を複製できません: %w", err)
	}
	if err := validator.ValidateCandidateContent(
		ctx, bytes.Clone(currentManifest.raw), manifestDocument,
	); err != nil {
		return fmt.Errorf("candidate content の外部参照検証に失敗しました: %w", err)
	}
	currentRequestArtifact, exists := artifacts.requests[currentRequest.EvaluationID]
	if !exists {
		return fmt.Errorf("current evaluation request が存在しません")
	}
	requestDocument, err := DecodeEvaluationRequest(currentRequestArtifact.raw)
	if err != nil {
		return fmt.Errorf("検証済み evaluation request を複製できません: %w", err)
	}
	validation, err := validator.ValidateEvaluationRequest(
		ctx, bytes.Clone(currentRequestArtifact.raw), requestDocument,
	)
	if err != nil {
		return fmt.Errorf("evaluation request の外部参照検証に失敗しました: %w", err)
	}
	if err := validateCurrentSOTBinding(requestDocument, validation); err != nil {
		return err
	}
	return nil
}

func validateCurrentSOTBinding(
	request EvaluationRequest,
	validation RequestReferenceValidation,
) error {
	references := validation.CurrentRequiredReviewSOTs
	if err := validateSOTReferences(references, true); err != nil {
		return fmt.Errorf("current SOT index の検証結果が不正です: %w", err)
	}
	if !slices.Equal(references, request.RequiredReviewSOTs) ||
		SOTSetSHA256(references) != request.RequiredReviewSOTSetSHA256 {
		return fmt.Errorf("request の review SOT 集合が current SOT 原 byte と一致しません")
	}
	return nil
}

func prepareCurrent(
	pointer PointerDocument,
	request EvaluationRequest,
	artifacts preparationArtifacts,
) PreparedCurrent {
	manifest := artifacts.manifests[request.CandidateContentID].document
	attestations := make([]ReviewAttestation, 0, len(request.ReviewAttestations))
	for _, reference := range request.ReviewAttestations {
		attestations = append(attestations, artifacts.attestations[reference.AttestationID].document)
	}
	return PreparedCurrent{
		Pointer: pointer, Request: request, CandidateContent: manifest, ReviewAttestations: attestations,
	}
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
