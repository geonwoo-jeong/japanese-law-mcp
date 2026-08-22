package searchlaws

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawtarget"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// Service は、公開法令検索へ一リクエスト単位の期限を適用する。
type Service struct {
	provider       Port
	queryResolver  QueryResolver
	requestTimeout time.Duration
}

var _ Port = (*Service)(nil)

// NewService は、検索語解決器、選択済み provider と request timeout を結び付ける。
func NewService(
	provider Port,
	queryResolver QueryResolver,
	requestTimeout time.Duration,
) (*Service, error) {
	if isNilDependency(provider) {
		return nil, fmt.Errorf("search_laws provider は必須です")
	}
	if isNilDependency(queryResolver) {
		return nil, fmt.Errorf("search_laws query resolver は必須です")
	}
	if requestTimeout <= 0 {
		return nil, fmt.Errorf("requestTimeout は 0 秒より長くなければなりません")
	}
	return &Service{
		provider:       provider,
		queryResolver:  queryResolver,
		requestTimeout: requestTimeout,
	}, nil
}

// Search は、原検索を優先し、解決済み法令を page 内で安定的に優先する。
func (s *Service) Search(
	ctx context.Context,
	request Request,
) (model.LawSearchResult, error) {
	if ctx == nil {
		return model.LawSearchResult{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return model.LawSearchResult{}, err
	}
	requestContext, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	resolvedTarget, resolved, err := s.queryResolver.Resolve(
		requestContext,
		request.Query(),
	)
	if err != nil {
		return model.LawSearchResult{}, err
	}
	if resolved {
		if err := resolvedTarget.Validate(); err != nil {
			return model.LawSearchResult{}, err
		}
	}

	original, err := s.provider.Search(requestContext, request)
	if err != nil {
		return model.LawSearchResult{}, err
	}
	if err := original.Validate(); err != nil {
		return model.LawSearchResult{}, fmt.Errorf(
			"search_laws provider が不正な結果を返しました: %w",
			err,
		)
	}
	if original.TotalCount() > 0 {
		return prioritizeResolvedLawTarget(original, resolvedTarget, resolved)
	}
	if !resolved {
		return original, nil
	}
	resolvedRequest, err := request.WithQuery(resolvedTarget.OfficialTitle())
	if err != nil {
		return model.LawSearchResult{}, fmt.Errorf(
			"解決した search_laws query が有効ではありません: %w",
			err,
		)
	}
	result, err := s.provider.Search(requestContext, resolvedRequest)
	if err != nil {
		return model.LawSearchResult{}, err
	}
	if err := result.Validate(); err != nil {
		return model.LawSearchResult{}, fmt.Errorf(
			"search_laws provider が不正な結果を返しました: %w",
			err,
		)
	}
	if result.TotalCount() == 0 {
		return original, nil
	}
	return prioritizeResolvedLawTarget(result, resolvedTarget, true)
}

func isNilDependency(value any) bool {
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

func prioritizeResolvedLawTarget(
	result model.LawSearchResult,
	target lawtarget.ResolvedLawTarget,
	resolved bool,
) (model.LawSearchResult, error) {
	if !resolved {
		return result, nil
	}
	prioritized, changed, err := lawtarget.Prioritize(
		result.Items(),
		target,
		func(item model.LawSummary) string { return item.LawID() },
	)
	if err != nil {
		return model.LawSearchResult{}, fmt.Errorf(
			"search_laws の解決済み法令を優先できません: %w",
			err,
		)
	}
	if !changed {
		return result, nil
	}
	nextOffset, hasNextOffset := result.NextOffset()
	values := model.LawSearchResultValues{
		TotalCount: result.TotalCount(),
		Items:      prioritized,
	}
	if hasNextOffset {
		values.NextOffset = &nextOffset
	}
	reordered, err := model.NewLawSearchResult(values)
	if err != nil {
		return model.LawSearchResult{}, fmt.Errorf(
			"優先順位付き search_laws result を構築できません: %w",
			err,
		)
	}
	return reordered, nil
}
