package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryadoption"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval/evaluators"
)

const standardAdoptionPath = "testdata/legalquery/adoptions/current.json"

type options struct {
	Adoption string
	Format   string
}

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr, execute)
	stop()
	os.Exit(code)
}

func run(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	execute func(context.Context, options) ([]byte, error),
) int {
	current, err := parseOptions(args, io.Discard)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			writeUsage(stdout)
			return 0
		}
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	result, err := execute(ctx, current)
	if len(result) > 0 {
		if _, writeErr := stdout.Write(result); writeErr != nil {
			_, _ = fmt.Fprintf(
				stderr,
				"統合照会評価の JSON を出力できません: %v\n",
				writeErr,
			)
			return 1
		}
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "統合照会評価 command が失敗しました: %v\n", err)
		return 1
	}
	return 0
}

func writeUsage(output io.Writer) {
	_, _ = fmt.Fprintln(
		output,
		"使用方法: legal-query-eval "+
			"--adoption <testdata/legalquery/adoptions/current.json> --format json",
	)
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	flags := flag.NewFlagSet("legal-query-eval", flag.ContinueOnError)
	flags.SetOutput(stderr)

	adoption := singleOption{name: "--adoption"}
	outputFormat := singleOption{name: "--format"}
	flags.Var(&adoption, "adoption", "採用済み profile set pointer")
	flags.Var(&outputFormat, "format", "出力形式")
	if err := flags.Parse(args); err != nil {
		return options{}, fmt.Errorf("統合照会評価の引数を解釈できません: %w", err)
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("統合照会評価に位置引数は指定できません")
	}
	for _, value := range []*singleOption{
		&adoption,
		&outputFormat,
	} {
		if err := value.requireOne(); err != nil {
			return options{}, fmt.Errorf("統合照会評価の引数が不正です: %w", err)
		}
	}
	current := options{
		Adoption: normalizeRepositoryPath(adoption.value),
		Format:   outputFormat.value,
	}
	if current.Adoption != standardAdoptionPath {
		return options{}, fmt.Errorf(
			"統合照会評価の --adoption は %s だけを受け付けます",
			standardAdoptionPath,
		)
	}
	if current.Format != "json" {
		return options{}, fmt.Errorf(
			"統合照会評価の --format は json だけを受け付けます: %q",
			current.Format,
		)
	}
	return current, nil
}

func normalizeRepositoryPath(value string) string {
	return path.Clean(strings.ReplaceAll(value, `\`, "/"))
}

type singleOption struct {
	name  string
	value string
	count int
}

func (o *singleOption) String() string {
	if o == nil {
		return ""
	}
	return o.value
}

func (o *singleOption) Set(value string) error {
	if o.count != 0 {
		return fmt.Errorf("%s は一回だけ指定できます", o.name)
	}
	o.count++
	o.value = value
	return nil
}

func (o *singleOption) requireOne() error {
	if o.count != 1 || o.value == "" {
		return fmt.Errorf("%s を一回指定してください", o.name)
	}
	return nil
}

func execute(ctx context.Context, current options) ([]byte, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context は nil にできません")
	}
	if current.Adoption != standardAdoptionPath || current.Format != "json" {
		return nil, fmt.Errorf("標準評価 option が adoption 基準ではありません")
	}
	adoption, err := legalqueryadoption.LoadCurrent(ctx)
	if err != nil {
		return nil, fmt.Errorf("current adoption を解決できません: %w", err)
	}
	evaluator, err := evaluators.New(adoption.EvaluatorVersion())
	if err != nil {
		return nil, err
	}
	identity, err := evaluator.Identity()
	if err != nil {
		return nil, err
	}
	if err := verifyAdoptionProfileIdentity(adoption, identity); err != nil {
		return nil, err
	}
	corpus, err := legalquerycorpus.Load(
		ctx,
		".",
		path.Join("testdata/legalquery", adoption.CorpusVersion()),
	)
	if err != nil {
		return nil, fmt.Errorf("corpus を読み込めません: %w", err)
	}
	if err := verifyAdoptionCorpusIdentity(adoption, corpus); err != nil {
		return nil, err
	}
	baseline, err := legalqueryeval.LoadCurrentBaseline(
		ctx,
		adoption.BaselineVersion(),
	)
	if err != nil {
		return nil, err
	}
	if err := verifyAdoptionBaselineIdentity(adoption, baseline); err != nil {
		return nil, err
	}
	report, err := evaluator.BuildStandardReport(
		ctx,
		corpus,
		adoption.BaselineVersion(),
	)
	if err != nil {
		return nil, fmt.Errorf("標準評価 report を構築できません: %w", err)
	}

	encoded, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("標準評価 report を JSON 化できません: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := legalqueryeval.VerifyStandardAcceptance(report); err != nil {
		return encoded, fmt.Errorf("受入基準を満たしません: %w", err)
	}
	if err := legalqueryeval.CompareStandardBaseline(report, baseline.Report()); err != nil {
		return encoded, err
	}
	return encoded, nil
}
