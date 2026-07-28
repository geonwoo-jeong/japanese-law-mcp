package legalquery

import (
	"errors"
	"testing"
	"time"
)

func TestNewServiceは未初期化の依存を拒否する(t *testing.T) {
	t.Parallel()

	newDependencies := func(
		t *testing.T,
	) (
		*servicePreprocessorFake,
		QueryProfileSet,
		PackState,
		Executor,
	) {
		t.Helper()
		executor, _, _ := mustServiceExecutor(t)
		return &servicePreprocessorFake{},
			mustServiceSingleProfileSet(t),
			mustSelectorTestPackState(t, nil, nil),
			executor
	}

	t.Run("preprocessorなし", func(t *testing.T) {
		t.Parallel()
		_, profiles, packState, executor := newDependencies(t)
		if _, err := NewService(
			nil,
			profiles,
			packState,
			executor,
			time.Second,
		); err == nil {
			t.Fatal("SOT-ARCH-022: preprocessor なしを受理しました")
		}
	})

	t.Run("preprocessorのtyped nil", func(t *testing.T) {
		t.Parallel()
		_, profiles, packState, executor := newDependencies(t)
		var preprocessor *servicePreprocessorFake
		if _, err := NewService(
			preprocessor,
			profiles,
			packState,
			executor,
			time.Second,
		); err == nil {
			t.Fatal("SOT-ARCH-022: preprocessor の typed nil を受理しました")
		}
	})

	t.Run("profile setのzero value", func(t *testing.T) {
		t.Parallel()
		preprocessor, _, packState, executor := newDependencies(t)
		if _, err := NewService(
			preprocessor,
			QueryProfileSet{},
			packState,
			executor,
			time.Second,
		); err == nil {
			t.Fatal("SOT-ARCH-022: profile set の zero value を受理しました")
		}
	})

	t.Run("pack stateなし", func(t *testing.T) {
		t.Parallel()
		preprocessor, profiles, _, executor := newDependencies(t)
		if _, err := NewService(
			preprocessor,
			profiles,
			nil,
			executor,
			time.Second,
		); err == nil {
			t.Fatal("SOT-ARCH-022: pack state なしを受理しました")
		}
	})

	t.Run("pack stateのtyped nil", func(t *testing.T) {
		t.Parallel()
		preprocessor, profiles, _, executor := newDependencies(t)
		var packState *servicePackStateFake
		if _, err := NewService(
			preprocessor,
			profiles,
			packState,
			executor,
			time.Second,
		); err == nil {
			t.Fatal("SOT-ARCH-022: pack state の typed nil を受理しました")
		}
	})

	t.Run("executorのzero value", func(t *testing.T) {
		t.Parallel()
		preprocessor, profiles, packState, _ := newDependencies(t)
		if _, err := NewService(
			preprocessor,
			profiles,
			packState,
			Executor{},
			time.Second,
		); err == nil {
			t.Fatal("SOT-ARCH-022: Executor の zero value を受理しました")
		}
	})

	for name, timeout := range map[string]time.Duration{
		"zero timeout":     0,
		"negative timeout": -time.Second,
	} {
		name, timeout := name, timeout
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			preprocessor, profiles, packState, executor := newDependencies(t)
			if _, err := NewService(
				preprocessor,
				profiles,
				packState,
				executor,
				timeout,
			); err == nil {
				t.Fatal("SOT-SCN-009: 正でない request timeout を受理しました")
			}
		})
	}
}

func TestServiceValidateは構築境界を再検証する(t *testing.T) {
	t.Parallel()

	if err := (*Service)(nil).Validate(); err == nil {
		t.Fatal("nil Service を受理しました")
	}
	if err := (&Service{}).Validate(); err == nil {
		t.Fatal("zero-value Service を受理しました")
	}

	preprocessor := &servicePreprocessorFake{}
	profiles := mustServiceSingleProfileSet(t)
	packState := mustSelectorTestPackState(t, nil, nil)
	executor, _, core := mustServiceExecutor(t)
	service := mustService(
		t,
		preprocessor,
		profiles,
		packState,
		executor,
		time.Second,
	)
	if err := service.Validate(); err != nil {
		t.Fatalf("構築済み Service が有効ではありません: %v", err)
	}

	cause := errors.New("試験用 facade 構成エラー")
	core.validateErr = cause
	if err := service.Validate(); !errors.Is(err, cause) {
		t.Fatalf("依存の検証エラーを保持しませんでした: %v", err)
	}
}

func TestQueryProfileSetValidateは構築とmetadata固定を確認する(
	t *testing.T,
) {
	t.Parallel()

	if err := (QueryProfileSet{}).Validate(); err == nil {
		t.Fatal("profile set の zero value を受理しました")
	}
	profile := &selectorTestProfile{
		metadata: mustSelectorTestMetadata(
			t,
			"core",
			"core-profile-v1",
			selectorTestRankingVersion,
		),
		selectionMode: QuerySelectionModeAutomatic,
	}
	profiles, err := NewQueryProfileSet([]QueryProfile{profile})
	if err != nil {
		t.Fatalf("profile set を作成できません: %v", err)
	}
	if err := profiles.Validate(); err != nil {
		t.Fatalf("構築済み profile set が有効ではありません: %v", err)
	}
	profile.metadata = mustSelectorTestMetadata(
		t,
		"core",
		"core-profile-v2",
		selectorTestRankingVersion,
	)
	if err := profiles.Validate(); err == nil {
		t.Fatal("構築後に metadata が変わった profile set を受理しました")
	}
}

func TestQueryProfileSetCollectは候補生成中のmetadata変更を拒否する(
	t *testing.T,
) {
	t.Parallel()

	base := mustSelectorTestMetadata(
		t,
		"core",
		"core-profile-v1",
		selectorTestRankingVersion,
	)
	changed := mustSelectorTestMetadata(
		t,
		"core",
		"core-profile-v2",
		selectorTestRankingVersion,
	)
	profile := &selectorTestProfile{
		metadata:       base,
		selectionMode:  QuerySelectionModeAutomatic,
		profileVersion: base.ProfileVersion(),
		rankingVersion: base.RankingVersion(),
	}
	profile.generate = func(
		_ CandidateIDScope,
	) (QueryProfileContribution, error) {
		generation, err := NewCandidateGeneration(
			QueryProfileContributionValues{
				ProfileID:      base.ProfileID(),
				ProfileVersion: base.ProfileVersion(),
				RankingVersion: base.RankingVersion(),
				SelectionMode:  QuerySelectionModeAutomatic,
			},
		)
		profile.metadata = changed
		return generation, err
	}
	profiles, err := NewQueryProfileSet([]QueryProfile{profile})
	if err != nil {
		t.Fatalf("profile set を作成できません: %v", err)
	}
	if _, err := profiles.Collect(
		mustSelectorTestPreprocessResult(t),
	); err == nil {
		t.Fatal("候補生成中に metadata が変わった profile を受理しました")
	}
}
