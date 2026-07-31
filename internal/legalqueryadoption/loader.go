// Package legalqueryadoption は、採用済み profile set の固定 manifest を解決する。
package legalqueryadoption

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

const (
	adoptionSchemaFilename        = "schema-v1.json"
	maximumAdoptionSchemaBytes    = 1 << 20
	maximumAdoptionPointerBytes   = 64 << 10
	maximumAdoptionManifestBytes  = 256 << 10
	maximumAdoptionHistoryFiles   = 4096
	maximumAdoptionHistoryBytes   = 32 << 20
	maximumAdoptionDocumentDepth  = 8
	maximumAdoptionDocumentValues = 4096
)

var adoptionIDPattern = regexp.MustCompile(`^adoption-sha256-[0-9a-f]{64}$`)

// LoadCurrent は、repository の固定 pointer が指す採用 tuple を全履歴検証後に返す。
func LoadCurrent(ctx context.Context) (Manifest, error) {
	manifest, err := loadCurrent(ctx, ".")
	if err != nil {
		return Manifest{}, err
	}
	repository, err := legalqueryartifact.OpenRepository(".")
	if err != nil {
		return Manifest{}, fmt.Errorf("adoption repository を開けません: %w", err)
	}
	defer func() { _ = repository.Close() }()
	corpus, err := legalquerycorpus.Load(
		ctx,
		".",
		path.Join("testdata/legalquery", manifest.CorpusVersion()),
	)
	if err != nil {
		return Manifest{}, fmt.Errorf("adoption corpus を読めません: %w", err)
	}
	if err := verifyCatalog(ctx, repository, manifest, corpus); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func loadCurrent(
	ctx context.Context,
	repositoryRoot string,
) (Manifest, error) {
	if err := checkAdoptionContext(ctx); err != nil {
		return Manifest{}, err
	}
	repository, err := legalqueryartifact.OpenRepository(repositoryRoot)
	if err != nil {
		return Manifest{}, fmt.Errorf("adoption repository を開けません: %w", err)
	}
	defer func() { _ = repository.Close() }()
	root, err := openAdoptionRoot(repository)
	if err != nil {
		return Manifest{}, err
	}
	defer func() { _ = root.Close() }()
	if err := validateAdoptionRootEntries(root); err != nil {
		return Manifest{}, err
	}
	schema, err := loadAdoptionSchema(root)
	if err != nil {
		return Manifest{}, err
	}
	pointer, err := loadAdoptionPointer(root, schema)
	if err != nil {
		return Manifest{}, err
	}
	historyRoot, err := root.OpenChild("history")
	if err != nil {
		return Manifest{}, fmt.Errorf("adoption history を開けません: %w", err)
	}
	defer func() { _ = historyRoot.Close() }()
	history, err := loadAdoptionHistory(ctx, historyRoot, schema)
	if err != nil {
		return Manifest{}, err
	}
	if err := validateAdoptionGraph(history); err != nil {
		return Manifest{}, err
	}
	current, exists := history[pointer.AdoptionID]
	if !exists {
		return Manifest{}, fmt.Errorf("current adoption が history に存在しません")
	}
	return manifestFromDocument(current), nil
}

func openAdoptionRoot(
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
	defer func() { _ = legalqueryRoot.Close() }()
	root, err := legalqueryRoot.OpenChild("adoptions")
	if err != nil {
		return nil, fmt.Errorf("adoption directory を開けません: %w", err)
	}
	return root, nil
}

func validateAdoptionRootEntries(root *legalqueryartifact.Repository) error {
	entries, err := root.ReadDirectory(
		3,
		maximumAdoptionSchemaBytes+maximumAdoptionPointerBytes,
	)
	if err != nil {
		return fmt.Errorf("adoption directory を検査できません: %w", err)
	}
	if len(entries) != 3 ||
		entries[0].Name() != "current.json" ||
		entries[1].Name() != "history" ||
		entries[2].Name() != adoptionSchemaFilename ||
		!entries[0].Info().Mode().IsRegular() ||
		!entries[1].Info().IsDir() ||
		entries[1].Info().Mode()&os.ModeSymlink != 0 ||
		!entries[2].Info().Mode().IsRegular() {
		return fmt.Errorf("adoption directory の entry が不正です")
	}
	return nil
}

func loadAdoptionSchema(
	root *legalqueryartifact.Repository,
) (adoptionSchemaV1, error) {
	raw, err := root.ReadRegular(adoptionSchemaFilename, maximumAdoptionSchemaBytes)
	if err != nil {
		return adoptionSchemaV1{}, fmt.Errorf("adoption schema を読めません: %w", err)
	}
	return newAdoptionSchemaV1(raw)
}

func loadAdoptionPointer(
	root *legalqueryartifact.Repository,
	schema adoptionSchemaV1,
) (pointerDocument, error) {
	raw, err := root.ReadRegular("current.json", maximumAdoptionPointerBytes)
	if err != nil {
		return pointerDocument{}, fmt.Errorf("adoption pointer を読めません: %w", err)
	}
	if err := inspectAdoptionDocument(raw); err != nil {
		return pointerDocument{}, err
	}
	if err := schema.validate(raw); err != nil {
		return pointerDocument{}, err
	}
	var pointer pointerDocument
	if err := legalqueryartifact.DecodeClosed(raw, &pointer); err != nil {
		return pointerDocument{}, fmt.Errorf("adoption pointer を解釈できません: %w", err)
	}
	if pointer.ArtifactKind != "legal_query_adoption_pointer" ||
		pointer.SchemaVersion != 1 ||
		!adoptionIDPattern.MatchString(pointer.AdoptionID) {
		return pointerDocument{}, fmt.Errorf("adoption pointer が不正です")
	}
	if err := verifyCanonicalJSON(raw, pointer); err != nil {
		return pointerDocument{}, fmt.Errorf("adoption pointer が canonical ではありません")
	}
	return pointer, nil
}

func loadAdoptionHistory(
	ctx context.Context,
	root *legalqueryartifact.Repository,
	schema adoptionSchemaV1,
) (map[string]manifestDocument, error) {
	entries, err := root.ReadDirectory(
		maximumAdoptionHistoryFiles,
		maximumAdoptionHistoryBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("adoption history を列挙できません: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("adoption history は一件以上必要です")
	}
	history := make(map[string]manifestDocument, len(entries))
	for _, entry := range entries {
		if err := checkAdoptionContext(ctx); err != nil {
			return nil, err
		}
		name := entry.Name()
		info := entry.Info()
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() ||
			info.Size() < 1 || info.Size() > maximumAdoptionManifestBytes ||
			!strings.HasSuffix(name, ".json") {
			return nil, fmt.Errorf("adoption history の entry が不正です")
		}
		adoptionID := strings.TrimSuffix(name, ".json")
		if !adoptionIDPattern.MatchString(adoptionID) {
			return nil, fmt.Errorf("adoption history の file 名が不正です")
		}
		raw, err := root.ReadRegular(name, maximumAdoptionManifestBytes)
		if err != nil {
			return nil, fmt.Errorf("adoption manifest を読めません: %w", err)
		}
		document, err := decodeAdoptionManifest(raw, schema)
		if err != nil {
			return nil, err
		}
		if document.AdoptionID != adoptionID {
			return nil, fmt.Errorf("adoptionId が file 名と一致しません")
		}
		if _, duplicate := history[document.AdoptionID]; duplicate {
			return nil, fmt.Errorf("adoptionId を重複させることはできません")
		}
		history[document.AdoptionID] = document
	}
	return history, nil
}

func decodeAdoptionManifest(
	raw []byte,
	schema adoptionSchemaV1,
) (manifestDocument, error) {
	if err := inspectAdoptionDocument(raw); err != nil {
		return manifestDocument{}, err
	}
	if err := schema.validate(raw); err != nil {
		return manifestDocument{}, err
	}
	var document manifestDocument
	if err := legalqueryartifact.DecodeClosed(raw, &document); err != nil {
		return manifestDocument{}, fmt.Errorf("adoption manifest を解釈できません: %w", err)
	}
	if document.ArtifactKind != "legal_query_adoption" || document.SchemaVersion != 1 {
		return manifestDocument{}, fmt.Errorf("adoption manifest の版が不正です")
	}
	if err := verifyCanonicalJSON(raw, document); err != nil {
		return manifestDocument{}, fmt.Errorf("adoption manifest が canonical ではありません")
	}
	if err := validateManifestIdentity(document); err != nil {
		return manifestDocument{}, err
	}
	return document, nil
}

func inspectAdoptionDocument(raw []byte) error {
	if err := legalqueryartifact.InspectJSONObject(raw, legalqueryartifact.JSONLimits{
		Depth:      maximumAdoptionDocumentDepth,
		Values:     maximumAdoptionDocumentValues,
		RejectNull: true,
	}); err != nil {
		return fmt.Errorf("adoption JSON が不正です: %w", err)
	}
	return nil
}

func verifyCanonicalJSON(raw []byte, value any) error {
	canonical, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("adoption 成果物を canonical JSON にできません")
	}
	canonical = append(canonical, '\n')
	if !bytes.Equal(raw, canonical) {
		return fmt.Errorf("adoption 成果物の原 byte が canonical ではありません")
	}
	return nil
}

func checkAdoptionContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("adoption context は nil にできません")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("adoption 読取りが取り消されました: %w", err)
	}
	return nil
}
