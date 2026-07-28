package legalquery

import (
	"slices"
	"strings"
	"testing"
)

type selectorTypedNilPackState struct {
	enabled bool
	adopted bool
}

func (s *selectorTypedNilPackState) State(string) (bool, bool) {
	return s.enabled, s.adopted
}

type selectorTypedNilQueryProfile struct {
	metadata QueryProfileMetadata
}

func (p *selectorTypedNilQueryProfile) Metadata() QueryProfileMetadata {
	return p.metadata
}

func (*selectorTypedNilQueryProfile) CueVocabulary() []CueVocabularyEntry {
	return nil
}

func (*selectorTypedNilQueryProfile) Generate(
	CandidateGenerationInput,
	CandidateIDScope,
) (CandidateGeneration, error) {
	return CandidateGeneration{}, nil
}

func TestSelectorはtypedNilのPackStateをfailClosedで拒否する(
	t *testing.T,
) {
	t.Parallel()

	var state *selectorTypedNilPackState
	_, err := SelectLegalQueryPlan(SelectorInput{
		ProfileSetResult: mustSelectorTestProfileSetResult(
			t,
			nil,
			nil,
			QuerySelectionModeAutomatic,
			nil,
		),
		PackState:       state,
		LimitPerAttempt: DefaultLimitPerAttempt,
	})
	if err == nil || !strings.Contains(err.Error(), "packState") {
		t.Fatalf("typed nil の packState を fail-closed で拒否しませんでした: %v", err)
	}
}

func TestQueryProfileSetはtypedNilのProfileをfailClosedで拒否する(
	t *testing.T,
) {
	t.Parallel()

	var profile *selectorTypedNilQueryProfile
	_, err := NewQueryProfileSet([]QueryProfile{profile})
	if err == nil || !strings.Contains(err.Error(), "profiles[0]") {
		t.Fatalf("typed nil の profile を fail-closed で拒否しませんでした: %v", err)
	}
}

func TestCollectProfileCandidatesはtypedNilのProfileをfailClosedで拒否する(
	t *testing.T,
) {
	t.Parallel()

	var profile *selectorTypedNilQueryProfile
	_, err := CollectProfileCandidates(
		profile,
		PreprocessResult{},
		CandidateIDScope{},
	)
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("typed nil の profile を fail-closed で拒否しませんでした: %v", err)
	}
}

func TestSelectorは候補のない対象外Taskに実態と一致する理由を返す(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name    string
		signals []CandidateGenerationSignal
	}{
		{
			name: "法的助言だけ",
			signals: []CandidateGenerationSignal{
				CandidateSignalUnsupportedLegalAdvice,
			},
		},
		{
			name: "翻訳だけ",
			signals: []CandidateGenerationSignal{
				CandidateSignalUnsupportedTranslation,
			},
		},
		{
			name: "法的助言と翻訳だけ",
			signals: []CandidateGenerationSignal{
				CandidateSignalUnsupportedLegalAdvice,
				CandidateSignalUnsupportedTranslation,
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			plan := mustSelectTestPlan(
				t,
				mustSelectorTestProfileSetResult(
					t,
					nil,
					test.signals,
					QuerySelectionModeAutomatic,
					nil,
				),
				mustSelectorTestPackState(t, nil, nil),
				DefaultLimitPerAttempt,
			)
			if plan.Decision() != PlanDecisionUnsupported ||
				!slices.Equal(
					plan.ReasonCodes(),
					[]ReasonCode{ReasonCodeUnsupportedTaskOrResource},
				) ||
				len(plan.RankedCandidates()) != 0 ||
				len(plan.Selected()) != 0 ||
				len(plan.Budget().StepBudgets()) != 0 {
				t.Fatalf(
					"SOT-MODEL-023: candidate 0 の対象外 task plan = decision:%q reasons:%#v ranked:%d selected:%d budgets:%d",
					plan.Decision(),
					plan.ReasonCodes(),
					len(plan.RankedCandidates()),
					len(plan.Selected()),
					len(plan.Budget().StepBudgets()),
				)
			}
		})
	}
}
