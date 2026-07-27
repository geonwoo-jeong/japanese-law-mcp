package application

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
)

const (
	// ProviderRouteSelectionPrimary は、能力の既定 provider を一つ選ぶ route である。
	ProviderRouteSelectionPrimary = "primary"
)

type providerRouteKey struct {
	capabilityID string
	majorVersion int
}

func lawSearchProviderRouteKey() providerRouteKey {
	return providerRouteKey{
		capabilityID: lawsearch.CapabilityID,
		majorVersion: lawsearch.MajorVersion,
	}
}

func lawContentSearchProviderRouteKey() providerRouteKey {
	return providerRouteKey{
		capabilityID: lawcontentsearch.CapabilityID,
		majorVersion: lawcontentsearch.MajorVersion,
	}
}

func lawDocumentReadProviderRouteKey() providerRouteKey {
	return providerRouteKey{
		capabilityID: lawdocumentread.CapabilityID,
		majorVersion: lawdocumentread.MajorVersion,
	}
}

func lawArticleReadProviderRouteKey() providerRouteKey {
	return providerRouteKey{
		capabilityID: lawarticleread.CapabilityID,
		majorVersion: lawarticleread.MajorVersion,
	}
}

func lawUpdateListProviderRouteKey() providerRouteKey {
	return providerRouteKey{
		capabilityID: lawupdatelist.CapabilityID,
		majorVersion: lawupdatelist.MajorVersion,
	}
}

func judicialDecisionSearchProviderRouteKey() providerRouteKey {
	return providerRouteKey{
		capabilityID: judicialdecisionsearch.CapabilityID,
		majorVersion: judicialdecisionsearch.MajorVersion,
	}
}

func judicialDecisionReadProviderRouteKey() providerRouteKey {
	return providerRouteKey{
		capabilityID: judicialdecisionread.CapabilityID,
		majorVersion: judicialdecisionread.MajorVersion,
	}
}

// ProviderRouteValues は、一つの primary route の起動時設定を保持する。
type ProviderRouteValues struct {
	CapabilityID       string
	MajorVersion       int
	Selection          string
	DefaultProviderID  string
	RollbackProviderID string
}

// ProviderRoutes は、対応能力に対する検証済みの実効 primary route を保持する。
type ProviderRoutes struct {
	registry    ProviderBindingRegistry
	providerIDs map[providerRouteKey]string
	initialized bool
}

// NewProviderRoutes は、primary と明示 rollback の binding を検証して route を返す。
func NewProviderRoutes(
	registry ProviderBindingRegistry,
	values []ProviderRouteValues,
) (ProviderRoutes, error) {
	if !registry.initialized {
		return ProviderRoutes{}, fmt.Errorf(
			"ProviderBindingRegistry は NewProviderBindingRegistry で作成しなければなりません",
		)
	}
	providerIDs := make(map[providerRouteKey]string, len(values))
	for index, value := range values {
		key := providerRouteKey{
			capabilityID: value.CapabilityID,
			majorVersion: value.MajorVersion,
		}
		if !isSupportedProviderRouteKey(key) {
			return ProviderRoutes{}, fmt.Errorf(
				"providerRoutes[%d] の capability %s@%d は登録できません",
				index,
				value.CapabilityID,
				value.MajorVersion,
			)
		}
		if value.Selection != ProviderRouteSelectionPrimary {
			return ProviderRoutes{}, fmt.Errorf(
				"providerRoutes[%d] の selection は primary でなければなりません",
				index,
			)
		}
		if _, exists := providerIDs[key]; exists {
			return ProviderRoutes{}, fmt.Errorf(
				"provider route %s@%d が重複しています",
				key.capabilityID,
				key.majorVersion,
			)
		}
		if !registry.hasBinding(
			value.DefaultProviderID,
			key,
		) {
			return ProviderRoutes{}, fmt.Errorf(
				"defaultProviderId %q に %s@%d binding がありません",
				value.DefaultProviderID,
				key.capabilityID,
				key.majorVersion,
			)
		}
		effectiveProviderID := value.DefaultProviderID
		if value.RollbackProviderID != "" {
			if !registry.hasBinding(value.RollbackProviderID, key) {
				return ProviderRoutes{}, fmt.Errorf(
					"rollbackProviderId %q に %s@%d binding がありません",
					value.RollbackProviderID,
					key.capabilityID,
					key.majorVersion,
				)
			}
			effectiveProviderID = value.RollbackProviderID
		}
		providerIDs[key] = effectiveProviderID
	}
	for _, key := range requiredProviderRouteKeys() {
		if _, exists := providerIDs[key]; !exists {
			return ProviderRoutes{}, fmt.Errorf(
				"必須の primary route %s@%d がありません",
				key.capabilityID,
				key.majorVersion,
			)
		}
	}
	return ProviderRoutes{
		registry:    registry,
		providerIDs: providerIDs,
		initialized: true,
	}, nil
}

// ProviderID は、能力と majorVersion が完全一致する実効 providerId を返す。
func (r ProviderRoutes) ProviderID(
	capabilityID string,
	majorVersion int,
) (string, bool) {
	if !r.initialized {
		return "", false
	}
	providerID, exists := r.providerIDs[providerRouteKey{
		capabilityID: capabilityID,
		majorVersion: majorVersion,
	}]
	return providerID, exists
}

// LawSearch は、law.search@1 の実効 primary port を返す。
func (r ProviderRoutes) LawSearch() (lawsearch.Port, bool) {
	providerID, exists := r.ProviderID(
		lawsearch.CapabilityID,
		lawsearch.MajorVersion,
	)
	if !exists {
		return nil, false
	}
	return r.registry.LawSearch(providerID)
}

// LawContentSearch は、law.content.search@1 の実効 primary port を返す。
func (r ProviderRoutes) LawContentSearch() (lawcontentsearch.Port, bool) {
	providerID, exists := r.ProviderID(
		lawcontentsearch.CapabilityID,
		lawcontentsearch.MajorVersion,
	)
	if !exists {
		return nil, false
	}
	return r.registry.LawContentSearch(providerID)
}

// LawDocumentRead は、law.document.read@1 の実効 primary port を返す。
func (r ProviderRoutes) LawDocumentRead() (lawdocumentread.Port, bool) {
	providerID, exists := r.ProviderID(
		lawdocumentread.CapabilityID,
		lawdocumentread.MajorVersion,
	)
	if !exists {
		return nil, false
	}
	return r.registry.LawDocumentRead(providerID)
}

// LawArticleRead は、law.article.read@1 の実効 primary port を返す。
func (r ProviderRoutes) LawArticleRead() (lawarticleread.Port, bool) {
	providerID, exists := r.ProviderID(
		lawarticleread.CapabilityID,
		lawarticleread.MajorVersion,
	)
	if !exists {
		return nil, false
	}
	return r.registry.LawArticleRead(providerID)
}

// LawUpdateList は、law.update.list@1 の実効 primary port を返す。
func (r ProviderRoutes) LawUpdateList() (lawupdatelist.Port, bool) {
	providerID, exists := r.ProviderID(
		lawupdatelist.CapabilityID,
		lawupdatelist.MajorVersion,
	)
	if !exists {
		return nil, false
	}
	return r.registry.LawUpdateList(providerID)
}

// JudicialDecisionSearch は、judicial-decision.search@1 の実効 primary port を返す。
func (r ProviderRoutes) JudicialDecisionSearch() (judicialdecisionsearch.Port, bool) {
	providerID, exists := r.ProviderID(
		judicialdecisionsearch.CapabilityID,
		judicialdecisionsearch.MajorVersion,
	)
	if !exists {
		return nil, false
	}
	return r.registry.JudicialDecisionSearch(providerID)
}

// JudicialDecisionRead は、judicial-decision.read@1 の実効 primary port を返す。
func (r ProviderRoutes) JudicialDecisionRead() (judicialdecisionread.Port, bool) {
	providerID, exists := r.ProviderID(
		judicialdecisionread.CapabilityID,
		judicialdecisionread.MajorVersion,
	)
	if !exists {
		return nil, false
	}
	return r.registry.JudicialDecisionRead(providerID)
}

func (r ProviderBindingRegistry) hasBinding(
	providerID string,
	key providerRouteKey,
) bool {
	switch key {
	case judicialDecisionReadProviderRouteKey():
		_, exists := r.JudicialDecisionRead(providerID)
		return exists
	case judicialDecisionSearchProviderRouteKey():
		_, exists := r.JudicialDecisionSearch(providerID)
		return exists
	case lawSearchProviderRouteKey():
		_, exists := r.LawSearch(providerID)
		return exists
	case lawContentSearchProviderRouteKey():
		_, exists := r.LawContentSearch(providerID)
		return exists
	case lawDocumentReadProviderRouteKey():
		_, exists := r.LawDocumentRead(providerID)
		return exists
	case lawArticleReadProviderRouteKey():
		_, exists := r.LawArticleRead(providerID)
		return exists
	case lawUpdateListProviderRouteKey():
		_, exists := r.LawUpdateList(providerID)
		return exists
	default:
		return false
	}
}

func isSupportedProviderRouteKey(key providerRouteKey) bool {
	for _, supported := range supportedProviderRouteKeys() {
		if key == supported {
			return true
		}
	}
	return false
}

func supportedProviderRouteKeys() []providerRouteKey {
	return []providerRouteKey{
		judicialDecisionReadProviderRouteKey(),
		judicialDecisionSearchProviderRouteKey(),
		lawArticleReadProviderRouteKey(),
		lawContentSearchProviderRouteKey(),
		lawDocumentReadProviderRouteKey(),
		lawSearchProviderRouteKey(),
		lawUpdateListProviderRouteKey(),
	}
}

func requiredProviderRouteKeys() []providerRouteKey {
	return []providerRouteKey{
		lawArticleReadProviderRouteKey(),
		lawContentSearchProviderRouteKey(),
		lawDocumentReadProviderRouteKey(),
		lawSearchProviderRouteKey(),
		lawUpdateListProviderRouteKey(),
	}
}
