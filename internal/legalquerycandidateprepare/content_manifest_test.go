package legalquerycandidateprepare

import (
	"path/filepath"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
)

func TestBuildContentManifestは校正済み候補と許可辞書だけを固定する(
	t *testing.T,
) {
	t.Parallel()

	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("candidate-evaluation-candidate-content-identity: repository を解決できません: %v", err)
	}
	source := validSourceSetForTest(t)
	manifest, err := BuildContentManifest(t.Context(), repository, source)
	if err != nil {
		t.Fatalf("candidate-evaluation-candidate-content-identity: manifest を構成できません: %v", err)
	}
	if manifest.CandidateContentID == "" ||
		manifest.ProfileSet.ProfileSetID != "default" ||
		manifest.ProfileSet.ProfileSetVersion != "profile-set-sha256-1c8c43fa032148bf0cadd2548afef04b3b0927a9d1a621ef175f5bee781c41a3" ||
		manifest.ProfileSet.RankingVersion != "legal-query-ranking-2026-07-31-2" {
		t.Fatalf("candidate-evaluation-candidate-content-identity: profileSet = %#v", manifest.ProfileSet)
	}
	if len(manifest.ProfileArtifacts) != 2 ||
		manifest.ProfileArtifacts[0].ProfileID != "core" ||
		manifest.ProfileArtifacts[0].ProfileVersion != "core-2026-07-31-37" ||
		manifest.ProfileArtifacts[1].ProfileID != "judicial-cases" ||
		manifest.ProfileArtifacts[1].ProfileVersion != "judicial-cases-2026-07-31-12" {
		t.Fatalf("candidate-evaluation-candidate-content-identity: profiles = %#v", manifest.ProfileArtifacts)
	}
	if len(manifest.LexiconArtifacts) != 2 ||
		manifest.LexiconArtifacts[0].LexiconID != "lawNames" ||
		len(manifest.LexiconArtifacts[0].Files) != 2 ||
		manifest.LexiconArtifacts[1].LexiconID != "legalConcepts" ||
		len(manifest.LexiconArtifacts[1].Files) != 1 {
		t.Fatalf("candidate-evaluation-candidate-content-identity: lexicons = %#v", manifest.LexiconArtifacts)
	}
	canonical, err := legalquerycandidateeval.MarshalCanonicalJSON(manifest)
	if err != nil {
		t.Fatalf("candidate-evaluation-candidate-content-identity: canonical JSON を作れません: %v", err)
	}
	if _, err := legalquerycandidateeval.DecodeCandidateContentManifest(canonical); err != nil {
		t.Fatalf("candidate-evaluation-candidate-content-identity: 自己検証できません: %v", err)
	}
}

func TestBuildContentManifestはSourceSetの一Byte差をIdentityへ反映する(
	t *testing.T,
) {
	t.Parallel()

	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repository を解決できません: %v", err)
	}
	firstSource := validSourceSetForTest(t)
	first, err := BuildContentManifest(t.Context(), repository, firstSource)
	if err != nil {
		t.Fatalf("最初の manifest を構成できません: %v", err)
	}
	secondSource := firstSource
	secondSource.Files = append([]legalquerycandidateeval.FileDigest(nil), firstSource.Files...)
	secondSource.Files[0].RawSHA256 = "1111111111111111111111111111111111111111111111111111111111111111"
	secondSource.SourceSetSHA256, err = legalquerycandidateeval.CanonicalSourceSetSHA256(secondSource)
	if err != nil {
		t.Fatalf("変更 source digest を計算できません: %v", err)
	}
	second, err := BuildContentManifest(t.Context(), repository, secondSource)
	if err != nil {
		t.Fatalf("変更後の manifest を構成できません: %v", err)
	}
	if first.CandidateContentID == second.CandidateContentID {
		t.Fatal("candidate-evaluation-candidate-content-identity: source byte digest の差が ID に反映されません")
	}
}

func TestProfilePackageRootは未知Profileを拒否する(t *testing.T) {
	t.Parallel()

	if _, err := profilePackageRoot("unknown-profile"); err == nil {
		t.Fatal("candidate-evaluation-candidate-content-identity: 未知 profile を受理しました")
	}
}

func validSourceSetForTest(t *testing.T) legalquerycandidateeval.SemanticSourceSet {
	t.Helper()
	source := legalquerycandidateeval.SemanticSourceSet{
		MainModulePath:     "github.com/geonwoo-jeong/japanese-law-mcp",
		GoLanguageVersion:  "1.25.0",
		GoToolchainVersion: "go1.26.5",
		GoDebugSettings:    []legalquerycandidateeval.GoDebugSetting{},
		GOOS:               "linux",
		GOARCH:             "amd64",
		GOAMD64:            "v1",
		GOEXPERIMENT:       "",
		CGOEnabled:         0,
		BuildTags:          []string{},
		Files: []legalquerycandidateeval.FileDigest{{
			Path:      "internal/application/legalquery/selector.go",
			RawSHA256: "0000000000000000000000000000000000000000000000000000000000000000",
		}},
		ModuleDependencies: []legalquerycandidateeval.ModuleDependency{},
	}
	var err error
	source.SourceSetSHA256, err = legalquerycandidateeval.CanonicalSourceSetSHA256(source)
	if err != nil {
		t.Fatalf("source set digest を計算できません: %v", err)
	}
	return source
}
