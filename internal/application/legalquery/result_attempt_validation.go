package legalquery

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func validateLawSearchAttemptItems(
	items []model.SourcedResource[model.LawSummary],
) error {
	for index, item := range items {
		if err := validateLawResourceIdentity(item.Ref(), item.Data()); err != nil {
			return fmt.Errorf("items[%d] が law.search@1 と一致しません: %w", index, err)
		}
	}
	return nil
}

func validateLawContentAttemptItems(
	items []model.SourcedResource[model.LawContentMatch],
) error {
	type contentIdentity struct {
		ref      model.SourceResourceRef
		location string
	}
	seen := make(map[contentIdentity]struct{}, len(items))
	for index, item := range items {
		data := item.Data()
		if err := validateLawResourceIdentity(item.Ref(), data.Law()); err != nil {
			return fmt.Errorf(
				"items[%d] が law.content.search@1 と一致しません: %w",
				index,
				err,
			)
		}
		provenance := item.Provenance()
		location, exists := provenance[len(provenance)-1].Location()
		if !exists || location != data.Location() {
			return fmt.Errorf(
				"items[%d] の最後の provenance.location と data.location が一致しません",
				index,
			)
		}
		identity := contentIdentity{
			ref:      item.Ref(),
			location: data.Location(),
		}
		if _, exists := seen[identity]; exists {
			return fmt.Errorf("items[%d] の ref と location の組が重複しています", index)
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func validateLawUpdatesAttemptItems(
	header legalQueryAttemptHeader,
	items []model.SourcedResource[model.LawUpdate],
) error {
	input, ok := header.logicalInput.(LawUpdateListIntentV1)
	if !ok {
		return fmt.Errorf("law.update.list@1 の logical input が一致しません")
	}
	date := input.Date()
	for index, item := range items {
		key := item.Ref().Key()
		data := item.Data()
		switch {
		case key.ResourceType() != "law-update-list":
			return fmt.Errorf(
				"items[%d].ref.key.resourceType は law-update-list でなければなりません",
				index,
			)
		case data.UpdatedOn() != date:
			return fmt.Errorf("items[%d].data.updatedOn と要求日が一致しません", index)
		case key.ResourceID() != date.String():
			return fmt.Errorf("items[%d].ref.key.resourceId と要求日が一致しません", index)
		case key.SourceID() != data.Source().ID():
			return fmt.Errorf(
				"items[%d].ref.key.sourceId と data.source.id が一致しません",
				index,
			)
		}
		if _, exists := key.VersionID(); exists {
			return fmt.Errorf("items[%d].ref.key.versionId は指定できません", index)
		}
	}
	return nil
}

func validateJudicialSearchAttemptItems(
	items []model.SourcedResource[model.JudicialDecisionSummary],
) error {
	for index, item := range items {
		if err := validateJudicialResourceIdentity(
			item.Ref(),
			item.Data().Source(),
		); err != nil {
			return fmt.Errorf(
				"items[%d] が judicial-decision.search@1 と一致しません: %w",
				index,
				err,
			)
		}
	}
	return nil
}

func validateLawDocumentAttemptItem(
	header legalQueryAttemptHeader,
	item model.SourcedResource[model.LawDocumentRepresentation],
) error {
	law := item.Data().Law()
	if err := validateLawResourceIdentity(item.Ref(), law); err != nil {
		return fmt.Errorf("item が law.document.read@1 と一致しません: %w", err)
	}
	input, ok := header.logicalInput.(LawReadIntentV1)
	if !ok {
		return fmt.Errorf("law.document.read@1 の logical input が一致しません")
	}
	if err := validateLawReadTarget(
		item.Ref(),
		law,
		input,
	); err != nil {
		return fmt.Errorf("item が読取り対象と一致しません: %w", err)
	}
	return nil
}

func validateLawArticleAttemptItem(
	header legalQueryAttemptHeader,
	item model.SourcedResource[model.LawArticleFragment],
) error {
	data := item.Data()
	law := data.Law()
	if err := validateLawResourceIdentity(item.Ref(), law); err != nil {
		return fmt.Errorf("item が law.article.read@1 と一致しません: %w", err)
	}
	input, ok := header.logicalInput.(LawArticleReadIntentV1)
	if !ok {
		return fmt.Errorf("law.article.read@1 の logical input が一致しません")
	}
	if err := validateLawArticleReadTarget(
		item.Ref(),
		law,
		data.Location(),
		input,
	); err != nil {
		return fmt.Errorf("item が条文読取り対象と一致しません: %w", err)
	}
	return nil
}

func validateJudicialDecisionAttemptItem(
	header legalQueryAttemptHeader,
	item model.SourcedResource[model.JudicialDecisionDetails],
) error {
	summary := item.Data().Summary()
	if err := validateJudicialResourceIdentity(
		item.Ref(),
		summary.Source(),
	); err != nil {
		return fmt.Errorf("item が judicial-decision.read@1 と一致しません: %w", err)
	}
	input, ok := header.logicalInput.(JudicialDecisionReadIntentV1)
	if !ok {
		return fmt.Errorf("judicial-decision.read@1 の logical input が一致しません")
	}
	if !sameSourceResourceRef(item.Ref(), input.Ref()) {
		return fmt.Errorf("item.ref が裁判例読取り対象と一致しません")
	}
	return nil
}

func validateLawResourceIdentity(
	ref model.SourceResourceRef,
	law model.LawSummary,
) error {
	key := ref.Key()
	switch {
	case key.ResourceType() != "law":
		return fmt.Errorf("ref.key.resourceType は law でなければなりません")
	case key.ResourceID() != law.LawID():
		return fmt.Errorf("ref.key.resourceId と data.lawId が一致しません")
	case key.SourceID() != law.Source().ID():
		return fmt.Errorf("ref.key.sourceId と data.source.id が一致しません")
	}
	versionID, exists := key.VersionID()
	if !exists || versionID != law.RevisionID() {
		return fmt.Errorf("ref.key.versionId と data.revisionId が一致しません")
	}
	return nil
}

func validateJudicialResourceIdentity(
	ref model.SourceResourceRef,
	source model.InformationSource,
) error {
	key := ref.Key()
	switch {
	case key.ResourceType() != "judicial-decision":
		return fmt.Errorf("ref.key.resourceType は judicial-decision でなければなりません")
	case key.SourceID() != source.ID():
		return fmt.Errorf("ref.key.sourceId と data.source.id が一致しません")
	}
	if _, exists := key.VersionID(); exists {
		return fmt.Errorf("ref.key.versionId は指定できません")
	}
	return nil
}

func validateLawReadTarget(
	outputRef model.SourceResourceRef,
	law model.LawSummary,
	input LawReadIntentV1,
) error {
	if inputRef, exists := input.Ref(); exists {
		return validateCompatibleLawRef(outputRef, inputRef)
	}
	lawID, _ := input.LawID()
	if law.LawID() != lawID {
		return fmt.Errorf("data.lawId が logical input と一致しません")
	}
	if revisionID, exists := input.RevisionID(); exists &&
		law.RevisionID() != revisionID {
		return fmt.Errorf("data.revisionId が logical input と一致しません")
	}
	return nil
}

func validateLawArticleReadTarget(
	outputRef model.SourceResourceRef,
	law model.LawSummary,
	location model.LawArticleLocation,
	input LawArticleReadIntentV1,
) error {
	if inputRef, exists := input.Ref(); exists {
		if err := validateCompatibleLawRef(outputRef, inputRef); err != nil {
			return err
		}
	} else {
		lawID, _ := input.LawID()
		if law.LawID() != lawID {
			return fmt.Errorf("data.lawId が logical input と一致しません")
		}
	}
	if !sameLawArticleLocation(location, input.Location()) {
		return fmt.Errorf("data.location が logical input と一致しません")
	}
	return nil
}

func validateCompatibleLawRef(
	output model.SourceResourceRef,
	input model.SourceResourceRef,
) error {
	outputKey := output.Key()
	inputKey := input.Key()
	if output.ProviderID() != input.ProviderID() ||
		outputKey.SourceID() != inputKey.SourceID() ||
		outputKey.ResourceType() != inputKey.ResourceType() ||
		outputKey.ResourceID() != inputKey.ResourceID() {
		return fmt.Errorf("item.ref が法令読取り対象と一致しません")
	}
	inputVersion, inputHasVersion := inputKey.VersionID()
	outputVersion, outputHasVersion := outputKey.VersionID()
	if inputHasVersion &&
		(!outputHasVersion || outputVersion != inputVersion) {
		return fmt.Errorf("item.ref.key.versionId が法令読取り対象と一致しません")
	}
	return nil
}

func sameSourceResourceRef(
	left model.SourceResourceRef,
	right model.SourceResourceRef,
) bool {
	if left.ProviderID() != right.ProviderID() {
		return false
	}
	leftKey := left.Key()
	rightKey := right.Key()
	leftVersion, leftHasVersion := leftKey.VersionID()
	rightVersion, rightHasVersion := rightKey.VersionID()
	return leftKey.SourceID() == rightKey.SourceID() &&
		leftKey.ResourceType() == rightKey.ResourceType() &&
		leftKey.ResourceID() == rightKey.ResourceID() &&
		leftHasVersion == rightHasVersion &&
		leftVersion == rightVersion
}

func sameLawArticleLocation(
	left model.LawArticleLocation,
	right model.LawArticleLocation,
) bool {
	if left.Provision() != right.Provision() ||
		left.ArticleNumber() != right.ArticleNumber() {
		return false
	}
	leftParagraph, leftHasParagraph := left.ParagraphNumber()
	rightParagraph, rightHasParagraph := right.ParagraphNumber()
	return leftHasParagraph == rightHasParagraph &&
		(!leftHasParagraph || leftParagraph == rightParagraph)
}
