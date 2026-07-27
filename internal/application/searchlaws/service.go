package searchlaws

import (
	"context"
	"fmt"
	"reflect"
	"time"

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

// Search は、原検索を優先し、正常な空結果だけを一意な正式名称で再検索する。
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
		return original, nil
	}

	resolvedQuery, resolved, err := s.queryResolver.Resolve(
		requestContext,
		request.Query(),
	)
	if err != nil {
		return model.LawSearchResult{}, err
	}
	if !resolved || resolvedQuery == request.Query() {
		return original, nil
	}
	resolvedRequest, err := request.WithQuery(resolvedQuery)
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
	return result, nil
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
