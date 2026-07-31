package legalqueryeval

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
)

const (
	baselineSchemaFilename        = "legal-query-baseline-v1.schema.json"
	maximumBaselineSchemaBytes    = 1 << 20
	maximumBaselineHistoryFiles   = 4096
	maximumBaselineHistoryBytes   = 256 << 20
	maximumBaselineDocumentDepth  = 12
	maximumBaselineDocumentValues = 65536
)

var baselineVersionPattern = regexp.MustCompile(`^default-[1-9][0-9]*$`)

// BaselineArtifact は、採用済み baseline の原 byte、digest と typed report を束ねる。
type BaselineArtifact struct {
	report StandardReport
	raw    []byte
	sha256 string
}

// Report は検証済みの標準 report を返す。
func (a BaselineArtifact) Report() StandardReport { return a.report }

// RawBytes は baseline の原 byte の複製を返す。
func (a BaselineArtifact) RawBytes() []byte {
	return append([]byte(nil), a.raw...)
}

// SHA256 は baseline の原 byte digest を返す。
func (a BaselineArtifact) SHA256() string { return a.sha256 }

// LoadCurrentBaseline は、repository 内の current baseline と同版の履歴を読む。
func LoadCurrentBaseline(
	ctx context.Context,
	baselineVersion string,
) (BaselineArtifact, error) {
	return loadBaselineRepository(ctx, ".", baselineVersion)
}

func loadBaselineRepository(
	ctx context.Context,
	repositoryRoot string,
	baselineVersion string,
) (BaselineArtifact, error) {
	if err := checkBaselineContext(ctx); err != nil {
		return BaselineArtifact{}, err
	}
	if !baselineVersionPattern.MatchString(baselineVersion) {
		return BaselineArtifact{}, fmt.Errorf("baselineVersion が不正です")
	}
	repository, err := legalqueryartifact.OpenRepository(repositoryRoot)
	if err != nil {
		return BaselineArtifact{}, fmt.Errorf("baseline repository を開けません: %w", err)
	}
	defer func() { _ = repository.Close() }()
	legalqueryRoot, err := openBaselineLegalQueryRoot(repository)
	if err != nil {
		return BaselineArtifact{}, err
	}
	defer func() { _ = legalqueryRoot.Close() }()

	schema, err := loadBaselineSchema(legalqueryRoot)
	if err != nil {
		return BaselineArtifact{}, err
	}
	baselineRoot, err := legalqueryRoot.OpenChild("baselines")
	if err != nil {
		return BaselineArtifact{}, fmt.Errorf("baseline directory を開けません: %w", err)
	}
	defer func() { _ = baselineRoot.Close() }()
	if err := validateBaselineRootEntries(baselineRoot); err != nil {
		return BaselineArtifact{}, err
	}
	versionsRoot, err := baselineRoot.OpenChild("versions")
	if err != nil {
		return BaselineArtifact{}, fmt.Errorf("baseline history を開けません: %w", err)
	}
	defer func() { _ = versionsRoot.Close() }()
	if err := validateBaselineHistory(versionsRoot, baselineVersion); err != nil {
		return BaselineArtifact{}, err
	}

	current, err := baselineRoot.ReadRegular("default.json", maximumStandardBaselineBytes)
	if err != nil {
		return BaselineArtifact{}, fmt.Errorf("current baseline を読めません: %w", err)
	}
	versioned, err := versionsRoot.ReadRegular(
		baselineVersion+".json",
		maximumStandardBaselineBytes,
	)
	if err != nil {
		return BaselineArtifact{}, fmt.Errorf("version baseline を読めません: %w", err)
	}
	if !bytes.Equal(current, versioned) {
		return BaselineArtifact{}, fmt.Errorf("current baseline と version byte が一致しません")
	}
	report, err := decodeRepositoryBaseline(current, schema)
	if err != nil {
		return BaselineArtifact{}, err
	}
	if report.BaselineVersion() != baselineVersion {
		return BaselineArtifact{}, fmt.Errorf("baselineVersion が file 名と一致しません")
	}
	sum := sha256.Sum256(current)
	return BaselineArtifact{
		report: report,
		raw:    append([]byte(nil), current...),
		sha256: hex.EncodeToString(sum[:]),
	}, nil
}

func openBaselineLegalQueryRoot(
	repository *legalqueryartifact.Repository,
) (*legalqueryartifact.Repository, error) {
	testdata, err := repository.OpenChild("testdata")
	if err != nil {
		return nil, fmt.Errorf("testdata directory を開けません: %w", err)
	}
	defer func() { _ = testdata.Close() }()
	legalqueryRoot, err := testdata.OpenChild("legalquery")
	if err != nil {
		return nil, fmt.Errorf("legalquery directory を開けません: %w", err)
	}
	return legalqueryRoot, nil
}

func loadBaselineSchema(
	legalqueryRoot *legalqueryartifact.Repository,
) (baselineSchemaV1, error) {
	schemas, err := legalqueryRoot.OpenChild("schemas")
	if err != nil {
		return baselineSchemaV1{}, fmt.Errorf("baseline schema directory を開けません: %w", err)
	}
	defer func() { _ = schemas.Close() }()
	raw, err := schemas.ReadRegular(baselineSchemaFilename, maximumBaselineSchemaBytes)
	if err != nil {
		return baselineSchemaV1{}, fmt.Errorf("baseline schema を読めません: %w", err)
	}
	return newBaselineSchemaV1(raw)
}

func validateBaselineRootEntries(root *legalqueryartifact.Repository) error {
	entries, err := root.ReadDirectory(2, maximumStandardBaselineBytes)
	if err != nil {
		return fmt.Errorf("baseline directory を検査できません: %w", err)
	}
	if len(entries) != 2 ||
		entries[0].Name() != "default.json" ||
		entries[1].Name() != "versions" ||
		!entries[0].Info().Mode().IsRegular() ||
		!entries[1].Info().IsDir() ||
		entries[1].Info().Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("baseline directory の entry が不正です")
	}
	return nil
}

func validateBaselineHistory(
	root *legalqueryartifact.Repository,
	baselineVersion string,
) error {
	entries, err := root.ReadDirectory(
		maximumBaselineHistoryFiles,
		maximumBaselineHistoryBytes,
	)
	if err != nil {
		return fmt.Errorf("baseline history を列挙できません: %w", err)
	}
	var totalBytes int64
	found := false
	for _, entry := range entries {
		info := entry.Info()
		name := entry.Name()
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() ||
			info.Size() < 1 || info.Size() > maximumStandardBaselineBytes ||
			len(name) <= len(".json") || name[len(name)-len(".json"):] != ".json" ||
			!baselineVersionPattern.MatchString(name[:len(name)-len(".json")]) {
			return fmt.Errorf("baseline history の entry が不正です")
		}
		totalBytes += info.Size()
		if name == baselineVersion+".json" {
			found = true
		}
	}
	if err := validateBaselineHistoryBudget(len(entries), totalBytes); err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("指定した baseline version が存在しません")
	}
	return nil
}

func validateBaselineHistoryBudget(count int, totalBytes int64) error {
	if count < 1 || count > maximumBaselineHistoryFiles {
		return fmt.Errorf("baseline history の件数が上限外です")
	}
	if totalBytes < 1 || totalBytes > maximumBaselineHistoryBytes {
		return fmt.Errorf("baseline history の byte 合計が上限外です")
	}
	return nil
}

func decodeRepositoryBaseline(
	raw []byte,
	schema baselineSchemaV1,
) (StandardReport, error) {
	if len(raw) < 1 || len(raw) > maximumStandardBaselineBytes {
		return StandardReport{}, fmt.Errorf("baseline size が上限外です")
	}
	if err := legalqueryartifact.InspectJSONObject(raw, legalqueryartifact.JSONLimits{
		Depth:      maximumBaselineDocumentDepth,
		Values:     maximumBaselineDocumentValues,
		RejectNull: true,
	}); err != nil {
		return StandardReport{}, fmt.Errorf("baseline JSON が不正です: %w", err)
	}
	if err := schema.validate(raw); err != nil {
		return StandardReport{}, err
	}
	var document standardReportDTO
	if err := legalqueryartifact.DecodeClosed(raw, &document); err != nil {
		return StandardReport{}, fmt.Errorf("baseline typed decode に失敗しました: %w", err)
	}
	report, err := standardReportFromDTO(document)
	if err != nil {
		return StandardReport{}, err
	}
	canonical, err := json.Marshal(report)
	if err != nil {
		return StandardReport{}, fmt.Errorf("baseline を canonical JSON にできません: %w", err)
	}
	canonical = append(canonical, '\n')
	if !bytes.Equal(raw, canonical) {
		return StandardReport{}, fmt.Errorf("baseline byte が canonical 形式ではありません")
	}
	return report, nil
}

func checkBaselineContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("baseline context は nil にできません")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("baseline 読取りが取り消されました: %w", err)
	}
	return nil
}
