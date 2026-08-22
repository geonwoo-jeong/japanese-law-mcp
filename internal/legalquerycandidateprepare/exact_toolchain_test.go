package legalquerycandidateprepare

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	exactCandidateGoRootEnv      = "JAPANESE_LAW_MCP_EXACT_CANDIDATE_GOROOT"
	exactCandidateModuleCacheEnv = "JAPANESE_LAW_MCP_EXACT_CANDIDATE_GOMODCACHE"
	exactCandidateBuildCacheEnv  = "JAPANESE_LAW_MCP_EXACT_CANDIDATE_GOCACHE"
	exactCandidateTempEnv        = "JAPANESE_LAW_MCP_EXACT_CANDIDATE_TMPDIR"
)

type exactCandidateToolchainEnvironment struct {
	goRoot      string
	moduleCache string
	buildCache  string
	temporary   string
}

// useExactCandidateToolchain は、CI が準備した候補再現用の閉じた Go 環境だけを使用する。
func useExactCandidateToolchain(t *testing.T) bool {
	t.Helper()
	infrastructure, enabled := loadExactCandidateToolchainEnvironment(t)
	if !enabled {
		if os.Getenv("CI") != "" {
			t.Fatal("CI では候補再現用 Go 環境が必須です")
		}
		return false
	}
	goBinary := requireExactCandidateGoBinary(t, infrastructure.goRoot)
	goBinaryDirectory := filepath.Dir(goBinary)
	path := goBinaryDirectory
	if inherited := os.Getenv("PATH"); inherited != "" {
		path += string(os.PathListSeparator) + inherited
	}
	t.Setenv("HOME", "")
	t.Setenv("PATH", path)
	t.Setenv("GOROOT", infrastructure.goRoot)
	t.Setenv("GOMODCACHE", infrastructure.moduleCache)
	t.Setenv("GOCACHE", infrastructure.buildCache)
	t.Setenv("TMPDIR", infrastructure.temporary)
	return true
}

func loadExactCandidateToolchainEnvironment(t *testing.T) (exactCandidateToolchainEnvironment, bool) {
	t.Helper()
	goRoot := os.Getenv(exactCandidateGoRootEnv)
	if goRoot == "" {
		return exactCandidateToolchainEnvironment{}, false
	}
	infrastructure := exactCandidateToolchainEnvironment{
		goRoot:      goRoot,
		moduleCache: os.Getenv(exactCandidateModuleCacheEnv),
		buildCache:  os.Getenv(exactCandidateBuildCacheEnv),
		temporary:   os.Getenv(exactCandidateTempEnv),
	}
	requireExactCandidateDirectory(t, exactCandidateGoRootEnv, infrastructure.goRoot)
	requireExactCandidateDirectory(t, exactCandidateModuleCacheEnv, infrastructure.moduleCache)
	requireExactCandidateDirectory(t, exactCandidateBuildCacheEnv, infrastructure.buildCache)
	requireExactCandidateDirectory(t, exactCandidateTempEnv, infrastructure.temporary)
	return infrastructure, true
}

func requireExactCandidateDirectory(t *testing.T, name string, value string) {
	t.Helper()
	if value == "" || !filepath.IsAbs(value) || filepath.Clean(value) != value {
		t.Fatalf("%s は空でない正規化済み絶対 path でなければなりません", name)
	}
	info, err := os.Lstat(value)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		t.Fatalf("%s は symlink ではない directory でなければなりません", name)
	}
	resolved, err := filepath.EvalSymlinks(value)
	if err != nil || resolved != value {
		t.Fatalf("%s の経路に symlink を含められません", name)
	}
}

func requireExactCandidateGoBinary(t *testing.T, goRoot string) string {
	t.Helper()
	goBinary := filepath.Join(goRoot, "bin", "go")
	info, err := os.Lstat(goBinary)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		t.Fatal("候補再現用 Go executable は symlink ではない実行可能な regular file でなければなりません")
	}
	return goBinary
}
