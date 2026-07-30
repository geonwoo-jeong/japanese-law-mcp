package core

import (
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestProfileは列挙した法令名と条文検索を一候補にする(
	t *testing.T,
) {
	t.Parallel()

	ref := mustCoreCompositionJudicialRef(t)
	generation := generateQuery(
		t,
		"この裁判例参照を読み、成年後見の法令名・条文・裁判例も検索してください。",
		&ref,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("SOT-ARCH-027: candidates = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 2 ||
		steps[0].InputKind() != legalquery.InputKindLawSearch ||
		steps[1].InputKind() != legalquery.InputKindLawContentSearch {
		t.Fatalf("SOT-ARCH-027: core steps = %#v", steps)
	}
	law, lawOK := steps[0].LogicalInput().(legalquery.LawSearchIntentV1)
	content, contentOK := steps[1].LogicalInput().(legalquery.LawContentSearchIntentV1)
	if !lawOK || law.Query() != "成年後見" ||
		!contentOK ||
		!slices.Equal(content.AllTerms(), []string{"成年後見"}) ||
		len(content.AnyTerms()) != 0 ||
		len(content.ExcludeTerms()) != 0 {
		t.Fatalf(
			"SOT-MODEL-022: logical inputs = law:%#v content:%#v",
			steps[0].LogicalInput(),
			steps[1].LogicalInput(),
		)
	}
	members := generation.CompositionMembers()
	if len(members) != 1 ||
		members[0].CandidateID() != candidates[0].CandidateID() ||
		len(members[0].StepOrigins()) != 2 {
		t.Fatalf("SOT-MODEL-028: composition members = %#v", members)
	}
}

func TestProfileは代替または除外したResourceを同時検索にしない(
	t *testing.T,
) {
	t.Parallel()

	queries := []string{
		"成年後見を法令名または条文で検索してください。",
		"成年後見を法令名以外の条文で検索してください。",
		"成年後見を法令名・条文の一つの方法で検索してください。",
	}
	for _, query := range queries {
		generation := generateQuery(t, query, nil)
		for _, candidate := range generation.Candidates() {
			hasLaw := false
			hasContent := false
			for _, step := range candidate.Steps() {
				hasLaw = hasLaw ||
					step.InputKind() == legalquery.InputKindLawSearch
				hasContent = hasContent ||
					step.InputKind() == legalquery.InputKindLawContentSearch
			}
			if hasLaw && hasContent {
				t.Fatalf(
					"SOT-MODEL-026: 代替または除外 Resource を同時検索にしました: query=%q candidate=%#v",
					query,
					candidate,
				)
			}
		}
	}
}
