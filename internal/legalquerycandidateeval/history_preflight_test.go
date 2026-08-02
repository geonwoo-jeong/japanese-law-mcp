package legalquerycandidateeval

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"
)

const (
	verificationSingleHoldoutUse   = "candidate-evaluation-single-holdout-use"
	verificationLeakageExclusion   = "candidate-evaluation-leakage-exclusion"
	verificationLeakageIndexBounds = "candidate-evaluation-leakage-index-bounds"
)

func TestEvaluationPreflightは同じHoldoutの別Evaluationだけを拒否する(t *testing.T) {
	t.Parallel()

	template := validEvaluationRequest(t, manifestWithID(t))
	current := syntheticHistoryRequest(t, template, 1, "current")
	consumed := syntheticConsumedEvaluation(t, template, 2, EvaluationOutcomePassed, "history")
	consumed.Request.HoldoutDigest = current.HoldoutDigest
	consumed.Request.EvaluationID = mustEvaluationID(t, consumed.Request)
	consumed.Result = mustSyntheticEvaluationResult(
		t,
		consumed.Request,
		mustCanonicalJSON(t, consumed.Request),
		syntheticEvaluationReportRaw("same-holdout"),
		EvaluationOutcomePassed,
	)
	if err := CheckEvaluationPreflight(current, []ConsumedEvaluation{consumed}); err == nil {
		t.Fatalf("%s: 同じ holdoutDigest の別 evaluation を受理しました", verificationSingleHoldoutUse)
	}

	replay := ConsumedEvaluation{
		Request: current,
		Result: mustSyntheticEvaluationResult(
			t,
			current,
			mustCanonicalJSON(t, current),
			syntheticEvaluationReportRaw("replay"),
			EvaluationOutcomeFailed,
		),
	}
	if err := CheckEvaluationPreflight(current, []ConsumedEvaluation{replay}); err != nil {
		t.Fatalf("%s: 同じ evaluationId の replay を拒否しました: %v", verificationSingleHoldoutUse, err)
	}
}

func TestEvaluationPreflightはPassedとFailedのLeakage衝突を同じく拒否する(t *testing.T) {
	t.Parallel()

	template := validEvaluationRequest(t, manifestWithID(t))
	sharedDigest := syntheticDigest("shared-leakage", 1)
	current := syntheticHistoryRequest(t, template, 10, "current-leakage")
	current.HoldoutLeakageGroupDigests = []string{sharedDigest}
	current.EvaluationID = mustEvaluationID(t, current)

	for index, outcome := range []string{EvaluationOutcomePassed, EvaluationOutcomeFailed} {
		consumed := syntheticConsumedEvaluation(t, template, 20+index, outcome, "past-leakage")
		consumed.Request.HoldoutLeakageGroupDigests = []string{sharedDigest}
		consumed.Request.EvaluationID = mustEvaluationID(t, consumed.Request)
		consumed.Result = mustSyntheticEvaluationResult(
			t,
			consumed.Request,
			mustCanonicalJSON(t, consumed.Request),
			syntheticEvaluationReportRaw(outcome),
			outcome,
		)
		if err := CheckEvaluationPreflight(current, []ConsumedEvaluation{consumed}); err == nil {
			t.Fatalf("%s: outcome=%s の leakage digest 衝突を受理しました", verificationLeakageExclusion, outcome)
		}
	}

	separate := syntheticConsumedEvaluation(t, template, 30, EvaluationOutcomePassed, "separate")
	if err := CheckEvaluationPreflight(current, []ConsumedEvaluation{separate}); err != nil {
		t.Fatalf("%s: 衝突しない compact index を拒否しました: %v", verificationLeakageExclusion, err)
	}
}

func TestEvaluationPreflightはDigest四百件と履歴四千九十六件を境界にする(t *testing.T) {
	template := validEvaluationRequest(t, manifestWithID(t))
	current := syntheticHistoryRequest(t, template, 100, "bounds-current")
	current.HoldoutLeakageGroupDigests = syntheticSortedDigests("current-groups", 400)
	current.EvaluationID = mustEvaluationID(t, current)
	if err := CheckEvaluationPreflight(current, nil); err != nil {
		t.Fatalf("%s: leakage digest の正常最大 400 件を拒否しました: %v", verificationLeakageIndexBounds, err)
	}

	overDigestLimit := current
	overDigestLimit.HoldoutLeakageGroupDigests = syntheticSortedDigests("too-many-current-groups", 401)
	overDigestLimit.EvaluationID = mustEvaluationID(t, overDigestLimit)
	if err := CheckEvaluationPreflight(overDigestLimit, nil); err == nil {
		t.Fatalf("%s: leakage digest 401 件を受理しました", verificationLeakageIndexBounds)
	}

	history := make([]ConsumedEvaluation, 0, 4096)
	for index := range 4096 {
		outcome := EvaluationOutcomePassed
		if index%2 == 1 {
			outcome = EvaluationOutcomeFailed
		}
		history = append(history, syntheticConsumedEvaluation(t, template, 1000+index, outcome, "bounded-history"))
	}
	currentGroupsBefore := append([]string(nil), current.HoldoutLeakageGroupDigests...)
	firstHistoryGroupsBefore := append([]string(nil), history[0].Request.HoldoutLeakageGroupDigests...)
	if err := CheckEvaluationPreflight(current, history); err != nil {
		t.Fatalf("%s: request/result 履歴の正常最大 4096 件を拒否しました: %v", verificationLeakageIndexBounds, err)
	}
	if !slices.Equal(current.HoldoutLeakageGroupDigests, currentGroupsBefore) ||
		!slices.Equal(history[0].Request.HoldoutLeakageGroupDigests, firstHistoryGroupsBefore) {
		t.Fatalf("%s: preflight が入力履歴を変更しました", verificationImmutableVersion)
	}

	overHistoryLimit := append(
		append([]ConsumedEvaluation(nil), history...),
		syntheticConsumedEvaluation(t, template, 6000, EvaluationOutcomePassed, "bounded-history"),
	)
	if err := CheckEvaluationPreflight(current, overHistoryLimit); err == nil {
		t.Fatalf("%s: request/result 履歴 4097 件を受理しました", verificationLeakageIndexBounds)
	}
}

func TestEvaluationPreflightは消費済みRequestとResultの結合変更を拒否する(t *testing.T) {
	t.Parallel()

	template := validEvaluationRequest(t, manifestWithID(t))
	current := syntheticHistoryRequest(t, template, 7000, "immutable-current")
	consumed := syntheticConsumedEvaluation(t, template, 7001, EvaluationOutcomePassed, "immutable-history")
	consumed.Request.CorpusVersion = "changed-after-result"
	if err := CheckEvaluationPreflight(current, []ConsumedEvaluation{consumed}); err == nil {
		t.Fatalf("%s: result 完成後に変更された request を受理しました", verificationImmutableVersion)
	}
}

func TestEvaluationPreflightは消費済みBaseline予約の再利用を拒否する(t *testing.T) {
	t.Parallel()

	template := validEvaluationRequest(t, manifestWithID(t))
	current := syntheticHistoryRequest(t, template, 7100, "baseline-current")
	consumed := syntheticConsumedEvaluation(t, template, 7101, EvaluationOutcomeFailed, "baseline-history")
	consumed.Request.BaselineVersion = current.BaselineVersion
	consumed.Request.EvaluationID = mustEvaluationID(t, consumed.Request)
	consumed.Result = mustSyntheticEvaluationResult(
		t,
		consumed.Request,
		mustCanonicalJSON(t, consumed.Request),
		syntheticEvaluationReportRaw("baseline-reservation"),
		EvaluationOutcomeFailed,
	)
	if err := CheckEvaluationPreflight(current, []ConsumedEvaluation{consumed}); err == nil {
		t.Fatalf("%s: 消費済み baselineVersion の再利用を受理しました", verificationImmutableVersion)
	}
}

func TestRequestReservationPreflightは新しいV2CurrentのV2予約衝突を拒否する(
	t *testing.T,
) {
	t.Parallel()

	template := validEvaluationRequest(t, manifestWithID(t))
	current := syntheticHistoryRequest(t, template, 7200, "v2-current")
	reserved := syntheticHistoryRequest(t, template, 7201, "v2-reserved")
	reserved.HoldoutDigest = current.HoldoutDigest
	reserved.EvaluationID = mustEvaluationID(t, reserved)
	requests := map[string]loadedArtifact[EvaluationRequest]{
		current.EvaluationID:  {document: current},
		reserved.EvaluationID: {document: reserved},
	}

	if err := checkRequestReservationPreflight(current, requests); err == nil {
		t.Fatal("candidate-evaluation-schema-v3-superseded-request-reservation: 新しい v2 current による v2 予約値の再利用を受理しました")
	}
}

func TestRequestReservationPreflightは過去V2同士の衝突だけを遡及無効化しない(
	t *testing.T,
) {
	t.Parallel()

	v2Template := validEvaluationRequest(t, manifestWithID(t))
	first := syntheticHistoryRequest(t, v2Template, 7300, "legacy-first")
	second := syntheticHistoryRequest(t, v2Template, 7301, "legacy-second")
	second.HoldoutDigest = first.HoldoutDigest
	second.HoldoutLeakageGroupDigests = append(
		[]string(nil),
		first.HoldoutLeakageGroupDigests...,
	)
	second.EvaluationID = mustEvaluationID(t, second)

	v3Manifest := validCandidateManifestForSchema(t, SchemaVersionV3)
	current := validEvaluationRequestForSchema(t, v3Manifest)
	requests := map[string]loadedArtifact[EvaluationRequest]{
		first.EvaluationID:   {document: first},
		second.EvaluationID:  {document: second},
		current.EvaluationID: {document: current},
	}

	if err := checkRequestReservationPreflight(current, requests); err != nil {
		t.Fatalf("過去 v2 request 間だけの衝突を遡及的に拒否しました: %v", err)
	}
}

func TestRequestReservationPreflightは既知V2評価二件だけを遡及除外する(
	t *testing.T,
) {
	t.Parallel()

	first := loadRepositoryEvaluationRequest(
		t,
		legacyV2EvaluationIDDefault2,
	)
	second := loadRepositoryEvaluationRequest(
		t,
		legacyV2EvaluationIDDefault3,
	)

	requests := map[string]loadedArtifact[EvaluationRequest]{
		first.EvaluationID:  {document: first},
		second.EvaluationID: {document: second},
	}
	if err := checkRequestReservationPreflight(second, requests); err != nil {
		t.Fatalf("既存の default-2/default-3 評価二件に対する遡及除外を拒否しました: %v", err)
	}

	newCurrent := second
	newCurrent.BaselineVersion = "default-8"
	newCurrent.EvaluationID = mustEvaluationID(t, newCurrent)
	requests[newCurrent.EvaluationID] = loadedArtifact[EvaluationRequest]{document: newCurrent}
	if err := checkRequestReservationPreflight(newCurrent, requests); err == nil {
		t.Fatal("既存二件だけの遡及除外を新しい v2 current へ拡張して受理しました")
	}
}

func syntheticConsumedEvaluation(
	t *testing.T,
	template EvaluationRequest,
	index int,
	outcome string,
	domain string,
) ConsumedEvaluation {
	t.Helper()
	request := syntheticHistoryRequest(t, template, index, domain)
	requestRaw := mustCanonicalJSON(t, request)
	result := mustSyntheticEvaluationResult(
		t,
		request,
		requestRaw,
		syntheticEvaluationReportRaw(fmt.Sprintf("%s-%d-%s", domain, index, outcome)),
		outcome,
	)
	return ConsumedEvaluation{Request: request, Result: result}
}

func syntheticHistoryRequest(
	t *testing.T,
	template EvaluationRequest,
	index int,
	domain string,
) EvaluationRequest {
	t.Helper()
	request := template
	request.CorpusVersion = fmt.Sprintf("synthetic-%s-%d", domain, index)
	request.HoldoutDigest = syntheticDigest(domain+"-holdout", index)
	request.HoldoutLeakageGroupDigests = []string{syntheticDigest(domain+"-group", index)}
	request.BaselineVersion = fmt.Sprintf("default-%d", index+2)
	request.EvaluationID = mustEvaluationID(t, request)
	return request
}

func syntheticSortedDigests(domain string, count int) []string {
	digests := make([]string, 0, count)
	for index := range count {
		digests = append(digests, syntheticDigest(domain, index))
	}
	sort.Strings(digests)
	return digests
}

func syntheticDigest(domain string, index int) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", domain, index)))
	return fmt.Sprintf("%x", digest)
}

func loadRepositoryEvaluationRequest(t *testing.T, evaluationID string) EvaluationRequest {
	t.Helper()
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repository root を解決できません: %v", err)
	}
	requestPath := filepath.Join(
		repositoryRoot,
		"testdata",
		"legalquery",
		"candidate-evaluations",
		"requests",
		evaluationID+".json",
	)
	//nolint:gosec // 固定した repository 内 request だけを読む。
	raw, err := os.ReadFile(requestPath)
	if err != nil {
		t.Fatalf("既存 request を読めません: %v", err)
	}
	request, err := DecodeEvaluationRequest(raw)
	if err != nil {
		t.Fatalf("既存 request が不正です: %v", err)
	}
	return request
}
