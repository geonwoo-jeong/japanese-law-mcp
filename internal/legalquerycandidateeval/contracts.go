package legalquerycandidateeval

import (
	"fmt"
	"slices"
)

var requiredReviewSOTIDsV2 = [...]string{
	"SOT-ARCH-018",
	"SOT-ARCH-021",
	"SOT-ARCH-023",
	"SOT-ARCH-025",
	"SOT-ARCH-027",
	"SOT-ARCH-028",
	"SOT-ARCH-031",
	"SOT-ARCH-033",
	"SOT-ARCH-036",
	"SOT-ARCH-037",
	"SOT-ARCH-038",
	"SOT-ARCH-039",
	"SOT-ENG-007",
	"SOT-ENG-008",
	"SOT-ENG-009",
	"SOT-ENG-020",
	"SOT-ENG-022",
	"SOT-ENG-023",
	"SOT-ENG-024",
	"SOT-ENG-025",
	"SOT-ENG-026",
	"SOT-ENG-027",
	"SOT-ENG-028",
	"SOT-ENG-030",
	"SOT-ENG-031",
	"SOT-ENG-032",
	"SOT-ENG-033",
	"SOT-ENG-035",
	"SOT-ENG-036",
	"SOT-ENG-038",
	"SOT-ENG-039",
	"SOT-IF-022",
	"SOT-IF-023",
	"SOT-IF-024",
	"SOT-IF-025",
	"SOT-IF-034",
	"SOT-IF-040",
	"SOT-IF-041",
	"SOT-IF-042",
	"SOT-IF-051",
	"SOT-MODEL-013",
	"SOT-MODEL-016",
	"SOT-MODEL-018",
	"SOT-MODEL-022",
	"SOT-MODEL-023",
	"SOT-MODEL-024",
	"SOT-MODEL-025",
	"SOT-MODEL-026",
	"SOT-MODEL-027",
	"SOT-MODEL-028",
	"SOT-MODEL-030",
	"SOT-MODEL-031",
}

var additionalRequiredReviewSOTIDsV3 = [...]string{
	"SOT-ENG-040",
	"SOT-ENG-041",
	"SOT-ENG-042",
}

// RequiredReviewSOTIDs は schema v2 初回候補の閉じた SOT ID 集合を返す。
func RequiredReviewSOTIDs() []string {
	return append([]string(nil), requiredReviewSOTIDsV2[:]...)
}

// RequiredReviewSOTIDsForSchema は schema 版に固定された SOT ID 集合を返す。
func RequiredReviewSOTIDsForSchema(schemaVersion int) ([]string, error) {
	switch schemaVersion {
	case SchemaVersionV2:
		return RequiredReviewSOTIDs(), nil
	case SchemaVersionV3:
		ids := make([]string, 0, len(requiredReviewSOTIDsV2)+len(additionalRequiredReviewSOTIDsV3))
		ids = append(ids, requiredReviewSOTIDsV2[:]...)
		ids = append(ids, additionalRequiredReviewSOTIDsV3[:]...)
		slices.Sort(ids)
		return ids, nil
	case SchemaVersionV4:
		// SOT-ENG-045: 旧世代の集合を変更せず、後継の有効集合を構成する。
		previous, err := RequiredReviewSOTIDsForSchema(SchemaVersionV3)
		if err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(previous)+2)
		for _, id := range previous {
			if id != "SOT-IF-040" {
				ids = append(ids, id)
			}
		}
		ids = append(ids, "SOT-IF-077", "SOT-ENG-043", "SOT-ENG-045")
		slices.Sort(ids)
		return ids, nil
	case SchemaVersionV5:
		// SOT-ENG-048: v4 の集合へ後継四規定だけを明示的に追加する。
		previous, err := RequiredReviewSOTIDsForSchema(SchemaVersionV4)
		if err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(previous)+4)
		ids = append(ids, previous...)
		ids = append(ids, "SOT-ENG-046", "SOT-ENG-047", "SOT-ENG-048", "SOT-ENG-049")
		slices.Sort(ids)
		return ids, nil
	default:
		return nil, fmt.Errorf("candidate evaluation schema version が未対応です")
	}
}
