package listlawrevisions

import (
	"context"
	"fmt"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawrevisionlist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// Service は、primary provider の改正履歴能力を公開結果へ投影する。
type Service struct {
	lister         lawrevisionlist.Port
	requestTimeout time.Duration
}

var _ Port = (*Service)(nil)

func NewService(lister lawrevisionlist.Port, requestTimeout time.Duration) (*Service, error) {
	if lister == nil {
		return nil, fmt.Errorf("law.revision.list provider は必須です")
	}
	if requestTimeout <= 0 {
		return nil, fmt.Errorf("requestTimeout は 0 秒より長くなければなりません")
	}
	return &Service{lister: lister, requestTimeout: requestTimeout}, nil
}

func (s *Service) List(ctx context.Context, request Request) (Result, error) {
	if ctx == nil {
		return Result{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return Result{}, err
	}
	commonRequest, err := lawrevisionlist.NewRequest(lawrevisionlist.RequestValues{
		LawIDOrNumber: request.LawIDOrNumber(),
	})
	if err != nil {
		return Result{}, err
	}
	requestContext, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()
	page, err := s.lister.List(requestContext, commonRequest)
	if err != nil {
		return Result{}, err
	}
	return projectRevisionPage(page)
}

func projectRevisionPage(page lawrevisionlist.Page) (Result, error) {
	if err := page.Validate(); err != nil {
		return Result{}, fmt.Errorf("%w: %w", ErrInvalidSourceResponse, err)
	}
	sourcePage := page.Page()
	totalCount, totalExists := sourcePage.TotalCount()
	totalRelation, relationExists := sourcePage.TotalRelation()
	resources := page.Items()
	if !totalExists || !relationExists ||
		totalRelation != model.TotalRelationExact ||
		totalCount != sourcePage.ReturnedCount() || totalCount != len(resources) {
		return Result{}, fmt.Errorf("%w: 件数が一致しません", ErrInvalidSourceResponse)
	}
	items := make([]model.LawRevision, len(resources))
	for index, resource := range resources {
		if err := resource.Validate(); err != nil {
			return Result{}, fmt.Errorf("%w: items[%d]: %w", ErrInvalidSourceResponse, index, err)
		}
		items[index] = resource.Data()
	}
	result, err := NewResult(ResultValues{
		LawID: page.LawID(), TotalCount: totalCount, Items: items,
	})
	if err != nil {
		return Result{}, fmt.Errorf("%w: %w", ErrInvalidSourceResponse, err)
	}
	return result, nil
}
