package providerconformance

import (
	"fmt"
	"path/filepath"
	"reflect"
)

// InterfaceSOTReferenceReplacement は、matrix 内の一つの SOT 参照置換を表す。
type InterfaceSOTReferenceReplacement struct {
	RowIndex       int
	ReferenceIndex int
	CapabilityID   string
	MajorVersion   int
	Operation      string
	PreviousSOTID  string
	CurrentSOTID   string
}

// InterfaceSOTReferenceChange は、matrix 内の SOT 参照置換を行と要素の順序で保持する。
type InterfaceSOTReferenceChange struct {
	Replacements []InterfaceSOTReferenceReplacement
}

// ClassifyInterfaceSOTReferenceOnlyChange は、SOT-ENG-044 に従い、二つの
// provider matrix が interfaceSotIds の同位置置換だけで異なるかを判定する。
func ClassifyInterfaceSOTReferenceOnlyChange(
	repository string,
	providerID string,
	before []byte,
	after []byte,
) (InterfaceSOTReferenceChange, bool, error) {
	if !providerIDPattern.MatchString(providerID) {
		return InterfaceSOTReferenceChange{}, false, fmt.Errorf(
			"providerId %q が canonical 形式ではありません",
			providerID,
		)
	}

	root, err := resolveRepositoryRoot(repository)
	if err != nil {
		return InterfaceSOTReferenceChange{}, false, err
	}
	schemaPath := filepath.Join(root, filepath.FromSlash(canonicalSchemaRelativePath))
	schema, err := loadSchema(schemaPath)
	if err != nil {
		return InterfaceSOTReferenceChange{}, false, err
	}

	beforeMatrix, err := loadProviderMatrixBytes(before, providerID, schema)
	if err != nil {
		return InterfaceSOTReferenceChange{}, false, fmt.Errorf("変更前 matrix が不正です: %w", err)
	}
	afterMatrix, err := loadProviderMatrixBytes(after, providerID, schema)
	if err != nil {
		return InterfaceSOTReferenceChange{}, false, fmt.Errorf("変更後 matrix が不正です: %w", err)
	}

	if beforeMatrix.SchemaVersion != afterMatrix.SchemaVersion ||
		len(beforeMatrix.rows) != len(afterMatrix.rows) {
		return InterfaceSOTReferenceChange{}, false, nil
	}

	replacements := make([]InterfaceSOTReferenceReplacement, 0)
	for rowIndex := range beforeMatrix.rows {
		beforeRow := beforeMatrix.rows[rowIndex]
		afterRow := afterMatrix.rows[rowIndex]
		if !rowsEqualExceptInterfaceSOTReferences(beforeRow, afterRow) ||
			len(beforeRow.InterfaceSOTIDs) != len(afterRow.InterfaceSOTIDs) ||
			interfaceSOTReferenceOrderChanged(beforeRow.InterfaceSOTIDs, afterRow.InterfaceSOTIDs) {
			return InterfaceSOTReferenceChange{}, false, nil
		}

		for referenceIndex := range beforeRow.InterfaceSOTIDs {
			previous := beforeRow.InterfaceSOTIDs[referenceIndex]
			current := afterRow.InterfaceSOTIDs[referenceIndex]
			if previous == current {
				continue
			}
			replacements = append(replacements, InterfaceSOTReferenceReplacement{
				RowIndex:       rowIndex,
				ReferenceIndex: referenceIndex,
				CapabilityID:   beforeRow.CapabilityID,
				MajorVersion:   beforeRow.MajorVersion,
				Operation:      beforeRow.Operation,
				PreviousSOTID:  previous,
				CurrentSOTID:   current,
			})
		}
	}

	if len(replacements) == 0 {
		return InterfaceSOTReferenceChange{}, false, nil
	}
	return InterfaceSOTReferenceChange{Replacements: replacements}, true, nil
}

func rowsEqualExceptInterfaceSOTReferences(left, right Row) bool {
	left.InterfaceSOTIDs = nil
	right.InterfaceSOTIDs = nil
	return reflect.DeepEqual(left, right)
}

func interfaceSOTReferenceOrderChanged(before, after []string) bool {
	beforeIndexes := make(map[string]int, len(before))
	afterIndexes := make(map[string]int, len(after))
	for index, id := range before {
		beforeIndexes[id] = index
	}
	for index, id := range after {
		afterIndexes[id] = index
	}

	for index := range before {
		if before[index] == after[index] {
			continue
		}
		if previousIndex, exists := beforeIndexes[after[index]]; exists && previousIndex != index {
			return true
		}
		if currentIndex, exists := afterIndexes[before[index]]; exists && currentIndex != index {
			return true
		}
	}
	return false
}
