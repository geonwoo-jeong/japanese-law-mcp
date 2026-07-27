package application_test

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestProviderBindingRegistryRegistersExactTypedBindings(t *testing.T) {
	t.Parallel()

	bindings := newCompleteProviderBindings(t, "complete-provider")
	registry, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{bindings},
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-012: NewProviderBindingRegistry() のエラー = %v", err)
	}

	if descriptor, exists := registry.Descriptor("complete-provider"); !exists ||
		descriptor.ProviderID() != "complete-provider" {
		t.Fatalf("SOT-ARCH-012: Descriptor() = %#v, %t", descriptor, exists)
	}
	if port, exists := registry.LawSearch("complete-provider"); !exists ||
		port != bindings.LawSearch {
		t.Fatalf("SOT-ARCH-012: LawSearch() = %#v, %t", port, exists)
	}
	if port, exists := registry.LawContentSearch("complete-provider"); !exists ||
		port != bindings.LawContentSearch {
		t.Fatalf("SOT-ARCH-012: LawContentSearch() = %#v, %t", port, exists)
	}
	if port, exists := registry.LawDocumentRead("complete-provider"); !exists ||
		port != bindings.LawDocumentRead {
		t.Fatalf("SOT-ARCH-012: LawDocumentRead() = %#v, %t", port, exists)
	}
	if port, exists := registry.LawArticleRead("complete-provider"); !exists ||
		port != bindings.LawArticleRead {
		t.Fatalf("SOT-ARCH-012: LawArticleRead() = %#v, %t", port, exists)
	}
	if port, exists := registry.LawUpdateList("complete-provider"); !exists ||
		port != bindings.LawUpdateList {
		t.Fatalf("SOT-ARCH-012: LawUpdateList() = %#v, %t", port, exists)
	}
	if port, exists := registry.JudicialDecisionSearch("complete-provider"); !exists ||
		port != bindings.JudicialDecisionSearch {
		t.Fatalf("SOT-IF-041: JudicialDecisionSearch() = %#v, %t", port, exists)
	}
	if port, exists := registry.JudicialDecisionRead("complete-provider"); !exists ||
		port != bindings.JudicialDecisionRead {
		t.Fatalf("SOT-IF-042: JudicialDecisionRead() = %#v, %t", port, exists)
	}
}

func TestProviderBindingRegistryRejectsDeclarationAndPortMismatch(t *testing.T) {
	t.Parallel()

	complete := newCompleteProviderBindings(t, "complete-provider")
	missingPort := complete
	missingPort.LawUpdateList = nil

	undeclaredPort := complete
	undeclaredPort.Descriptor = newBindingDescriptor(t, "complete-provider",
		judicialdecisionread.CapabilityID,
		judicialdecisionsearch.CapabilityID,
		lawarticleread.CapabilityID,
		lawcontentsearch.CapabilityID,
		lawdocumentread.CapabilityID,
		lawsearch.CapabilityID,
	)

	wrongMajor := complete
	wrongMajor.Descriptor = newBindingDescriptorWithCapabilities(
		t,
		"complete-provider",
		[]capabilityValues{
			{
				id:           judicialdecisionread.CapabilityID,
				majorVersion: judicialdecisionread.MajorVersion,
			},
			{
				id:           judicialdecisionsearch.CapabilityID,
				majorVersion: judicialdecisionsearch.MajorVersion,
			},
			{id: lawarticleread.CapabilityID, majorVersion: lawarticleread.MajorVersion},
			{id: lawcontentsearch.CapabilityID, majorVersion: lawcontentsearch.MajorVersion},
			{id: lawdocumentread.CapabilityID, majorVersion: lawdocumentread.MajorVersion},
			{id: lawsearch.CapabilityID, majorVersion: lawsearch.MajorVersion},
			{id: lawupdatelist.CapabilityID, majorVersion: 2},
		},
	)

	var typedNil *fakeLawUpdateListBinding
	typedNilPort := complete
	typedNilPort.LawUpdateList = typedNil

	for name, bindings := range map[string]application.ProviderBindings{
		"宣言に対応する port の欠落": missingPort,
		"未宣言 port":         undeclaredPort,
		"未対応 majorVersion": wrongMajor,
		"typed nil port":   typedNilPort,
	} {
		name, bindings := name, bindings
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := application.NewProviderBindingRegistry(
				[]application.ProviderBindings{bindings},
			); err == nil {
				t.Fatalf("SOT-ARCH-012: 不一致 binding を受理した: %#v", bindings)
			}
		})
	}
}

func TestProviderBindingRegistryRejectsDuplicateAndInvalidDescriptor(t *testing.T) {
	t.Parallel()

	bindings := newCompleteProviderBindings(t, "complete-provider")
	if _, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{bindings, bindings},
	); err == nil {
		t.Fatal("SOT-ARCH-012: providerId の重複を受理した")
	}
	if _, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{{}},
	); err == nil {
		t.Fatal("SOT-ARCH-012: 無効な descriptor を受理した")
	}
}

type fakeLawSearchBinding struct {
	name string
}

type fakeJudicialDecisionSearchBinding struct {
	name string
}

func (*fakeJudicialDecisionSearchBinding) Search(
	context.Context,
	judicialdecisionsearch.Request,
) (judicialdecisionsearch.Page, error) {
	return judicialdecisionsearch.Page{}, nil
}

type fakeJudicialDecisionReadBinding struct {
	name string
}

func (*fakeJudicialDecisionReadBinding) Read(
	context.Context,
	judicialdecisionread.Request,
) (model.SourcedResource[model.JudicialDecisionDetails], error) {
	return model.SourcedResource[model.JudicialDecisionDetails]{}, nil
}

func (*fakeLawSearchBinding) Search(
	context.Context,
	lawsearch.Request,
) (lawsearch.Page, error) {
	return lawsearch.Page{}, nil
}

type fakeLawContentSearchBinding struct {
	name string
}

func (*fakeLawContentSearchBinding) Search(
	context.Context,
	lawcontentsearch.Request,
) (lawcontentsearch.Page, error) {
	return lawcontentsearch.Page{}, nil
}

type fakeLawDocumentReadBinding struct {
	name string
}

func (*fakeLawDocumentReadBinding) Read(
	context.Context,
	lawdocumentread.Request,
) (model.SourcedResource[model.LawDocumentRepresentation], error) {
	return model.SourcedResource[model.LawDocumentRepresentation]{}, nil
}

type fakeLawArticleReadBinding struct {
	name string
}

func (*fakeLawArticleReadBinding) Read(
	context.Context,
	lawarticleread.Request,
) (model.SourcedResource[model.LawArticleFragment], error) {
	return model.SourcedResource[model.LawArticleFragment]{}, nil
}

type fakeLawUpdateListBinding struct {
	name string
}

func (*fakeLawUpdateListBinding) List(
	context.Context,
	lawupdatelist.Request,
) (lawupdatelist.Page, error) {
	return lawupdatelist.Page{}, nil
}

func newCompleteProviderBindings(
	t *testing.T,
	providerID string,
) application.ProviderBindings {
	t.Helper()
	return application.ProviderBindings{
		Descriptor: newBindingDescriptor(
			t,
			providerID,
			judicialdecisionread.CapabilityID,
			judicialdecisionsearch.CapabilityID,
			lawarticleread.CapabilityID,
			lawcontentsearch.CapabilityID,
			lawdocumentread.CapabilityID,
			lawsearch.CapabilityID,
			lawupdatelist.CapabilityID,
		),
		JudicialDecisionRead:   &fakeJudicialDecisionReadBinding{name: providerID},
		JudicialDecisionSearch: &fakeJudicialDecisionSearchBinding{name: providerID},
		LawSearch:              &fakeLawSearchBinding{name: providerID},
		LawContentSearch:       &fakeLawContentSearchBinding{name: providerID},
		LawDocumentRead:        &fakeLawDocumentReadBinding{name: providerID},
		LawArticleRead:         &fakeLawArticleReadBinding{name: providerID},
		LawUpdateList:          &fakeLawUpdateListBinding{name: providerID},
	}
}

func newBindingDescriptor(
	t *testing.T,
	providerID string,
	capabilityIDs ...string,
) model.ProviderDescriptor {
	t.Helper()
	capabilities := make([]capabilityValues, len(capabilityIDs))
	for index, capabilityID := range capabilityIDs {
		capabilities[index] = capabilityValues{
			id:           capabilityID,
			majorVersion: 1,
		}
	}
	return newBindingDescriptorWithCapabilities(t, providerID, capabilities)
}

func newBindingDescriptorWithCapabilities(
	t *testing.T,
	providerID string,
	capabilities []capabilityValues,
) model.ProviderDescriptor {
	t.Helper()
	return newProviderDescriptor(t, providerDescriptorValues{
		providerID:   providerID,
		sourceID:     providerID,
		capabilities: capabilities,
	})
}
