package legalquerycandidateprepare

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateprofile"
)

func buildComposition(
	profileSet legalquerycandidateeval.ProfileSetIdentity,
	profiles []legalquerycandidateeval.ProfileArtifact,
	candidate legalquerycandidateprofile.Set,
) (legalquerycandidateeval.CompositionDescriptor, error) {
	components := []legalquerycandidateeval.CompositionComponent{{
		Role:            "preprocessor",
		ComponentID:     "query-preprocessor",
		SemanticVersion: queryPreprocessorSemanticVersion,
		PackageRoot:     "internal/querypreprocess",
	}}
	for _, profile := range profiles {
		packageRoot, err := profilePackageRoot(profile.ProfileID)
		if err != nil {
			return legalquerycandidateeval.CompositionDescriptor{}, err
		}
		components = append(components, legalquerycandidateeval.CompositionComponent{
			Role:            "profile",
			ComponentID:     profile.ProfileID,
			SemanticVersion: profile.ProfileVersion,
			PackageRoot:     packageRoot,
		})
	}
	compositionVersion := candidate.Profiles().CompositionVersion()
	components = append(components,
		legalquerycandidateeval.CompositionComponent{
			Role:            "composer",
			ComponentID:     "candidate-composer",
			SemanticVersion: compositionVersion,
			PackageRoot:     "internal/application/legalquery",
		},
		legalquerycandidateeval.CompositionComponent{
			Role:            "selector",
			ComponentID:     "legal-query-selector",
			SemanticVersion: selectorSemanticVersion,
			PackageRoot:     "internal/application/legalquery",
		},
	)
	descriptor := legalquerycandidateeval.CompositionDescriptor{
		DescriptorSchemaVersion: compositionDescriptorSchemaV2,
		ProfileSetID:            profileSet.ProfileSetID,
		ProfileSetVersion:       profileSet.ProfileSetVersion,
		RankingVersion:          profileSet.RankingVersion,
		CompositionVersion:      compositionVersion,
		Components:              components,
	}
	var err error
	descriptor.DescriptorSHA256, err =
		legalquerycandidateeval.CanonicalCompositionSHA256(descriptor)
	if err != nil {
		return legalquerycandidateeval.CompositionDescriptor{}, err
	}
	return descriptor, nil
}

func profilePackageRoot(profileID string) (string, error) {
	switch profileID {
	case "core":
		return "internal/queryprofile/core", nil
	case "judicial-cases":
		return "internal/queryprofile/judicialcases", nil
	default:
		return "", fmt.Errorf("候補 profile %q の package root は許可されていません", profileID)
	}
}
