package searchlawcontent

import (
	"context"
	"fmt"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

// Service は、公開本文検索へ一リクエスト単位の期限を適用する。
type Service struct {
	provider       Port
	requestTimeout time.Duration
}

var _ Port = (*Service)(nil)

// NewService は、選択済み e-Gov provider と request timeout を結び付ける。
func NewService(provider Port, requestTimeout time.Duration) (*Service, error) {
	if provider == nil {
		return nil, fmt.Errorf("search_law_content provider は必須です")
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
func (s *Service) Search(
	ctx context.Context,
	request Request,
) (model.LawContentSearchResult, error) {
	if ctx == nil {
		return model.LawContentSearchResult{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return model.LawContentSearchResult{}, err
	}
	requestContext, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()
	result, err := s.provider.Search(requestContext, request)
	if err != nil {
		return model.LawContentSearchResult{}, err
	}
	if err := result.Validate(); err != nil {
		return model.LawContentSearchResult{}, fmt.Errorf(
			"search_law_content の結果が有効ではありません: %w",
			err,
		)
	}
	return result, nil
}
