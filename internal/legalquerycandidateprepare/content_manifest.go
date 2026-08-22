// Package legalquerycandidateprepare は、holdout を開かずに候補評価成果物を準備する。
package legalquerycandidateprepare

import (
	"context"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateprofile"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/metadataartifact"
)

const (
	profileSetID                     = "default"
	compositionDescriptorSchemaV2    = 2
	queryPreprocessorSemanticVersion = "query-preprocessor-v2"
	selectorSemanticVersion          = "legal-query-selector-v1"
)

type profileArtifactSource struct {
	profileID    string
	metadataPath string
	cuesPath     string
}

var candidateProfileSources = [...]profileArtifactSource{
	{
		profileID:    "core",
		metadataPath: "internal/legalquerycandidateartifact/data/core/profile.json",
		cuesPath:     "internal/legalquerycandidateartifact/data/core/cues.json",
	},
	{
		profileID:    "judicial-cases",
		metadataPath: "internal/legalquerycandidateartifact/data/judicial-cases/profile.json",
		cuesPath:     "internal/legalquerycandidateartifact/data/judicial-cases/cues.json",
	},
}

// BuildContentManifest は、候補 profile、辞書、composition と source set を固定する。
func BuildContentManifest(
	ctx context.Context,
	repositoryRoot string,
	sourceSet legalquerycandidateeval.SemanticSourceSet,
) (legalquerycandidateeval.CandidateContentManifest, error) {
	if ctx == nil {
		return legalquerycandidateeval.CandidateContentManifest{},
			fmt.Errorf("候補準備 context は nil にできません")
	}
	if err := ctx.Err(); err != nil {
		return legalquerycandidateeval.CandidateContentManifest{},
			fmt.Errorf("候補準備が中止されました: %w", err)
	}
	repository, err := openPrepareRepository(repositoryRoot)
	if err != nil {
		return legalquerycandidateeval.CandidateContentManifest{}, err
	}
	defer func() { _ = repository.Close() }()
	candidate, err := legalquerycandidateprofile.Load()
	if err != nil {
		return legalquerycandidateeval.CandidateContentManifest{}, err
	}
	profiles, err := buildProfileArtifacts(ctx, repository, candidate)
	if err != nil {
		return legalquerycandidateeval.CandidateContentManifest{}, err
	}
	profileSet := legalquerycandidateeval.ProfileSetIdentity{
		ProfileSetID:      profileSetID,
		ProfileSetVersion: candidate.Profiles().ProfileVersion(),
		RankingVersion:    candidate.Profiles().RankingVersion(),
	}
	lexicons, err := buildLexiconArtifacts(ctx, repository, candidate)
	if err != nil {
		return legalquerycandidateeval.CandidateContentManifest{}, err
	}
	composition, err := buildComposition(profileSet, profiles, candidate)
	if err != nil {
		return legalquerycandidateeval.CandidateContentManifest{}, err
	}
	manifest := legalquerycandidateeval.CandidateContentManifest{
		ArtifactKind:      legalquerycandidateeval.ArtifactKindCandidateContent,
		SchemaVersion:     legalquerycandidateeval.SchemaVersionV3,
		ProfileSet:        profileSet,
		ProfileArtifacts:  profiles,
		LexiconArtifacts:  lexicons,
		Composition:       composition,
		SemanticSourceSet: sourceSet,
	}
	manifest.CandidateContentID, err =
		legalquerycandidateeval.CanonicalCandidateContentID(manifest)
	if err != nil {
		return legalquerycandidateeval.CandidateContentManifest{}, err
	}
	return manifest, nil
}

func buildProfileArtifacts(
	ctx context.Context,
	repository *prepareRepository,
	candidate legalquerycandidateprofile.Set,
) ([]legalquerycandidateeval.ProfileArtifact, error) {
	metadata := candidate.ProfileMetadata()
	if len(metadata) != len(candidateProfileSources) {
		return nil, fmt.Errorf("候補 profile の件数が固定集合と一致しません")
	}
	result := make([]legalquerycandidateeval.ProfileArtifact, 0, len(metadata))
	for index, source := range candidateProfileSources {
		if err := checkPrepareContext(ctx); err != nil {
			return nil, err
		}
		artifact, err := buildProfileArtifact(repository, source, metadata[index])
		if err != nil {
			return nil, err
		}
		result = append(result, artifact)
	}
	return result, nil
}

func buildProfileArtifact(
	repository *prepareRepository,
	source profileArtifactSource,
	metadata interface {
		ProfileID() string
		ProfileVersion() string
		SchemaVersion() int
		CueSetVersion() string
	},
) (legalquerycandidateeval.ProfileArtifact, error) {
	metadataRaw, err := repository.Read(source.metadataPath, 64<<10)
	if err != nil {
		return legalquerycandidateeval.ProfileArtifact{}, err
	}
	parsed, err := metadataartifact.Load(metadataRaw)
	if err != nil {
		return legalquerycandidateeval.ProfileArtifact{}, err
	}
	if parsed.Metadata().ProfileID() != source.profileID ||
		metadata.ProfileID() != source.profileID ||
		parsed.Metadata().ProfileVersion() != metadata.ProfileVersion() {
		return legalquerycandidateeval.ProfileArtifact{},
			fmt.Errorf("候補 profile metadata の identity が一致しません")
	}
	cuesRaw, err := repository.Read(source.cuesPath, 1<<20)
	if err != nil {
		return legalquerycandidateeval.ProfileArtifact{}, err
	}
	return legalquerycandidateeval.ProfileArtifact{
		ProfileID:               metadata.ProfileID(),
		ProfileVersion:          metadata.ProfileVersion(),
		MetadataSchemaVersion:   metadata.SchemaVersion(),
		MetadataCanonicalSHA256: legalquerycandidateeval.RawSHA256(parsed.CanonicalBytes()),
		CueSetVersion:           metadata.CueSetVersion(),
		CueArtifactSHA256:       legalquerycandidateeval.RawSHA256(cuesRaw),
	}, nil
}
