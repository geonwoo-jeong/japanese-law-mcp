package legalqueryadoption

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

const adoptionIDPrefix = "adoption-sha256-"

func canonicalAdoptionID(document manifestDocument) string {
	var target bytes.Buffer
	writeObjectStart(&target, 15)
	writeStringField(&target, "artifactKind", document.ArtifactKind)
	writeIntegerField(&target, "schemaVersion", document.SchemaVersion)
	writeFieldName(&target, "previousAdoptionId")
	if document.PreviousAdoptionID == nil {
		target.WriteString("m;")
	} else {
		writeString(&target, *document.PreviousAdoptionID)
	}
	writeStringField(&target, "profileSetId", document.ProfileSetID)
	writeStringField(&target, "profileSetVersion", document.ProfileSetVersion)
	writeStringField(&target, "rankingVersion", document.RankingVersion)
	writeStringField(&target, "compositionVersion", document.CompositionVersion)
	writeStringField(&target, "evaluatorVersion", document.EvaluatorVersion)
	writeFieldName(&target, "profiles")
	writeArrayStart(&target, len(document.Profiles))
	for _, profile := range document.Profiles {
		writeObjectStart(&target, 3)
		writeStringField(&target, "profileId", profile.ProfileID)
		writeStringField(&target, "profileVersion", profile.ProfileVersion)
		writeStringField(&target, "cueSetVersion", profile.CueSetVersion)
		target.WriteByte('}')
	}
	target.WriteByte(']')
	writeStringField(&target, "corpusVersion", document.CorpusVersion)
	writeStringField(&target, "holdoutDigest", document.HoldoutDigest)
	writeStringField(&target, "baselineVersion", document.BaselineVersion)
	writeStringField(&target, "baselineSha256", document.BaselineSHA256)
	writeStringField(&target, "catalogVersion", document.CatalogVersion)
	writeStringField(&target, "catalogSha256", document.CatalogSHA256)
	target.WriteByte('}')
	sum := sha256.Sum256(target.Bytes())
	return adoptionIDPrefix + hex.EncodeToString(sum[:])
}

func writeStringField(target *bytes.Buffer, name, value string) {
	writeFieldName(target, name)
	writeString(target, value)
}

func writeIntegerField(target *bytes.Buffer, name string, value int) {
	writeFieldName(target, name)
	encoded := strconv.Itoa(value)
	target.WriteByte('i')
	writeLength(target, len(encoded))
	target.WriteString(encoded)
	target.WriteByte(';')
}

func writeFieldName(target *bytes.Buffer, name string) {
	writeString(target, name)
}

func writeString(target *bytes.Buffer, value string) {
	target.WriteByte('s')
	writeLength(target, len([]byte(value)))
	target.WriteString(value)
	target.WriteByte(';')
}

func writeObjectStart(target *bytes.Buffer, fieldCount int) {
	target.WriteByte('o')
	target.WriteString(strconv.Itoa(fieldCount))
	target.WriteByte('{')
}

func writeArrayStart(target *bytes.Buffer, count int) {
	target.WriteByte('a')
	target.WriteString(strconv.Itoa(count))
	target.WriteByte('[')
}

func writeLength(target *bytes.Buffer, length int) {
	target.WriteString(strconv.Itoa(length))
	target.WriteByte(':')
}

func validateManifestIdentity(document manifestDocument) error {
	if document.AdoptionID != canonicalAdoptionID(document) {
		return fmt.Errorf("adoptionId が canonical tuple digest と一致しません")
	}
	seen := make(map[string]struct{}, len(document.Profiles))
	for _, profile := range document.Profiles {
		if _, duplicate := seen[profile.ProfileID]; duplicate {
			return fmt.Errorf("profileId を重複させることはできません")
		}
		seen[profile.ProfileID] = struct{}{}
	}
	return nil
}
