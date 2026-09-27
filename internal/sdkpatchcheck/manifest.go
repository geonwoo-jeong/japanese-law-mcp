// Package sdkpatchcheck は、SOT-ENG-052 の固定 SDK 複製と許可差分を照合する。
package sdkpatchcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
)

const (
	// ModulePath は、複製元 SDK の module path である。
	ModulePath = "github.com/modelcontextprotocol/go-sdk"
	// UpstreamVersion は、version 限定置換する複製元の版である。
	UpstreamVersion     = "v1.6.1"
	sourceDirectory     = "third_party/modelcontextprotocol-go-sdk"
	patchDirectory      = "third_party/modelcontextprotocol-go-sdk-patch"
	manifestName        = "MCP-SDK-PATCH.json"
	implementationPath  = "internal/json/json.go"
	equivalenceTestPath = "internal/json/json_equivalence_test.go"
	maxManifestBytes    = 64 << 10
	maxSourceBytes      = 4 << 20
	maxArchiveBytes     = 32 << 20
)

type fileDigest struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type changedFile struct {
	Path         string `json:"path"`
	BeforeSHA256 string `json:"beforeSHA256"`
	AfterSHA256  string `json:"afterSHA256"`
	Bytes        int64  `json:"bytes"`
}

type moduleIdentity struct {
	Path         string `json:"path"`
	Version      string `json:"version"`
	OriginCommit string `json:"originCommit"`
	ZipSum       string `json:"zipSum"`
	GoModSum     string `json:"goModSum"`
	ZipSHA256    string `json:"zipSHA256"`
	GoModSHA256  string `json:"goModSHA256"`
}

type patchManifest struct {
	SchemaVersion   int            `json:"schemaVersion"`
	PatchID         string         `json:"patchId"`
	Module          moduleIdentity `json:"module"`
	SourceDirectory string         `json:"sourceDirectory"`
	PatchFile       string         `json:"patchFile"`
	PatchSHA256     string         `json:"patchSHA256"`
	Reason          string         `json:"reason"`
	UpstreamFiles   []fileDigest   `json:"upstreamFiles"`
	Changes         []changedFile  `json:"changes"`
	Additions       []fileDigest   `json:"additions"`
}

func decodeManifest(data []byte) (patchManifest, error) {
	if len(data) == 0 || len(data) > maxManifestBytes {
		return patchManifest{}, fmt.Errorf("SDK patch manifest のサイズが範囲外です")
	}
	if err := legalqueryartifact.InspectJSONObject(data, legalqueryartifact.JSONLimits{Depth: 6, Values: 4096, RejectNull: true}); err != nil {
		return patchManifest{}, err
	}
	var value patchManifest
	if err := legalqueryartifact.DecodeClosed(data, &value); err != nil {
		return patchManifest{}, err
	}
	return value, value.validate()
}

func (m patchManifest) validate() error {
	if m.SchemaVersion != 1 || m.SourceDirectory != sourceDirectory || m.PatchFile != "json-unmarshal.patch" ||
		m.PatchID != "sha256-"+m.PatchSHA256 || !validHex(m.PatchSHA256, sha256.Size) || m.Reason == "" {
		return fmt.Errorf("SDK patch manifest の識別が不正です")
	}
	if m.Module.Path != ModulePath || m.Module.Version != UpstreamVersion || !validHex(m.Module.OriginCommit, 20) ||
		!validHex(m.Module.ZipSHA256, sha256.Size) || !validHex(m.Module.GoModSHA256, sha256.Size) ||
		!strings.HasPrefix(m.Module.ZipSum, "h1:") || !strings.HasPrefix(m.Module.GoModSum, "h1:") {
		return fmt.Errorf("SDK patch manifest の元 module 識別が不正です")
	}
	if len(m.Changes) != 1 || m.Changes[0].Path != implementationPath ||
		!validHex(m.Changes[0].BeforeSHA256, sha256.Size) || !validHex(m.Changes[0].AfterSHA256, sha256.Size) ||
		m.Changes[0].Bytes <= 0 || m.Changes[0].Bytes > maxSourceBytes {
		return fmt.Errorf("SDK の実装差分は固定 JSON file 一つでなければなりません")
	}
	if len(m.Additions) != 1 || m.Additions[0].Path != equivalenceTestPath || !validDigest(m.Additions[0]) {
		return fmt.Errorf("SDK の追加 file は固定した JSON 差分試験だけでなければなりません")
	}
	if len(m.UpstreamFiles) == 0 || len(m.UpstreamFiles) > 1024 {
		return fmt.Errorf("SDK の元 file 集合が不正です")
	}
	previous := ""
	foundChange := false
	for _, file := range m.UpstreamFiles {
		if !validDigest(file) || file.Path <= previous || file.Path == equivalenceTestPath {
			return fmt.Errorf("SDK の元 file 一覧が不正・重複・順序不一致です")
		}
		previous = file.Path
		if file.Path == implementationPath {
			foundChange = file.SHA256 == m.Changes[0].BeforeSHA256
		}
	}
	if !foundChange {
		return fmt.Errorf("SDK の変更前 hash が元 file 一覧と一致しません")
	}
	return nil
}

func validDigest(value fileDigest) bool {
	return validRelativePath(value.Path) && value.Bytes >= 0 && value.Bytes <= maxSourceBytes && validHex(value.SHA256, sha256.Size)
}

func validRelativePath(value string) bool {
	return value != "." && fs.ValidPath(value) && !strings.ContainsAny(value, "\\\x00\r\n:")
}

func validHex(value string, size int) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == size && value == strings.ToLower(value)
}

func digest(data []byte) string {
	value := sha256.Sum256(data)
	return hex.EncodeToString(value[:])
}
