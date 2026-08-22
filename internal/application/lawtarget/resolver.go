package lawtarget

import (
	"context"
	"fmt"
	"reflect"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
)

// QueryResolver は、位置付きの検証済み入力から一意な法令対象だけを解決する。
type QueryResolver interface {
	Resolve(context.Context, string) (ResolvedLawTarget, bool, error)
}

// LogicalInputResolver は、分離済み検索語を再解析せず一意な対象だけを解決する。
type LogicalInputResolver interface {
	ResolveLogicalInput(context.Context, string) (ResolvedLawTarget, bool, error)
}

// Resolver は、専門検索と統合照会が共有する二つの解決入口を持つ。
type Resolver interface {
	QueryResolver
	LogicalInputResolver
}

// PreprocessResolver は、共通前処理の位置付き法令名事実を対象へ縮約する。
type PreprocessResolver struct {
	preprocessor legalquery.QueryPreprocessor
	direct       *searchquery.Resolver
}

var _ Resolver = PreprocessResolver{}

// NewPreprocessResolver は、共通前処理を共有する law-target resolver を返す。
func NewPreprocessResolver(
	preprocessor legalquery.QueryPreprocessor,
	entries []lawnamelexicon.Entry,
) (PreprocessResolver, error) {
	if isNilPreprocessor(preprocessor) {
		return PreprocessResolver{}, fmt.Errorf("法令対象の共通前処理器は必須です")
	}
	directEntries := make([]searchquery.EntryValues, len(entries))
	for index, entry := range entries {
		directEntries[index] = searchquery.EntryValues{
			ResourceID: entry.ResourceID,
			Canonical:  entry.Canonical,
			Terms:      append([]string(nil), entry.Terms...),
		}
	}
	direct, err := searchquery.NewResolver(directEntries, directOnlyAnalyzer{})
	if err != nil {
		return PreprocessResolver{}, fmt.Errorf(
			"法令対象の直接照合索引を構築できません: %w",
			err,
		)
	}
	return PreprocessResolver{preprocessor: preprocessor, direct: direct}, nil
}

// Resolve は、異なる位置を含めて法令名出現が一件の場合だけ対象を返す。
func (r PreprocessResolver) Resolve(
	ctx context.Context,
	query string,
) (ResolvedLawTarget, bool, error) {
	if ctx == nil {
		return ResolvedLawTarget{}, false, fmt.Errorf("context は必須です")
	}
	if err := ctx.Err(); err != nil {
		return ResolvedLawTarget{}, false, err
	}
	if isNilPreprocessor(r.preprocessor) || r.direct == nil {
		return ResolvedLawTarget{}, false, fmt.Errorf("法令対象 resolver は初期化されていません")
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{Query: query})
	if err != nil {
		// search_laws と統合照会では空白の扱いが異なる。統合照会用 request に
		// 変換できない公開検索語も、対象なしとして原検索へ渡す。
		return ResolvedLawTarget{}, false, nil
	}
	result, err := r.preprocessor.Preprocess(ctx, request)
	if err != nil {
		return ResolvedLawTarget{}, false, err
	}
	if err := result.Validate(); err != nil {
		return ResolvedLawTarget{}, false, fmt.Errorf(
			"法令対象の前処理結果が有効ではありません: %w",
			err,
		)
	}
	if result.Query() != request.Query() {
		return ResolvedLawTarget{}, false,
			fmt.Errorf("法令対象の前処理結果が検証済み query と一致しません")
	}
	mentions := result.LawNameMentions()
	if len(mentions) > 1 {
		return ResolvedLawTarget{}, false, nil
	}
	if len(mentions) == 0 {
		return ResolvedLawTarget{}, false, nil
	}
	mention := mentions[0]
	target, err := NewResolvedLawTarget(
		mention.LawID(),
		mention.Canonical(),
		MatchKind(mention.MatchKind()),
	)
	if err != nil {
		return ResolvedLawTarget{}, false, err
	}
	return target, true, nil
}

// ResolveLogicalInput は、Kagome を再実行せず辞書全体との直接照合だけを行う。
func (r PreprocessResolver) ResolveLogicalInput(
	ctx context.Context,
	query string,
) (ResolvedLawTarget, bool, error) {
	if ctx == nil {
		return ResolvedLawTarget{}, false, fmt.Errorf("context は必須です")
	}
	if err := ctx.Err(); err != nil {
		return ResolvedLawTarget{}, false, err
	}
	if r.direct == nil {
		return ResolvedLawTarget{}, false, fmt.Errorf("法令対象 resolver は初期化されていません")
	}
	matches, err := r.direct.ResolveMatches(ctx, query)
	if err != nil {
		return ResolvedLawTarget{}, false, err
	}
	if len(matches) != 1 {
		return ResolvedLawTarget{}, false, nil
	}
	match := matches[0]
	target, err := NewResolvedLawTarget(
		match.ResourceID(),
		match.Canonical(),
		MatchKind(match.Kind()),
	)
	if err != nil {
		return ResolvedLawTarget{}, false, err
	}
	return target, true, nil
}

// directOnlyAnalyzer は、分離済み検索語で登録語の再抽出を行わない。
type directOnlyAnalyzer struct{}

func (directOnlyAnalyzer) RegisteredTerms(
	ctx context.Context,
	_ string,
) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []string{}, nil
}

func isNilPreprocessor(value legalquery.QueryPreprocessor) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
