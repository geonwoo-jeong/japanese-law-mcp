package application

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func validateLawSearchFacadeResult(
	result lawsearch.Page,
	request lawsearch.Request,
	binding ProviderBindingMetadata,
) error {
	if err := result.Validate(); err != nil {
		return fmt.Errorf("law.search@1 の結果が有効ではありません: %w", err)
	}
	items := result.Items()
	if len(items) > request.Limit() {
		return fmt.Errorf("law.search@1 の結果が request.limit を超えています")
	}
	for index, item := range items {
		if err := validateCoreFacadeRef(item.Ref(), binding); err != nil {
			return fmt.Errorf(
				"law.search@1 の items[%d] が選択済み binding と一致しません: %w",
				index,
				err,
			)
		}
	}
	return nil
}

func validateLawContentFacadeResult(
	result lawcontentsearch.Page,
	request lawcontentsearch.Request,
	binding ProviderBindingMetadata,
) error {
	if err := result.Validate(); err != nil {
		return fmt.Errorf(
			"law.content.search@1 の結果が有効ではありません: %w",
			err,
		)
	}
	items := result.Items()
	if len(items) > request.Limit() {
		return fmt.Errorf(
			"law.content.search@1 の結果が request.limit を超えています",
		)
	}
	for index, item := range items {
		if err := validateCoreFacadeRef(item.Ref(), binding); err != nil {
			return fmt.Errorf(
				"law.content.search@1 の items[%d] が選択済み binding と一致しません: %w",
				index,
				err,
			)
		}
	}
	return nil
}

func validateLawUpdateFacadeResult(
	result lawupdatelist.Page,
	request lawupdatelist.Request,
	binding ProviderBindingMetadata,
) error {
	if err := result.Validate(); err != nil {
		return fmt.Errorf("law.update.list@1 の結果が有効ではありません: %w", err)
	}
	if result.Date() != request.Date() {
		return fmt.Errorf("law.update.list@1 の結果日付が request.date と一致しません")
	}
	for index, item := range result.Items() {
		if err := validateCoreFacadeRef(item.Ref(), binding); err != nil {
			return fmt.Errorf(
				"law.update.list@1 の items[%d] が選択済み binding と一致しません: %w",
				index,
				err,
			)
		}
	}
	return nil
}

func validateLawDocumentFacadeResult(
	result model.SourcedResource[model.LawDocumentRepresentation],
	request lawdocumentread.Request,
	binding ProviderBindingMetadata,
) error {
	if err := validateCoreFacadeResource(
		lawdocumentread.CapabilityID+"@1",
		result,
		binding,
	); err != nil {
		return err
	}
	if err := validateExactLawRef(
		result.Ref(),
		request.Resource(),
		lawdocumentread.CapabilityID+"@1",
	); err != nil {
		return err
	}
	law := result.Data().Law()
	if law.LawID() != request.Resource().Key().ResourceID() {
		return fmt.Errorf(
			"law.document.read@1 の結果 lawId が request.resource と一致しません",
		)
	}
	if err := validateResolvedRevision(
		result.Ref(),
		law.RevisionID(),
		lawdocumentread.CapabilityID+"@1",
	); err != nil {
		return err
	}
	requestAsOf, requestHasAsOf := request.AsOf()
	resultAsOf, resultHasAsOf := result.Data().AsOf()
	if requestHasAsOf != resultHasAsOf ||
		(requestHasAsOf && requestAsOf != resultAsOf) {
		return fmt.Errorf(
			"law.document.read@1 の結果 asOf が request.asOf と一致しません",
		)
	}
	return nil
}

func validateLawArticleFacadeResult(
	result model.SourcedResource[model.LawArticleFragment],
	request lawarticleread.Request,
	binding ProviderBindingMetadata,
) error {
	if err := validateCoreFacadeResource(
		lawarticleread.CapabilityID+"@1",
		result,
		binding,
	); err != nil {
		return err
	}
	if err := validateExactLawRef(
		result.Ref(),
		request.Resource(),
		lawarticleread.CapabilityID+"@1",
	); err != nil {
		return err
	}
	data := result.Data()
	if data.Law().LawID() != request.Resource().Key().ResourceID() {
		return fmt.Errorf(
			"law.article.read@1 の結果 lawId が request.resource と一致しません",
		)
	}
	if err := validateResolvedRevision(
		result.Ref(),
		data.Law().RevisionID(),
		lawarticleread.CapabilityID+"@1",
	); err != nil {
		return err
	}
	if !sameLawArticleLocation(data.Location(), request.Location()) {
		return fmt.Errorf(
			"law.article.read@1 の結果 location が request.location と一致しません",
		)
	}
	return nil
}

func validateCoreFacadeResource[T interface{ Validate() error }](
	capabilityID string,
	result model.SourcedResource[T],
	binding ProviderBindingMetadata,
) error {
	if err := result.Validate(); err != nil {
		return fmt.Errorf("%s の結果が有効ではありません: %w", capabilityID, err)
	}
	if err := validateCoreFacadeRef(result.Ref(), binding); err != nil {
		return fmt.Errorf(
			"%s の結果が選択済み binding と一致しません: %w",
			capabilityID,
			err,
		)
	}
	return nil
}

func validateCoreFacadeRef(
	ref model.SourceResourceRef,
	binding ProviderBindingMetadata,
) error {
	if ref.ProviderID() != binding.ProviderID() {
		return fmt.Errorf("ref.providerId が一致しません")
	}
	if ref.Key().SourceID() != binding.SourceID() {
		return fmt.Errorf("ref.key.sourceId が一致しません")
	}
	return nil
}

func validateExactLawRef(
	resultRef model.SourceResourceRef,
	requestRef model.SourceResourceRef,
	capabilityID string,
) error {
	resultKey := resultRef.Key()
	requestKey := requestRef.Key()
	if resultKey.ResourceType() != requestKey.ResourceType() {
		return fmt.Errorf("%s の結果 resourceType が request.resource と一致しません", capabilityID)
	}
	if resultKey.ResourceID() != requestKey.ResourceID() {
		return fmt.Errorf("%s の結果 resourceId が request.resource と一致しません", capabilityID)
	}
	requestVersion, requestHasVersion := requestKey.VersionID()
	if !requestHasVersion {
		return nil
	}
	resultVersion, resultHasVersion := resultKey.VersionID()
	if !resultHasVersion || resultVersion != requestVersion {
		return fmt.Errorf("%s の結果 versionId が request.resource と一致しません", capabilityID)
	}
	return nil
}

func validateResolvedRevision(
	ref model.SourceResourceRef,
	revisionID string,
	capabilityID string,
) error {
	refRevisionID, exists := ref.Key().VersionID()
	if !exists {
		return fmt.Errorf("%s の結果 ref.versionId がありません", capabilityID)
	}
	if refRevisionID != revisionID {
		return fmt.Errorf("%s の結果 ref.versionId と data.law.revisionId が一致しません", capabilityID)
	}
	return nil
}

func sameLawArticleLocation(
	left model.LawArticleLocation,
	right model.LawArticleLocation,
) bool {
	leftParagraph, leftExists := left.ParagraphNumber()
	rightParagraph, rightExists := right.ParagraphNumber()
	return left.Provision() == right.Provision() &&
		left.ArticleNumber() == right.ArticleNumber() &&
		leftExists == rightExists &&
		(!leftExists || leftParagraph == rightParagraph)
}
