package core

import (
	"bytes"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
)

func TestLoadEmbeddedは法令コア五能力と辞書版を固定する(t *testing.T) {
	t.Parallel()

	profile, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ENG-025: LoadEmbedded() のエラー = %v", err)
	}
	metadata := profile.Metadata()
	if metadata.ProfileID() != "core" ||
		metadata.ProfileVersion() != "core-2026-07-29-1" ||
		metadata.RankingVersion() != "legal-query-ranking-2026-07-28-1" ||
		metadata.CueSetVersion() != "core-cues-2026-07-28-3" {
		t.Fatalf("metadata = %#v", metadata)
	}
	const lawVersion = "e-gov-law-api-v2-laws-2026-07-27+ndl-common-abbreviations-2026-07-27"
	if metadata.LawNameLexiconVersion() != lawVersion ||
		metadata.LegalConceptLexiconVersion() != "legal-concept-2026-07-28-2" {
		t.Fatalf(
			"lexicon versions = %q, %q",
			metadata.LawNameLexiconVersion(),
			metadata.LegalConceptLexiconVersion(),
		)
	}
	targets := metadata.Targets()
	wantKinds := []legalquery.LogicalInputKind{
		legalquery.InputKindLawSearch,
		legalquery.InputKindLawContentSearch,
		legalquery.InputKindLawRead,
		legalquery.InputKindLawArticleRead,
		legalquery.InputKindLawUpdates,
	}
	if len(targets) != len(wantKinds) {
		t.Fatalf("targets = %#v", targets)
	}
	for index, target := range targets {
		if target.InputKind() != wantKinds[index] {
			t.Fatalf("targets[%d] = %#v", index, target)
		}
	}
	if len(profile.CueVocabulary()) == 0 {
		t.Fatal("cue vocabulary が空です")
	}

	var _ legalquery.QueryProfile = profile
}

func TestProfileGetterはcueとmetadataを変更させない(t *testing.T) {
	t.Parallel()

	profile, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() のエラー = %v", err)
	}
	cues := profile.CueVocabulary()
	cues[0].ProfileID = "changed"
	cues[0].Terms[0] = "changed"
	targets := profile.Metadata().Targets()
	targets[0] = legalquery.QueryProfileTarget{}

	nextCues := profile.CueVocabulary()
	if nextCues[0].ProfileID != "core" ||
		nextCues[0].Terms[0] == "changed" ||
		profile.Metadata().Targets()[0].InputKind() != legalquery.InputKindLawSearch {
		t.Fatal("SOT-ENG-025: profile getter から内部状態を変更できました")
	}
}

func TestLoadは未知項目trailing値辞書版不一致を拒否する(t *testing.T) {
	t.Parallel()

	lawNames, concepts := mustEmbeddedLexicons(t)
	tests := map[string]struct {
		profile []byte
		cues    []byte
	}{
		"profile 未知項目": {
			profile: bytes.Replace(
				embeddedProfile,
				[]byte(`"schemaVersion": 1,`),
				[]byte(`"schemaVersion": 1, "unknown": true,`),
				1,
			),
			cues: embeddedCues,
		},
		"profile trailing": {
			profile: append(append([]byte(nil), embeddedProfile...), []byte(`{}`)...),
			cues:    embeddedCues,
		},
		"cue 未知項目": {
			profile: embeddedProfile,
			cues: bytes.Replace(
				embeddedCues,
				[]byte(`"schemaVersion": 1,`),
				[]byte(`"schemaVersion": 1, "unknown": true,`),
				1,
			),
		},
		"cue 版不一致": {
			profile: embeddedProfile,
			cues: bytes.Replace(
				embeddedCues,
				[]byte(`core-cues-2026-07-28-3`),
				[]byte(`core-cues-2026-07-28-9`),
				1,
			),
		},
		"法概念辞書版不一致": {
			profile: bytes.Replace(
				embeddedProfile,
				[]byte(`legal-concept-2026-07-28-2`),
				[]byte(`legal-concept-2026-07-28-9`),
				1,
			),
			cues: embeddedCues,
		},
	}
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load(test.profile, test.cues, lawNames, concepts); err == nil {
				t.Fatal("不正な profile data を受理しました")
			}
		})
	}
}

func TestLoadは辞書依存の欠落を起動時に拒否する(t *testing.T) {
	t.Parallel()

	lawNames, concepts := mustEmbeddedLexicons(t)
	if _, err := Load(embeddedProfile, embeddedCues, nil, concepts); err == nil {
		t.Fatal("nil law name lexicon を受理しました")
	}
	if _, err := Load(embeddedProfile, embeddedCues, lawNames, nil); err == nil {
		t.Fatal("nil legal concept lexicon を受理しました")
	}
}

func mustEmbeddedLexicons(
	t *testing.T,
) (*lawnamelexicon.Lexicon, *legalconceptlexicon.Lexicon) {
	t.Helper()
	lawNames, err := lawnamelexicon.LoadEmbedded()
	if err != nil {
		t.Fatalf("法令名辞書を読み込めません: %v", err)
	}
	concepts, err := legalconceptlexicon.LoadEmbedded()
	if err != nil {
		t.Fatalf("法概念辞書を読み込めません: %v", err)
	}
	return lawNames, concepts
}
