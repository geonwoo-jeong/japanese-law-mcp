package application

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func validateJudicialDecisionSearchFacadeResult(
	result judicialdecisionsearch.Page,
	request judicialdecisionsearch.Request,
	binding ProviderBindingMetadata,
) error {
	if err := result.Validate(); err != nil {
		return fmt.Errorf(
			"judicial-decision.search@1 の結果が有効ではありません: %w",
			err,
		)
	}
	items := result.Items()
	if len(items) > request.Limit() {
		return fmt.Errorf(
			"judicial-decision.search@1 の結果が request.limit を超えています",
		)
	}
	for index, item := range items {
		if err := validateLegalQueryFacadeRef(item.Ref(), binding); err != nil {
			return fmt.Errorf(
				"judicial-decision.search@1 の items[%d] が選択済み binding と一致しません: %w",
				index,
				err,
			)
		}
	}
	return nil
}

func validateJudicialDecisionReadFacadeResult(
	result model.SourcedResource[model.JudicialDecisionDetails],
	request judicialdecisionread.Request,
	binding ProviderBindingMetadata,
) error {
	if err := result.Validate(); err != nil {
		return fmt.Errorf(
			"judicial-decision.read@1 の結果が有効ではありません: %w",
			err,
		)
	}
	if err := validateLegalQueryFacadeRef(result.Ref(), binding); err != nil {
		return fmt.Errorf(
			"judicial-decision.read@1 の結果が選択済み binding と一致しません: %w",
			err,
		)
	}
	if result.Ref() != request.Ref() {
		return fmt.Errorf(
			"judicial-decision.read@1 の結果 ref が request.ref と一致しません",
		)
	}
	return nil
}
