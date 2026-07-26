package getlaw

import (
	"context"
	"fmt"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// Service は、primary provider の共通本文取得能力を公開 get_law へ投影する。
type Service struct {
	reader         lawdocumentread.Port
	provider       model.ProviderDescriptor
	requestTimeout time.Duration
}

var _ Port = (*Service)(nil)

// NewService は、選択済み primary provider と request timeout を結び付ける。
func NewService(
	reader lawdocumentread.Port,
	provider model.ProviderDescriptor,
	requestTimeout time.Duration,
) (*Service, error) {
	if reader == nil {
		return nil, fmt.Errorf("law.document.read provider は必須です")
	}
	if err := provider.Validate(); err != nil {
		return nil, fmt.Errorf("primary provider が有効ではありません: %w", err)
	}
	if !hasDocumentReadCapability(provider) {
		return nil, fmt.Errorf("primary provider は law.document.read@1 を宣言しなければなりません")
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

// Get は、公開入力から共通資源参照を組み立て、XML の LawDocument へ投影する。
func (s *Service) Get(
	ctx context.Context,
	request Request,
) (model.LawDocument, error) {
	if ctx == nil {
		return model.LawDocument{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return model.LawDocument{}, err
	}
	resource, err := s.newResourceRef(request.LawID())
	if err != nil {
		return model.LawDocument{}, err
	}
	asOf, exists := request.AsOf()
	var asOfPointer *model.Date
	if exists {
		asOfPointer = &asOf
	}
	readRequest, err := lawdocumentread.NewRequest(lawdocumentread.RequestValues{
		Resource: resource,
		AsOf:     asOfPointer,
	})
	if err != nil {
		return model.LawDocument{}, err
	}

	requestContext, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()
	result, err := s.reader.Read(requestContext, readRequest)
	if err != nil {
		return model.LawDocument{}, err
	}
	if err := result.Validate(); err != nil {
		return model.LawDocument{}, fmt.Errorf("law.document.read の結果が有効ではありません: %w", err)
	}
	document, err := model.NewLawDocumentFromRepresentation(result.Data())
	if err != nil {
		return model.LawDocument{}, fmt.Errorf("法令本文を公開形式へ投影できません: %w", err)
	}
	return document, nil
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

func hasDocumentReadCapability(provider model.ProviderDescriptor) bool {
	for _, capability := range provider.Capabilities() {
		if capability.ID() == lawdocumentread.CapabilityID &&
			capability.MajorVersion() == lawdocumentread.MajorVersion {
			return true
		}
	}
	return false
}
