package listlawupdates

import (
	"context"
	"fmt"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// Service は、primary provider の更新一覧能力を公開結果へ投影する。
type Service struct {
	lister         lawupdatelist.Port
	requestTimeout time.Duration
}

var _ Port = (*Service)(nil)

// NewService は、選択済み primary provider と request timeout を結び付ける。
func NewService(
	lister lawupdatelist.Port,
	requestTimeout time.Duration,
) (*Service, error) {
	if lister == nil {
		return nil, fmt.Errorf("law.update.list provider は必須です")
	}
	if requestTimeout <= 0 {
		return nil, fmt.Errorf("requestTimeout は 0 秒より長くなければなりません")
	}
	return &Service{
		lister:         lister,
		requestTimeout: requestTimeout,
	}, nil
}

// List は、共通能力を期限付きで呼び出し、内部情報を除いた一覧へ投影する。
func (s *Service) List(
	ctx context.Context,
	request Request,
) (Result, error) {
	if ctx == nil {
		return Result{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return Result{}, err
	}
	commonRequest, err := lawupdatelist.NewRequest(lawupdatelist.RequestValues{
		Date: request.Date(),
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
	return projectPage(request, page)
}

func projectPage(request Request, result lawupdatelist.Page) (Result, error) {
	if err := result.Validate(); err != nil {
		return Result{}, fmt.Errorf(
			"%w: law.update.list の結果が有効ではありません: %w",
			ErrInvalidSourceResponse,
			err,
		)
	}
	if result.Date() != request.Date() {
		return Result{}, fmt.Errorf(
			"%w: law.update.list の対象日が要求と一致しません",
			ErrInvalidSourceResponse,
		)
	}

	sourcePage := result.Page()
	totalCount, totalExists := sourcePage.TotalCount()
	totalRelation, relationExists := sourcePage.TotalRelation()
	resources := result.Items()
	if !totalExists ||
		!relationExists ||
		totalRelation != model.TotalRelationExact ||
		totalCount != sourcePage.ReturnedCount() ||
		totalCount != len(resources) {
		return Result{}, fmt.Errorf(
			"%w: law.update.list の件数が有効ではありません",
			ErrInvalidSourceResponse,
		)
	}

	returnedCount := min(request.Limit(), len(resources))
	items := make([]model.LawUpdate, returnedCount)
	for index, resource := range resources {
		if err := resource.Validate(); err != nil {
			return Result{}, fmt.Errorf(
				"%w: law.update.list の items[%d] が有効ではありません: %w",
				ErrInvalidSourceResponse,
				index,
				err,
			)
		}
		if index < returnedCount {
			items[index] = resource.Data()
		}
	}
	resultValues := ResultValues{
		Date:       request.Date(),
		TotalCount: totalCount,
		Items:      items,
	}
	projected, err := NewResult(resultValues)
	if err != nil {
		return Result{}, fmt.Errorf(
			"%w: law.update.list を公開結果へ投影できません: %w",
			ErrInvalidSourceResponse,
			err,
		)
	}
	return projected, nil
}
