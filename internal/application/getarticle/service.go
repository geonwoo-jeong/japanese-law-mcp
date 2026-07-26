package getarticle

import (
	"context"
	"fmt"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// Service は、primary provider の共通条文取得能力を公開 get_article へ投影する。
type Service struct {
	reader         lawarticleread.Port
	provider       model.ProviderDescriptor
	requestTimeout time.Duration
}

var _ Port = (*Service)(nil)

// NewService は、選択済み primary provider と request timeout を結び付ける。
func NewService(
	reader lawarticleread.Port,
	provider model.ProviderDescriptor,
	requestTimeout time.Duration,
) (*Service, error) {
	if reader == nil {
		return nil, fmt.Errorf("law.article.read provider は必須です")
	}
	if err := provider.Validate(); err != nil {
		return nil, fmt.Errorf("primary provider が有効ではありません: %w", err)
	}
	if !hasArticleReadCapability(provider) {
		return nil, fmt.Errorf("primary provider は law.article.read@1 を宣言しなければなりません")
	}
	if requestTimeout <= 0 {
		return nil, fmt.Errorf("requestTimeout は 0 秒より長くなければなりません")
	}
	return &Service{
		reader:         reader,
		provider:       provider,
		requestTimeout: requestTimeout,
	}, nil
}

// Get は、公開入力から共通資源参照を組み立て、XML の条文へ投影する。
func (s *Service) Get(
	ctx context.Context,
	request Request,
) (model.LawArticleFragment, error) {
	if ctx == nil {
		return model.LawArticleFragment{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return model.LawArticleFragment{}, err
	}
	resource, err := s.newResourceRef(request.LawID())
	if err != nil {
		return model.LawArticleFragment{}, err
	}
	asOf, exists := request.AsOf()
	var asOfPointer *model.Date
	if exists {
		asOfPointer = &asOf
	}
	readRequest, err := lawarticleread.NewRequest(lawarticleread.RequestValues{
		Resource: resource,
		AsOf:     asOfPointer,
		Location: request.Location(),
	})
	if err != nil {
		return model.LawArticleFragment{}, err
	}

	requestContext, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()
	result, err := s.reader.Read(requestContext, readRequest)
	if err != nil {
		return model.LawArticleFragment{}, err
	}
	if err := result.Validate(); err != nil {
		return model.LawArticleFragment{}, fmt.Errorf("law.article.read の結果が有効ではありません: %w", err)
	}
	fragment := result.Data()
	if fragment.Format() != model.LawArticleFormatXML {
		return model.LawArticleFragment{}, fmt.Errorf("公開 get_article は XML だけを返せます")
	}
	if !articleLocationsEqual(fragment.Location(), request.Location()) {
		return model.LawArticleFragment{}, fmt.Errorf("取得した条文位置が要求と一致しません")
	}
	return fragment, nil
}

func (s *Service) newResourceRef(lawID string) (model.SourceResourceRef, error) {
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     s.provider.Source().ID(),
		ResourceType: "law",
		ResourceID:   lawID,
	})
	if err != nil {
		return model.SourceResourceRef{}, err
	}
	return model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: s.provider.ProviderID(),
		Key:        key,
	})
}

func hasArticleReadCapability(provider model.ProviderDescriptor) bool {
	for _, capability := range provider.Capabilities() {
		if capability.ID() == lawarticleread.CapabilityID &&
			capability.MajorVersion() == lawarticleread.MajorVersion {
			return true
		}
	}
	return false
}

func articleLocationsEqual(left, right model.LawArticleLocation) bool {
	if left.Provision() != right.Provision() ||
		left.ArticleNumber() != right.ArticleNumber() {
		return false
	}
	leftParagraph, leftExists := left.ParagraphNumber()
	rightParagraph, rightExists := right.ParagraphNumber()
	return leftExists == rightExists &&
		(!leftExists || leftParagraph == rightParagraph)
}
