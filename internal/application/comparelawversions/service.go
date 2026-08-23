package comparelawversions

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawversioncompare"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// Service は、primary provider の共通比較能力を公開 compare_law_versions へ投影する。
type Service struct {
	comparer       lawversioncompare.Port
	provider       model.ProviderDescriptor
	requestTimeout time.Duration
}

var _ Port = (*Service)(nil)

func NewService(
	comparer lawversioncompare.Port,
	provider model.ProviderDescriptor,
	requestTimeout time.Duration,
) (*Service, error) {
	if isNilComparer(comparer) {
		return nil, fmt.Errorf("law.version.compare provider は必須です")
	}
	if err := provider.Validate(); err != nil {
		return nil, fmt.Errorf("primary provider が有効ではありません: %w", err)
	}
	if !hasCompareCapability(provider) {
		return nil, fmt.Errorf("primary provider は law.version.compare@1 を宣言しなければなりません")
	}
	if requestTimeout <= 0 {
		return nil, fmt.Errorf("requestTimeout は 0 秒より長くなければなりません")
	}
	return &Service{comparer: comparer, provider: provider, requestTimeout: requestTimeout}, nil
}

func (s *Service) Compare(ctx context.Context, request Request) (model.LawVersionComparison, error) {
	if ctx == nil {
		return model.LawVersionComparison{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return model.LawVersionComparison{}, err
	}
	resource, err := s.newResourceRef(request.LawID())
	if err != nil {
		return model.LawVersionComparison{}, err
	}
	commonRequest, err := lawversioncompare.NewRequest(lawversioncompare.RequestValues{
		Resource: resource,
		Before:   request.Before(),
		After:    request.After(),
	})
	if err != nil {
		return model.LawVersionComparison{}, err
	}
	requestContext, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()
	result, err := s.comparer.Compare(requestContext, commonRequest)
	if err != nil {
		return model.LawVersionComparison{}, err
	}
	if err := validateSourcedComparison(s.provider, request.LawID(), result); err != nil {
		return model.LawVersionComparison{}, fmt.Errorf("%w: %w", ErrInvalidSourceResponse, err)
	}
	return result.Data(), nil
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

func hasCompareCapability(provider model.ProviderDescriptor) bool {
	for _, capability := range provider.Capabilities() {
		if capability.ID() == lawversioncompare.CapabilityID &&
			capability.MajorVersion() == lawversioncompare.MajorVersion {
			return true
		}
	}
	return false
}

func isNilComparer(comparer lawversioncompare.Port) bool {
	if comparer == nil {
		return true
	}
	value := reflect.ValueOf(comparer)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func validateSourcedComparison(
	provider model.ProviderDescriptor,
	expectedLawID string,
	result model.SourcedResource[model.LawVersionComparison],
) error {
	if err := result.Validate(); err != nil {
		return err
	}
	data := result.Data()
	if data.LawID() != expectedLawID {
		return fmt.Errorf("比較結果の lawId が要求と一致しません")
	}
	beforeLaw := data.Before().Law()
	afterLaw := data.After().Law()
	legalSource, err := model.NewLegalSource(provider.Source())
	if err != nil {
		return err
	}
	if beforeLaw.Source() != legalSource || afterLaw.Source() != legalSource {
		return fmt.Errorf("比較結果の source が primary provider と一致しません")
	}
	beforeKey, err := comparisonResourceKey(provider.Source().ID(), data.LawID(), beforeLaw.RevisionID())
	if err != nil {
		return err
	}
	afterKey, err := comparisonResourceKey(provider.Source().ID(), data.LawID(), afterLaw.RevisionID())
	if err != nil {
		return err
	}
	ref := result.Ref()
	if ref.ProviderID() != provider.ProviderID() || ref.Key() != afterKey {
		return fmt.Errorf("ref は primary provider の比較後版を指さなければなりません")
	}
	provenance := result.Provenance()
	final := provenance[len(provenance)-1]
	if final.Source() != provider.Source() ||
		final.Transformation() != model.ProvenanceTransformationDerived {
		return fmt.Errorf("最後の provenance は primary provider の derived でなければなりません")
	}
	inputKeys, exists := final.InputKeys()
	if !exists || len(inputKeys) != 2 || inputKeys[0] != beforeKey || inputKeys[1] != afterKey {
		return fmt.Errorf("最後の provenance.inputKeys は比較前後の版をこの順で持たなければなりません")
	}
	return nil
}

func comparisonResourceKey(
	sourceID string,
	lawID string,
	revisionID string,
) (model.SourceResourceKey, error) {
	return model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: "law",
		ResourceID:   lawID,
		VersionID:    revisionID,
	})
}
