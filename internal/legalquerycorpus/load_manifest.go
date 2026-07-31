package legalquerycorpus

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

// ManifestArtifact は、fixture を開かずに検証した manifest と原 byte を束ねる。
type ManifestArtifact struct {
	manifest Manifest
	raw      []byte
	sha256   string
}

// Manifest は、内部 field を直接変更できない検証済み manifest 値を返す。
func (a ManifestArtifact) Manifest() Manifest { return a.manifest }

// RawBytes は manifest 原 byte の複製を返す。
func (a ManifestArtifact) RawBytes() []byte { return bytes.Clone(a.raw) }

// SHA256 は manifest 原 byte の小文字 SHA-256 を返す。
func (a ManifestArtifact) SHA256() string { return a.sha256 }

// LoadManifest は、corpus fixture directory を走査せず manifest だけを検証する。
func LoadManifest(
	ctx context.Context,
	repositoryRoot string,
	corpusDirectory string,
) (artifact ManifestArtifact, err error) {
	if err := checkCorpusFilesystemContext(ctx); err != nil {
		return ManifestArtifact{}, err
	}
	paths, err := resolveCorpusFilesystemPaths(repositoryRoot, corpusDirectory)
	if err != nil {
		return ManifestArtifact{}, err
	}
	filesystem := &corpusFilesystem{corpusVersion: paths.corpusVersion}
	defer func() {
		if closeErr := filesystem.close(); closeErr != nil {
			artifact = ManifestArtifact{}
			err = errors.Join(err, closeErr)
		}
	}()
	if err := filesystem.openRoots(ctx, paths.repositoryRoot); err != nil {
		return ManifestArtifact{}, err
	}
	manifestInfo, err := validateCorpusRootEntries(ctx, filesystem.corpusRoot)
	if err != nil {
		return ManifestArtifact{}, err
	}
	raw, err := readRegularCorpusFile(
		ctx,
		filesystem.corpusRoot,
		"manifest.json",
		corpusManifestMaximumBytes,
		manifestInfo,
	)
	if err != nil {
		return ManifestArtifact{}, err
	}
	manifest, err := decodeManifestWithoutFixtures(ctx, filesystem, raw)
	if err != nil {
		return ManifestArtifact{}, err
	}
	digest := sha256.Sum256(raw)
	return ManifestArtifact{
		manifest: manifest,
		raw:      bytes.Clone(raw),
		sha256:   hex.EncodeToString(digest[:]),
	}, nil
}

func decodeManifestWithoutFixtures(
	ctx context.Context,
	filesystem *corpusFilesystem,
	raw []byte,
) (Manifest, error) {
	header, err := inspectJSONDocument(raw)
	if err != nil {
		return Manifest{}, err
	}
	if header.artifactKind != ArtifactKindCorpusManifest ||
		!isSupportedCorpusSchemaVersion(header.schemaVersion) {
		return Manifest{}, fmt.Errorf("corpus manifest の kind または schema version が不正です")
	}
	schemaRaw, err := filesystem.readSchema(ctx, header.schemaVersion)
	if err != nil {
		return Manifest{}, err
	}
	schema, err := newCorpusSchema(header.schemaVersion, schemaRaw)
	if err != nil {
		return Manifest{}, err
	}
	decoded, err := schema.validateAndDecode(raw, header)
	if err != nil {
		return Manifest{}, err
	}
	if decoded.kind != ArtifactKindCorpusManifest {
		return Manifest{}, fmt.Errorf("corpus manifest を復元できません")
	}
	return decoded.manifest, nil
}
