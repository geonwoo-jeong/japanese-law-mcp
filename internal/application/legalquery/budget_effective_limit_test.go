package legalquery

import "testing"

func TestEffectiveLimitForStepsは固定予算式を一元的に公開する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		limitPerAttempt int
		readSteps       int
		collectionSteps int
		wantLimit       int
		wantLimitExists bool
	}{
		{
			name:            "collectionなし",
			limitPerAttempt: DefaultLimitPerAttempt,
			readSteps:       MaxCapabilityCalls,
		},
		{
			name:            "request上限",
			limitPerAttempt: 7,
			collectionSteps: 1,
			wantLimit:       7,
			wantLimitExists: true,
		},
		{
			name:            "step上限",
			limitPerAttempt: MaxLimitPerAttempt,
			collectionSteps: 1,
			wantLimit:       MaxItemsPerCollectionStep,
			wantLimitExists: true,
		},
		{
			name:            "全体上限",
			limitPerAttempt: MaxLimitPerAttempt,
			readSteps:       1,
			collectionSteps: 3,
			wantLimit:       13,
			wantLimitExists: true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, exists, err := EffectiveLimitForSteps(
				test.limitPerAttempt,
				test.readSteps,
				test.collectionSteps,
			)
			if err != nil {
				t.Fatalf("SOT-MODEL-023: EffectiveLimitForSteps() error = %v", err)
			}
			if got != test.wantLimit || exists != test.wantLimitExists {
				t.Fatalf(
					"SOT-MODEL-023: EffectiveLimitForSteps() = %d, %t; want %d, %t",
					got,
					exists,
					test.wantLimit,
					test.wantLimitExists,
				)
			}
		})
	}
}

func TestEffectiveLimitForStepsは不正な予算入力を拒否する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		limitPerAttempt int
		readSteps       int
		collectionSteps int
	}{
		{name: "limit下限", limitPerAttempt: 0, collectionSteps: 1},
		{
			name:            "limit上限",
			limitPerAttempt: MaxLimitPerAttempt + 1,
			collectionSteps: 1,
		},
		{
			name:            "read負数",
			limitPerAttempt: DefaultLimitPerAttempt,
			readSteps:       -1,
			collectionSteps: 1,
		},
		{
			name:            "collection負数",
			limitPerAttempt: DefaultLimitPerAttempt,
			collectionSteps: -1,
		},
		{
			name:            "step総数超過",
			limitPerAttempt: DefaultLimitPerAttempt,
			readSteps:       MaxCapabilityCalls,
			collectionSteps: 1,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if _, _, err := EffectiveLimitForSteps(
				test.limitPerAttempt,
				test.readSteps,
				test.collectionSteps,
			); err == nil {
				t.Fatal("SOT-MODEL-023: 不正な固定予算入力を受理した")
			}
		})
	}
}
