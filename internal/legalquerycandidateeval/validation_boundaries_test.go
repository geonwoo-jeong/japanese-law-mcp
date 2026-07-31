package legalquerycandidateeval

import (
	"fmt"
	"testing"
)

func TestProfileArtifactの各境界を拒否する(t *testing.T) {
	t.Parallel()

	valid := ProfileArtifact{
		ProfileID:               "core",
		ProfileVersion:          "core-v1",
		MetadataSchemaVersion:   1,
		MetadataCanonicalSHA256: repeatHex('1'),
		CueSetVersion:           "core-cues-v1",
		CueArtifactSHA256:       repeatHex('2'),
	}
	tests := []struct {
		name   string
		mutate func(*ProfileArtifact)
	}{
		{name: "profile-version", mutate: func(value *ProfileArtifact) { value.ProfileVersion = "" }},
		{name: "metadata-version", mutate: func(value *ProfileArtifact) { value.MetadataSchemaVersion = 0 }},
		{name: "metadata-digest", mutate: func(value *ProfileArtifact) { value.MetadataCanonicalSHA256 = "bad" }},
		{name: "cue-version", mutate: func(value *ProfileArtifact) { value.CueSetVersion = "" }},
		{name: "cue-digest", mutate: func(value *ProfileArtifact) { value.CueArtifactSHA256 = "bad" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := valid
			test.mutate(&value)
			if err := validateProfileArtifact(value); err == nil {
				t.Fatal("不正な profile artifact を受理しました")
			}
		})
	}
}

func TestGoDebugSettingsの順序と値を検証する(t *testing.T) {
	t.Parallel()

	if err := validateGoDebugSettings([]GoDebugSetting{
		{Name: "asynctimerchan", Value: "1"},
		{Name: "http2client", Value: "0"},
	}); err != nil {
		t.Fatalf("正しい GODEBUG 設定を拒否しました: %v", err)
	}
	for _, settings := range [][]GoDebugSetting{
		nil,
		{{Name: "http2client", Value: "0"}, {Name: "asynctimerchan", Value: "1"}},
		{{Name: "", Value: "0"}},
		{{Name: "http2client", Value: ""}},
	} {
		if err := validateGoDebugSettings(settings); err == nil {
			t.Fatal("不正な GODEBUG 設定を受理しました")
		}
	}
}

func TestModuleDependencyの各境界を拒否する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*ModuleDependency)
	}{
		{name: "module-path", mutate: func(value *ModuleDependency) { value.ModulePath = "" }},
		{name: "version", mutate: func(value *ModuleDependency) { value.Version = "" }},
		{name: "zip-sum", mutate: func(value *ModuleDependency) { value.ModuleZipSum = "bad" }},
		{name: "zip-digest", mutate: func(value *ModuleDependency) { value.ModuleZipRawSHA256 = "bad" }},
		{name: "zip-size", mutate: func(value *ModuleDependency) { value.ModuleZipByteLength = 0 }},
		{name: "gomod-sum", mutate: func(value *ModuleDependency) { value.ModuleGoModSum = "bad" }},
		{name: "gomod-digest", mutate: func(value *ModuleDependency) { value.ModuleGoModRawSHA256 = "bad" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validModuleDependency()
			test.mutate(&value)
			if err := validateModuleDependency(value); err == nil {
				t.Fatal("不正な module dependency を受理しました")
			}
		})
	}
	if err := validateModuleDependency(validModuleDependency()); err != nil {
		t.Fatalf("正しい module dependency を拒否しました: %v", err)
	}
}

func TestRequestの各閉じた境界を拒否する(t *testing.T) {
	t.Parallel()

	manifest := manifestWithID(t)
	tests := []struct {
		name   string
		mutate func(*EvaluationRequest)
	}{
		{name: "evaluator", mutate: func(value *EvaluationRequest) { value.EvaluatorVersion = "legal-query-evaluator-v0" }},
		{name: "corpus", mutate: func(value *EvaluationRequest) { value.CorpusVersion = "" }},
		{name: "corpus-digest", mutate: func(value *EvaluationRequest) { value.CorpusManifestSHA256 = "bad" }},
		{name: "holdout-digest", mutate: func(value *EvaluationRequest) { value.HoldoutDigest = "bad" }},
		{name: "candidate-id", mutate: func(value *EvaluationRequest) { value.CandidateContentID = "bad" }},
		{name: "rubric", mutate: func(value *EvaluationRequest) { value.ReviewRubricSHA256 = "bad" }},
		{name: "sot-set", mutate: func(value *EvaluationRequest) { value.RequiredReviewSOTSetSHA256 = "bad" }},
		{name: "baseline", mutate: func(value *EvaluationRequest) { value.BaselineVersion = "default-0" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validEvaluationRequest(t, manifest)
			test.mutate(&value)
			value.EvaluationID = mustEvaluationID(t, value)
			if err := validateEvaluationRequest(value); err == nil {
				t.Fatal("不正な request を受理しました")
			}
		})
	}
}

func TestCanonicalEncodingは許可外の型とnil配列を拒否する(t *testing.T) {
	t.Parallel()

	if _, err := canonicalBytes(true, ""); err == nil {
		t.Fatal("bool canonical 入力を受理しました")
	}
	var values []string
	if _, err := canonicalBytes(values, ""); err == nil {
		t.Fatal("nil array canonical 入力を受理しました")
	}
}

func TestInMemoryValidationはSchemaのversionCountArchive上限と一致する(t *testing.T) {
	t.Parallel()

	sourceSet := validCandidateManifest(t).SemanticSourceSet
	sourceSet.GoLanguageVersion = "go1.25"
	if err := validateSourceBuildContext(sourceSet); err == nil {
		t.Fatal("schema 外の goLanguageVersion を in-memory 検証が受理しました")
	}

	settings := make([]GoDebugSetting, 0, 129)
	for index := range 129 {
		settings = append(settings, GoDebugSetting{
			Name: fmt.Sprintf("setting%03d", index), Value: "0",
		})
	}
	if err := validateGoDebugSettings(settings); err == nil {
		t.Fatal("schema 上限を超える goDebugSettings を in-memory 検証が受理しました")
	}

	dependency := validModuleDependency()
	dependency.ModuleZipByteLength = (64 << 20) + 1
	if err := validateModuleDependency(dependency); err == nil {
		t.Fatal("schema 上限を超える module zip を in-memory 検証が受理しました")
	}
}
