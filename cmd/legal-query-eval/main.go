package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval/defaultprofile"
)

const (
	standardCorpusPath      = "testdata/legalquery/corpus-v9"
	standardBaselinePath    = "testdata/legalquery/baselines/default.json"
	standardBaselineVersion = "default-1"
)

type options struct {
	Corpus     string
	ProfileSet string
	Baseline   string
	Format     string
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
	current, err := parseOptions(args, stderr)
	if err != nil {
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

func parseOptions(args []string, stderr io.Writer) (options, error) {
	flags := flag.NewFlagSet("legal-query-eval", flag.ContinueOnError)
	flags.SetOutput(stderr)

	corpus := singleOption{name: "--corpus"}
	profileSet := singleOption{name: "--profile-set"}
	baseline := singleOption{name: "--baseline"}
	outputFormat := singleOption{name: "--format"}
	flags.Var(&corpus, "corpus", "評価 corpus directory")
	flags.Var(&profileSet, "profile-set", "評価 profile set")
	flags.Var(&baseline, "baseline", "review 済み baseline path")
	flags.Var(&outputFormat, "format", "出力形式")
	if err := flags.Parse(args); err != nil {
		return options{}, fmt.Errorf("統合照会評価の引数を解釈できません: %w", err)
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("統合照会評価に位置引数は指定できません")
	}
	for _, value := range []*singleOption{
		&corpus,
		&profileSet,
		&baseline,
		&outputFormat,
	} {
		if err := value.requireOne(); err != nil {
			return options{}, fmt.Errorf("統合照会評価の引数が不正です: %w", err)
		}
	}
	current := options{
		Corpus:     normalizeRepositoryPath(corpus.value),
		ProfileSet: profileSet.value,
		Baseline:   normalizeRepositoryPath(baseline.value),
		Format:     outputFormat.value,
	}
	if current.Corpus != standardCorpusPath {
		return options{}, fmt.Errorf(
			"統合照会評価の --corpus は %s だけを受け付けます",
			standardCorpusPath,
		)
	}
	if current.Baseline != standardBaselinePath {
		return options{}, fmt.Errorf(
			"統合照会評価の --baseline は %s だけを受け付けます",
			standardBaselinePath,
		)
	}
	if current.ProfileSet != "default" {
		return options{}, fmt.Errorf(
			"統合照会評価の --profile-set は default だけを受け付けます: %q",
			current.ProfileSet,
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
	evaluator, err := defaultprofile.New()
	if err != nil {
		return nil, fmt.Errorf("default evaluator を構築できません: %w", err)
	}
	corpus, err := legalquerycorpus.Load(
		ctx,
		".",
		current.Corpus,
	)
	if err != nil {
		return nil, fmt.Errorf("corpus を読み込めません: %w", err)
	}

	baseline, err := legalqueryeval.LoadStandardBaseline(current.Baseline)
	if err != nil {
		return nil, err
	}
	if err := verifyStandardBaselineVersion(
		baseline.BaselineVersion(),
	); err != nil {
		return nil, err
	}
	report, err := evaluator.BuildStandardReport(
		ctx,
		corpus,
		standardBaselineVersion,
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
	if err := legalqueryeval.CompareStandardBaseline(report, baseline); err != nil {
		return encoded, err
	}
	return encoded, nil
}

func verifyStandardBaselineVersion(version string) error {
	if version != standardBaselineVersion {
		return fmt.Errorf(
			"baselineVersion は %s でなければなりません",
			standardBaselineVersion,
		)
	}
	return nil
}
