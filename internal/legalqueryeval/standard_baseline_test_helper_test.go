package legalqueryeval

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

func loadStandardBaselineFromPath(path string) (StandardReport, error) {
	if path == "" {
		return StandardReport{}, fmt.Errorf("baseline path は必須です")
	}
	file, err := os.Open(path) //nolint:gosec // t.TempDir 配下の baseline fixture だけを読む。
	if err != nil {
		return StandardReport{}, fmt.Errorf("baseline を開けません: %w", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maximumStandardBaselineBytes {
		return StandardReport{}, fmt.Errorf("baseline file が不正です")
	}
	decoder := json.NewDecoder(io.LimitReader(file, maximumStandardBaselineBytes+1))
	decoder.DisallowUnknownFields()
	var document standardReportDTO
	if err := decoder.Decode(&document); err != nil {
		return StandardReport{}, fmt.Errorf("baseline JSON が不正です: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return StandardReport{}, fmt.Errorf("baseline JSON の末尾が不正です")
	}
	return standardReportFromDTO(document)
}
