package main

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryadoption"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval/defaultprofile"
)

func verifyAdoptionProfileIdentity(
	adoption legalqueryadoption.Manifest,
	identity defaultprofile.Identity,
) error {
	if adoption.ProfileSetID() != identity.ProfileSetID() ||
		adoption.ProfileSetVersion() != identity.ProfileSetVersion() ||
		adoption.RankingVersion() != identity.RankingVersion() ||
		adoption.CompositionVersion() != identity.CompositionVersion() {
		return fmt.Errorf("current adoption と production profile set が一致しません")
	}
	adoptedProfiles := adoption.Profiles()
	runtimeProfiles := identity.Profiles()
	if len(adoptedProfiles) != len(runtimeProfiles) {
		return fmt.Errorf("current adoption の profile 件数が一致しません")
	}
	for index := range adoptedProfiles {
		if adoptedProfiles[index].ProfileID() != runtimeProfiles[index].ProfileID() ||
			adoptedProfiles[index].ProfileVersion() != runtimeProfiles[index].ProfileVersion() ||
			adoptedProfiles[index].CueSetVersion() != runtimeProfiles[index].CueSetVersion() {
			return fmt.Errorf("current adoption の profile 固定順が一致しません")
		}
	}
	return nil
}

func verifyAdoptionCorpusIdentity(
	adoption legalqueryadoption.Manifest,
	corpus legalquerycorpus.Corpus,
) error {
	manifest := corpus.Manifest()
	if manifest.CorpusVersion() != adoption.CorpusVersion() ||
		manifest.HoldoutDigest() != adoption.HoldoutDigest() {
		return fmt.Errorf("current adoption と corpus manifest が一致しません")
	}
	return nil
}

func verifyAdoptionBaselineIdentity(
	adoption legalqueryadoption.Manifest,
	artifact legalqueryeval.BaselineArtifact,
) error {
	report := artifact.Report()
	if artifact.SHA256() != adoption.BaselineSHA256() ||
		report.BaselineVersion() != adoption.BaselineVersion() ||
		report.CorpusVersion() != adoption.CorpusVersion() ||
		report.HoldoutDigest() != adoption.HoldoutDigest() {
		return fmt.Errorf("current adoption と baseline が一致しません")
	}
	profileSet := report.ProfileSet()
	if profileSet.ProfileSetID() != adoption.ProfileSetID() ||
		profileSet.ProfileSetVersion() != adoption.ProfileSetVersion() ||
		profileSet.RankingVersion() != adoption.RankingVersion() {
		return fmt.Errorf("current adoption と baseline profile set が一致しません")
	}
	profiles := profileSet.Profiles()
	adoptedProfiles := adoption.Profiles()
	if len(profiles) != len(adoptedProfiles) {
		return fmt.Errorf("baseline の profile 件数が current adoption と一致しません")
	}
	for index := range profiles {
		if profiles[index].ProfileID() != adoptedProfiles[index].ProfileID() ||
			profiles[index].ProfileVersion() != adoptedProfiles[index].ProfileVersion() {
			return fmt.Errorf("baseline の profile 固定順が current adoption と一致しません")
		}
	}
	return nil
}
