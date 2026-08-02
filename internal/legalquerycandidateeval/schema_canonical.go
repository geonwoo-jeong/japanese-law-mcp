package legalquerycandidateeval

import (
	"context"
	"fmt"
	"sync"
)

var (
	canonicalSchemasOnce sync.Once
	canonicalSchemas     artifactSchemas
	canonicalSchemasErr  error
)

type artifactSchemas struct {
	v2 SchemaV2
	v3 SchemaV3
}

func loadCanonicalArtifactSchemas() (artifactSchemas, error) {
	canonicalSchemasOnce.Do(func() {
		canonicalSchemas.v2, canonicalSchemasErr = ParseSchemaV2(CanonicalSchemaV2())
		if canonicalSchemasErr != nil {
			return
		}
		canonicalSchemas.v3, canonicalSchemasErr = ParseSchemaV3(CanonicalSchemaV3())
	})
	if canonicalSchemasErr != nil {
		return artifactSchemas{}, fmt.Errorf("固定済み candidate evaluation schema を初期化できません: %w", canonicalSchemasErr)
	}
	return canonicalSchemas, nil
}

func (s artifactSchemas) validate(ctx context.Context, schemaVersion int, raw []byte) error {
	switch schemaVersion {
	case SchemaVersionV2:
		return s.v2.Validate(ctx, raw)
	case SchemaVersionV3:
		return s.v3.Validate(ctx, raw)
	default:
		return fmt.Errorf("candidate evaluation schema version が未対応です")
	}
}
