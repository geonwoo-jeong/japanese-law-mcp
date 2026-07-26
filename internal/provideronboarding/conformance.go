package provideronboarding

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/providerconformance"
)

func loadCanonicalRows(repository string) ([]matrixRow, error) {
	catalog, err := providerconformance.Load(repository)
	if err != nil {
		return nil, err
	}
	module, err := readModulePath(repository)
	if err != nil {
		return nil, err
	}
	loaded := catalog.Rows()
	rows := make([]matrixRow, 0, len(loaded))
	for _, row := range loaded {
		implementedBy, err := repositoryProviderPath(module, row.ImplementedBy)
		if err != nil {
			return nil, fmt.Errorf(
				"providerId=%s の implementedBy を解決できません: %w",
				row.ProviderID,
				err,
			)
		}
		rows = append(rows, matrixRow{
			providerID:    row.ProviderID,
			implementedBy: implementedBy,
			status:        row.Status,
		})
	}
	return rows, nil
}

func repositoryProviderPath(module, implementedBy string) (string, error) {
	prefix := module + "/"
	if !strings.HasPrefix(implementedBy, prefix) {
		return "", fmt.Errorf("module %q の package ではありません: %q", module, implementedBy)
	}
	return strings.TrimPrefix(implementedBy, prefix), nil
}

func runProviderConformanceTests(
	ctx context.Context,
	repository string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	command := exec.CommandContext(
		ctx,
		"go",
		"test",
		"-count=1",
		"./internal/providerconformance",
	)
	command.Dir = repository
	command.Stdin = nil
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("go test -count=1 ./internal/providerconformance: %w", err)
	}
	return nil
}
