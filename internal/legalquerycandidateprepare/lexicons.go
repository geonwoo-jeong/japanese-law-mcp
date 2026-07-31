package legalquerycandidateprepare

import (
	"context"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateprofile"
)

type lexiconSource struct {
	lexiconID string
	version   func(legalquerycandidateprofile.Set) (string, error)
	paths     []string
}

var candidateLexiconSources = [...]lexiconSource{
	{
		lexiconID: "lawNames",
		version: func(set legalquerycandidateprofile.Set) (string, error) {
			return sharedLexiconVersion(set, true)
		},
		paths: []string{
			"internal/lawnamelexicon/data/egov-current.json",
			"internal/lawnamelexicon/data/supplemental.json",
		},
	},
	{
		lexiconID: "legalConcepts",
		version: func(set legalquerycandidateprofile.Set) (string, error) {
			return sharedLexiconVersion(set, false)
		},
		paths: []string{"internal/legalconceptlexicon/data/current.json"},
	},
}

func buildLexiconArtifacts(
	ctx context.Context,
	repository *prepareRepository,
	candidate legalquerycandidateprofile.Set,
) ([]legalquerycandidateeval.LexiconArtifact, error) {
	result := make([]legalquerycandidateeval.LexiconArtifact, 0, len(candidateLexiconSources))
	for _, source := range candidateLexiconSources {
		if err := checkPrepareContext(ctx); err != nil {
			return nil, err
		}
		version, err := source.version(candidate)
		if err != nil {
			return nil, err
		}
		files, err := readLexiconFiles(ctx, repository, source.paths)
		if err != nil {
			return nil, err
		}
		result = append(result, legalquerycandidateeval.LexiconArtifact{
			LexiconID:       source.lexiconID,
			LexiconVersion:  version,
			Files:           files,
			AggregateSHA256: legalquerycandidateeval.LexiconAggregateSHA256(files),
		})
	}
	return result, nil
}

func readLexiconFiles(
	ctx context.Context,
	repository *prepareRepository,
	paths []string,
) ([]legalquerycandidateeval.FileDigest, error) {
	files := make([]legalquerycandidateeval.FileDigest, 0, len(paths))
	for _, path := range paths {
		if err := checkPrepareContext(ctx); err != nil {
			return nil, err
		}
		raw, err := repository.Read(path, 8<<20)
		if err != nil {
			return nil, err
		}
		files = append(files, legalquerycandidateeval.FileDigest{
			Path: path, RawSHA256: legalquerycandidateeval.RawSHA256(raw),
		})
	}
	return files, nil
}

func sharedLexiconVersion(
	candidate legalquerycandidateprofile.Set,
	lawNames bool,
) (string, error) {
	metadata := candidate.ProfileMetadata()
	if len(metadata) == 0 {
		return "", fmt.Errorf("候補 profile metadata がありません")
	}
	version := metadata[0].LegalConceptLexiconVersion()
	if lawNames {
		version = metadata[0].LawNameLexiconVersion()
	}
	for _, profile := range metadata[1:] {
		current := profile.LegalConceptLexiconVersion()
		if lawNames {
			current = profile.LawNameLexiconVersion()
		}
		if current != version {
			return "", fmt.Errorf("候補 profile が異なる辞書版を参照しています")
		}
	}
	return version, nil
}
