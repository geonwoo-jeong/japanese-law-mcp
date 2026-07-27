package judicialdecisionread_test

import (
	"context"
	"errors"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestResolverSelectsExactReferenceProviderWithoutFallback(t *testing.T) {
	t.Parallel()

	first := &recordingJudicialDecisionReader{}
	second := &recordingJudicialDecisionReader{}
	resolver, err := judicialdecisionread.NewResolver(
		[]judicialdecisionread.ProviderBinding{
			{
				Descriptor: newJudicialProviderDescriptor(
					t,
					"first-provider",
					"first-source",
					true,
				),
				Enabled: true,
				Port:    first,
			},
			{
				Descriptor: newJudicialProviderDescriptor(
					t,
					"second-provider",
					"second-source",
					true,
				),
				Enabled: true,
				Port:    second,
			},
		},
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-012: NewResolver() のエラー = %v", err)
	}
	request := newJudicialReadRequest(
		t,
		"second-provider",
		"second-source",
	)

	got, err := resolver.Resolve(request)
	if err != nil {
		t.Fatalf("SOT-ARCH-012: Resolve() のエラー = %v", err)
	}
	if got != second {
		t.Fatal("SOT-ARCH-012: ref.providerId と異なる binding を選択しました")
	}
	if got == first {
		t.Fatal("SOT-IF-042: 最初の provider へ fallback しました")
	}
}

func TestResolverDistinguishesReferenceAndProviderStateFailures(t *testing.T) {
	t.Parallel()

	supported := newJudicialProviderDescriptor(
		t,
		"supported-provider",
		"supported-source",
		true,
	)
	disabled := newJudicialProviderDescriptor(
		t,
		"disabled-provider",
		"disabled-source",
		true,
	)
	unsupported := newJudicialProviderDescriptor(
		t,
		"unsupported-provider",
		"unsupported-source",
		false,
	)
	resolver, err := judicialdecisionread.NewResolver(
		[]judicialdecisionread.ProviderBinding{
			{
				Descriptor: supported,
				Enabled:    true,
				Port:       &recordingJudicialDecisionReader{},
			},
			{
				Descriptor: disabled,
				Enabled:    false,
				Port:       &recordingJudicialDecisionReader{},
			},
			{
				Descriptor: unsupported,
				Enabled:    true,
			},
		},
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-012: NewResolver() のエラー = %v", err)
	}

	tests := map[string]struct {
		request judicialdecisionread.Request
		code    model.ErrorCode
	}{
		"未知の provider": {
			request: newJudicialReadRequest(t, "unknown-provider", "unknown-source"),
			code:    model.ErrorCodeInvalidArgument,
		},
		"provider と source の不一致": {
			request: newJudicialReadRequest(
				t,
				"supported-provider",
				"different-source",
			),
			code: model.ErrorCodeInvalidArgument,
		},
		"無効な provider": {
			request: newJudicialReadRequest(
				t,
				"disabled-provider",
				"disabled-source",
			),
			code: model.ErrorCodeConfigurationRequired,
		},
		"capability 非対応": {
			request: newJudicialReadRequest(
				t,
				"unsupported-provider",
				"unsupported-source",
			),
			code: model.ErrorCodeUnsupportedCapability,
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, resolveErr := resolver.Resolve(test.request)
			var typedErr judicialdecisionread.ResolutionError
			if !errors.As(resolveErr, &typedErr) {
				t.Fatalf("SOT-IF-042: error = %T %v", resolveErr, resolveErr)
			}
			if typedErr.Code() != test.code {
				t.Fatalf(
					"SOT-IF-042: code = %q、期待値 = %q",
					typedErr.Code(),
					test.code,
				)
			}
			if typedErr.ProviderID() != test.request.Ref().ProviderID() {
				t.Fatalf(
					"SOT-IF-042: providerId = %q",
					typedErr.ProviderID(),
				)
			}
		})
	}
}

func TestResolverRejectsInvalidRegistryDefinitions(t *testing.T) {
	t.Parallel()

	supported := newJudicialProviderDescriptor(
		t,
		"supported-provider",
		"supported-source",
		true,
	)
	unsupported := newJudicialProviderDescriptor(
		t,
		"unsupported-provider",
		"unsupported-source",
		false,
	)
	port := &recordingJudicialDecisionReader{}
	tests := map[string][]judicialdecisionread.ProviderBinding{
		"無効な descriptor": {
			{Descriptor: model.ProviderDescriptor{}, Enabled: true, Port: port},
		},
		"providerId の重複": {
			{Descriptor: supported, Enabled: true, Port: port},
			{Descriptor: supported, Enabled: true, Port: port},
		},
		"宣言済み capability の port 欠落": {
			{Descriptor: supported, Enabled: true},
		},
		"未宣言 capability の port 指定": {
			{Descriptor: unsupported, Enabled: true, Port: port},
		},
	}
	for name, bindings := range tests {
		name, bindings := name, bindings
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := judicialdecisionread.NewResolver(bindings); err == nil {
				t.Fatal("SOT-ARCH-012: 不正な binding 定義を受理しました")
			}
		})
	}

	var typedNil *recordingJudicialDecisionReader
	if _, err := judicialdecisionread.NewResolver(
		[]judicialdecisionread.ProviderBinding{
			{Descriptor: supported, Enabled: true, Port: typedNil},
		},
	); err == nil {
		t.Fatal("SOT-ARCH-012: typed nil port を受理しました")
	}
}

func TestZeroValueResolverCannotResolve(t *testing.T) {
	t.Parallel()

	var resolver judicialdecisionread.Resolver
	_, err := resolver.Resolve(
		newJudicialReadRequest(t, "some-provider", "some-source"),
	)
	if err == nil {
		t.Fatal("SOT-ARCH-012: Resolver のゼロ値が解決しました")
	}
}

type recordingJudicialDecisionReader struct {
	read func(
		context.Context,
		judicialdecisionread.Request,
	) (model.SourcedResource[model.JudicialDecisionDetails], error)
}

func (r *recordingJudicialDecisionReader) Read(
	ctx context.Context,
	request judicialdecisionread.Request,
) (model.SourcedResource[model.JudicialDecisionDetails], error) {
	if r.read == nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, nil
	}
	return r.read(ctx, request)
}

func newJudicialReadRequest(
	t *testing.T,
	providerID string,
	sourceID string,
) judicialdecisionread.Request {
	t.Helper()

	request, err := judicialdecisionread.NewRequest(
		judicialdecisionread.RequestValues{
			Ref: newJudicialDecisionRef(
				t,
				providerID,
				sourceID,
				"95570",
				"",
			),
		},
	)
	if err != nil {
		t.Fatalf("JudicialDecisionReadRequest を作成できません: %v", err)
	}
	return request
}

func newJudicialProviderDescriptor(
	t *testing.T,
	providerID string,
	sourceID string,
	supportsRead bool,
) model.ProviderDescriptor {
	t.Helper()

	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         sourceID,
		Name:       "裁判例検索",
		Publisher:  "裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://www.courts.go.jp/hanrei/",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できません: %v", err)
	}
	verifiedAt, err := model.NewDate("2026-07-27")
	if err != nil {
		t.Fatalf("Date を作成できません: %v", err)
	}
	capabilities := []model.ProviderCapability{}
	if supportsRead {
		capability, capabilityErr := model.NewProviderCapability(
			model.ProviderCapabilityValues{
				ID:           judicialdecisionread.CapabilityID,
				MajorVersion: judicialdecisionread.MajorVersion,
				Level:        model.CapabilityLevelExtended,
				Stability:    model.CapabilityStabilityStable,
			},
		)
		if capabilityErr != nil {
			t.Fatalf("ProviderCapability を作成できません: %v", capabilityErr)
		}
		capabilities = append(capabilities, capability)
	}
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             providerID,
		Source:                 source,
		AdapterContractVersion: "1.0.0",
		VerifiedAt:             verifiedAt,
		InterfaceType:          model.InterfaceTypeHTML,
		Capabilities:           capabilities,
	})
	if err != nil {
		t.Fatalf("ProviderDescriptor を作成できません: %v", err)
	}
	return descriptor
}
