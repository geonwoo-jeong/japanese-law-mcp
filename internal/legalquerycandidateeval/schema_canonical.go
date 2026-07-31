package legalquerycandidateeval

import (
	"fmt"
	"sync"
)

var (
	canonicalSchemaOnce sync.Once
	canonicalSchema     SchemaV2
	canonicalSchemaErr  error
)

func validateCanonicalArtifactSchema(raw []byte) error {
	canonicalSchemaOnce.Do(func() {
		canonicalSchema, canonicalSchemaErr = ParseSchemaV2(CanonicalSchemaV2())
	})
	if canonicalSchemaErr != nil {
		return fmt.Errorf("固定済み schema v2 を初期化できません: %w", canonicalSchemaErr)
	}
	if err := canonicalSchema.validateRaw(raw); err != nil {
		return err
	}
	return nil
}
