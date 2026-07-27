package judicialdecisionsearch

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

// Service は、裁判例検索へ一リクエスト単位の期限を適用する。
type Service struct {
	provider       Port
	requestTimeout time.Duration
}

var _ Port = (*Service)(nil)

// NewService は、選択済み provider と request timeout を結び付ける。
func NewService(provider Port, requestTimeout time.Duration) (*Service, error) {
	if isNilSearchPort(provider) {
		return nil, fmt.Errorf("judicial-decision.search provider は必須です")
	}
	if requestTimeout <= 0 {
		return nil, fmt.Errorf("requestTimeout は 0 秒より長くなければなりません")
	}
	return &Service{
		provider:       provider,
		requestTimeout: requestTimeout,
	}, nil
}

// Search は、入力を再検証してから期限付き context で provider を呼び出す。
func (s *Service) Search(ctx context.Context, request Request) (Page, error) {
	if ctx == nil {
		return Page{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return Page{}, err
	}
	requestContext, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()
	result, err := s.provider.Search(requestContext, request)
	if err != nil {
		return Page{}, err
	}
	if err := result.Validate(); err != nil {
		return Page{}, fmt.Errorf(
			"judicial-decision.search の結果が有効ではありません: %w",
			err,
		)
	}
	if result.Page().ReturnedCount() > request.Limit() {
		return Page{}, fmt.Errorf(
			"judicial-decision.search の結果が request.limit を超えています",
		)
	}
	return result, nil
}

func isNilSearchPort(provider Port) bool {
	if provider == nil {
		return true
	}
	value := reflect.ValueOf(provider)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
