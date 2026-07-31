package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

type coreNormalizationIdentity struct {
	purpose string
	target  string
}

func withCoreNormalizationGroups(
	value profileevidence.DraftValues,
	draft candidateDraft,
	facts map[string]coreEvidenceFact,
) (profileevidence.DraftValues, error) {
	if len(value.Steps) != len(draft.steps) {
		return profileevidence.DraftValues{}, fmt.Errorf(
			"正規化対象の logical step 対応が一致しません",
		)
	}
	for stepIndex := range value.Steps {
		step := &value.Steps[stepIndex]
		for evidenceIndex := range step.Evidence {
			step.Evidence[evidenceIndex].NormalizationGroup = ""
		}
		identities := make([]coreNormalizationIdentity, len(step.Evidence))
		eligible := make([]bool, len(step.Evidence))
		factsByIdentity := make(map[coreNormalizationIdentity]map[string]struct{})
		for evidenceIndex, evidence := range step.Evidence {
			fact, exists := facts[evidence.FactID]
			if !exists || fact.values.Span == nil {
				continue
			}
			identity, ok, err := coreEvidenceNormalizationIdentity(
				draft.steps[stepIndex].input,
				evidence,
				fact,
			)
			if err != nil {
				return profileevidence.DraftValues{}, err
			}
			if !ok {
				continue
			}
			identities[evidenceIndex] = identity
			eligible[evidenceIndex] = true
			if factsByIdentity[identity] == nil {
				factsByIdentity[identity] = make(map[string]struct{})
			}
			factsByIdentity[identity][evidence.FactID] = struct{}{}
		}
		for evidenceIndex := range step.Evidence {
			identity := identities[evidenceIndex]
			if !eligible[evidenceIndex] ||
				len(factsByIdentity[identity]) < 2 {
				continue
			}
			step.Evidence[evidenceIndex].NormalizationGroup =
				coreNormalizationGroupID(
					step.StepMeaningSignature,
					step.TopicOrdinal,
					identity,
				)
		}
	}
	return value, nil
}

func coreEvidenceNormalizationIdentity(
	input legalquery.LogicalInput,
	evidence profileevidence.EvidenceValues,
	fact coreEvidenceFact,
) (coreNormalizationIdentity, bool, error) {
	if evidence.Layer == profileevidence.LayerBoundary ||
		evidence.Layer == profileevidence.LayerExplicitTaskResource {
		return coreNormalizationIdentity{}, false, nil
	}
	signature, err := logicalInputSignature(input)
	if err != nil {
		return coreNormalizationIdentity{}, false, err
	}
	switch current := input.(type) {
	case legalquery.LawSearchIntentV1:
		if fact.kind == coreEvidenceFactDate {
			return coreNormalizationIdentity{
				purpose: "as-of",
				target:  signature,
			}, true, nil
		}
		return coreNormalizationIdentity{
			purpose: "law-search-target",
			target:  current.Query(),
		}, true, nil
	case legalquery.LawContentSearchIntentV1:
		if fact.kind == coreEvidenceFactDate {
			return coreNormalizationIdentity{
				purpose: "as-of",
				target:  signature,
			}, true, nil
		}
		purpose, target, matched, matchErr :=
			coreContentNormalizationTerm(current, fact)
		if matchErr != nil {
			return coreNormalizationIdentity{}, false, matchErr
		}
		return coreNormalizationIdentity{
			purpose: purpose,
			target:  target,
		}, matched, nil
	case legalquery.LawReadIntentV1:
		if fact.kind == coreEvidenceFactDate {
			return coreNormalizationIdentity{
				purpose: "as-of",
				target:  signature,
			}, true, nil
		}
		return coreNormalizationIdentity{
			purpose: "law-target",
			target:  signature,
		}, true, nil
	case legalquery.LawArticleReadIntentV1:
		purpose := "law-target"
		switch fact.kind {
		case coreEvidenceFactDate:
			purpose = "as-of"
		case coreEvidenceFactArticle:
			purpose = "article-location"
		case coreEvidenceFactParagraph:
			purpose = "paragraph-location"
		}
		return coreNormalizationIdentity{
			purpose: purpose,
			target:  signature,
		}, true, nil
	case legalquery.LawUpdateListIntentV1:
		return coreNormalizationIdentity{
			purpose: "update-date",
			target:  current.Date().String(),
		}, true, nil
	default:
		return coreNormalizationIdentity{}, false, fmt.Errorf(
			"core evidence 正規化が未対応の logical input を受け取りました",
		)
	}
}

func coreContentNormalizationTerm(
	input legalquery.LawContentSearchIntentV1,
	fact coreEvidenceFact,
) (string, string, bool, error) {
	type termGroup struct {
		purpose string
		values  []string
	}
	groups := []termGroup{
		{purpose: "all-term", values: input.AllTerms()},
		{purpose: "any-term", values: input.AnyTerms()},
		{purpose: "exclude-term", values: input.ExcludeTerms()},
	}
	if fact.kind == coreEvidenceFactLegalConcept {
		termCount := 0
		var purpose string
		var target string
		for _, group := range groups {
			for _, value := range group.values {
				termCount++
				purpose = group.purpose
				target = value
			}
		}
		if termCount == 1 {
			return purpose, target, true, nil
		}
		return "", "", false, fmt.Errorf(
			"法概念 fact %q の本文検索 target と purpose を一意に正規化できません",
			fact.values.FactID,
		)
	}
	var purpose string
	var target string
	matches := 0
	for _, group := range groups {
		for _, value := range group.values {
			if value != fact.surface {
				continue
			}
			matches++
			purpose = group.purpose
			target = value
		}
	}
	if matches > 1 {
		return "", "", false, fmt.Errorf(
			"fact %q の本文検索 target を一意に正規化できません",
			fact.values.FactID,
		)
	}
	return purpose, target, matches == 1, nil
}

func coreNormalizationGroupID(
	stepMeaningSignature string,
	topicOrdinal int,
	identity coreNormalizationIdentity,
) string {
	payload := strconv.Itoa(len(stepMeaningSignature)) + ":" +
		stepMeaningSignature + ":" + strconv.Itoa(topicOrdinal) + ":" +
		strconv.Itoa(len(identity.purpose)) + ":" + identity.purpose + ":" +
		strconv.Itoa(len(identity.target)) + ":" + identity.target
	digest := sha256.Sum256([]byte(payload))
	return "group-" + hex.EncodeToString(digest[:29])
}
