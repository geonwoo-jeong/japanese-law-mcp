package lawv2

import (
	"math"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func validateLawSearchPage(
	response lawSearchResponse,
	nextOffset responseNextOffset,
	limit int,
	offset int,
) (*int, error) {
	if response.totalCount < 0 ||
		response.count < 0 ||
		response.count > limit ||
		len(response.laws) != response.count {
		return nil, newSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
	if offset < 0 || response.count > maxInt()-offset {
		return nil, newSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
	endOffset := offset + response.count
	if endOffset > response.totalCount &&
		(offset <= response.totalCount || response.count != 0) {
		return nil, newSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}

	hasRemaining := endOffset < response.totalCount
	if nextOffset.present && nextOffset.isNull {
		if hasRemaining {
			return nil, newSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
		}
		return nil, nil
	}
	if nextOffset.present {
		if nextOffset.value != endOffset ||
			nextOffset.value <= offset ||
			nextOffset.value < 0 ||
			nextOffset.value > math.MaxInt32 ||
			nextOffset.value > response.totalCount ||
			!hasRemaining {
			return nil, newSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
		}
		value := nextOffset.value
		return &value, nil
	}
	if !hasRemaining {
		return nil, nil
	}
	if endOffset <= offset || endOffset > math.MaxInt32 {
		return nil, newSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
	value := endOffset
	return &value, nil
}
