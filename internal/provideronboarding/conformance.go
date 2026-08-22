package provideronboarding

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/providerconformance"
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
	return runProviderConformanceTestsWith(
		ctx,
		repository,
		stdout,
		stderr,
		runGoTestCommand,
	)
}

type goTestCommand func(
	context.Context,
	string,
	[]string,
	io.Writer,
	io.Writer,
) error

func runProviderConformanceTestsWith(
	ctx context.Context,
	repository string,
	stdout io.Writer,
	stderr io.Writer,
	run goTestCommand,
) error {
	catalog, err := providerconformance.Load(repository)
	if err != nil {
		return fmt.Errorf("canonical conformance matrix を読み込めませんでした: %w", err)
	}
	arguments := append(
		[]string{"test", "-count=1"},
		providerConformanceTestTargets(catalog.Rows())...,
	)
	if err := run(ctx, repository, arguments, stdout, stderr); err != nil {
		return fmt.Errorf("go %s: %w", strings.Join(arguments, " "), err)
	}
	return nil
}

func providerConformanceTestTargets(
	rows []providerconformance.Row,
) []string {
	targets := map[string]struct{}{
		"./internal/providerconformance": {},
	}
	for _, row := range rows {
		if row.Status == "implemented" {
			targets[row.ConformanceTarget] = struct{}{}
		}
	}
	result := make([]string, 0, len(targets))
	for target := range targets {
		result = append(result, target)
	}
	sort.Strings(result)
	return result
}

func runGoTestCommand(
	ctx context.Context,
	repository string,
	arguments []string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	//nolint:gosec // SOT-ENG-017: schema 検証済みの package target を shell を介さず argv で渡す。
	command := exec.CommandContext(
		ctx,
		"go",
		arguments...,
	)
	command.Dir = repository
	command.Stdin = nil
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}
