package legalquerycandidateworker

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	exactCandidateGoRootEnv      = "JAPANESE_LAW_MCP_EXACT_CANDIDATE_GOROOT"
	exactCandidateModuleCacheEnv = "JAPANESE_LAW_MCP_EXACT_CANDIDATE_GOMODCACHE"
	exactCandidateBuildCacheEnv  = "JAPANESE_LAW_MCP_EXACT_CANDIDATE_GOCACHE"
	exactCandidateTempDirEnv     = "JAPANESE_LAW_MCP_EXACT_CANDIDATE_TMPDIR"
)

// useExactCandidateToolchain は、CI が固定した候補再現用 Go 環境だけを対象 test に渡す。
func useExactCandidateToolchain(t *testing.T) bool {
	t.Helper()

	goRoot := os.Getenv(exactCandidateGoRootEnv)
	if goRoot == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("CI では候補再現用 Go 環境が必須です")
		}
		return false
	}
	moduleCache := requireExactCandidateDirectory(t, exactCandidateModuleCacheEnv)
	buildCache := requireExactCandidateDirectory(t, exactCandidateBuildCacheEnv)
	temporaryDirectory := requireExactCandidateDirectory(t, exactCandidateTempDirEnv)
	goRoot = validateExactCandidateDirectory(t, exactCandidateGoRootEnv, goRoot)
	goBinary := validateExactCandidateGoBinary(t, filepath.Join(goRoot, "bin", "go"))

	pathValue := filepath.Dir(goBinary)
	if inherited := os.Getenv("PATH"); inherited != "" {
		pathValue += string(os.PathListSeparator) + inherited
	}
	t.Setenv("HOME", "")
	t.Setenv("PATH", pathValue)
	t.Setenv("GOROOT", goRoot)
	t.Setenv("GOMODCACHE", moduleCache)
	t.Setenv("GOCACHE", buildCache)
	t.Setenv("TMPDIR", temporaryDirectory)
	return true
}

func requireExactCandidateDirectory(t *testing.T, environmentName string) string {
	t.Helper()
	value := os.Getenv(environmentName)
	if value == "" {
		t.Fatalf("候補再現用の %s が設定されていません", environmentName)
	}
	return validateExactCandidateDirectory(t, environmentName, value)
}

func validateExactCandidateDirectory(t *testing.T, environmentName, path string) string {
	t.Helper()
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		t.Fatalf("候補再現用の %s は正規化済み絶対 path でなければなりません", environmentName)
	}
	information, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("候補再現用の %s を検証できません", environmentName)
	}
	if information.Mode()&os.ModeSymlink != 0 || !information.IsDir() {
		t.Fatalf("候補再現用の %s は symlink ではない directory でなければなりません", environmentName)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != path {
		t.Fatalf("候補再現用の %s の経路に symlink を含められません", environmentName)
	}
	return path
}

func validateExactCandidateGoBinary(t *testing.T, path string) string {
	t.Helper()
	information, err := os.Lstat(path)
	if err != nil {
		t.Fatal("候補再現用の Go executable を検証できません")
	}
	if information.Mode()&os.ModeSymlink != 0 || !information.Mode().IsRegular() {
		t.Fatal("候補再現用の Go executable は symlink ではない通常 file でなければなりません")
	}
	if information.Mode().Perm()&0o111 == 0 {
		t.Fatal("候補再現用の Go executable に実行権限がありません")
	}
	return path
}
