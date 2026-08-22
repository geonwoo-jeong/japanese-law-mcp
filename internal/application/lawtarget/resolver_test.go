package lawtarget_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawtarget"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func TestPreprocessResolverは位置付き法令名一件だけを解決する(t *testing.T) {
	t.Parallel()

	resolver := newEmbeddedResolver(t)

	tests := []struct {
		name      string
		query     string
		lawID     string
		title     string
		matchKind lawtarget.MatchKind
	}{
		{
			name:      "正式名称",
			query:     "民法",
			lawID:     "129AC0000000089",
			title:     "民法",
			matchKind: lawtarget.MatchKindExact,
		},
		{
			name:      "自然文の登録略称",
			query:     "個情法について教えてください",
			lawID:     "415AC0000000057",
			title:     "個人情報の保護に関する法律",
			matchKind: lawtarget.MatchKindRegisteredTerm,
		},
		{
			name:      "自然文の一意な誤記",
			query:     "著作券法について教えてください",
			lawID:     "345AC0000000048",
			title:     "著作権法",
			matchKind: lawtarget.MatchKindUniqueTypoCorrection,
		},
		{
			name:      "検索語全体の挿入",
			query:     "著者作権法",
			lawID:     "345AC0000000048",
			title:     "著作権法",
			matchKind: lawtarget.MatchKindUniqueTypoCorrection,
		},
		{
			name:      "検索語全体の削除",
			query:     "著作法",
			lawID:     "345AC0000000048",
			title:     "著作権法",
			matchKind: lawtarget.MatchKindUniqueTypoCorrection,
		},
		{
			name:      "検索語全体の置換",
			query:     "著作券法",
			lawID:     "345AC0000000048",
			title:     "著作権法",
			matchKind: lawtarget.MatchKindUniqueTypoCorrection,
		},
		{
			name:      "検索語全体の隣接転置",
			query:     "著権作法",
			lawID:     "345AC0000000048",
			title:     "著作権法",
			matchKind: lawtarget.MatchKindUniqueTypoCorrection,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			target, resolved, resolveErr := resolver.Resolve(
				context.Background(),
				test.query,
			)
			if resolveErr != nil {
				t.Fatalf("SOT-ARCH-030: Resolve() のエラー = %v", resolveErr)
			}
			if !resolved ||
				target.LawID() != test.lawID ||
				target.OfficialTitle() != test.title ||
				target.MatchKind() != test.matchKind {
				t.Fatalf(
					"SOT-ARCH-030: resolved=%t lawId=%q title=%q matchKind=%q",
					resolved,
					target.LawID(),
					target.OfficialTitle(),
					target.MatchKind(),
				)
			}
		})
	}
}

func TestPreprocessResolverは複数Spanと曖昧な候補を解決しない(t *testing.T) {
	t.Parallel()

	resolver := newEmbeddedResolver(t)

	for _, query := range []string{
		"民法と民法",
		"民法と商法",
		"開示法",
		"法",
	} {
		query := query
		t.Run(query, func(t *testing.T) {
			t.Parallel()

			target, resolved, resolveErr := resolver.Resolve(
				context.Background(),
				query,
			)
			if resolveErr != nil {
				t.Fatalf("SOT-ARCH-030: Resolve() のエラー = %v", resolveErr)
			}
			if resolved || target != (lawtarget.ResolvedLawTarget{}) {
				t.Fatalf("SOT-ARCH-030: 曖昧な入力を解決しました: %t", resolved)
			}
		})
	}
}

func TestPreprocessResolverは不正依存と取消を拒否する(t *testing.T) {
	t.Parallel()

	if _, err := lawtarget.NewPreprocessResolver(nil, nil); err == nil {
		t.Fatal("SOT-ARCH-030: nil 前処理器を受理しました")
	}
	resolver := newEmbeddedResolver(t)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := resolver.Resolve(cancelled, "民法"); !errors.Is(err, context.Canceled) {
		t.Fatalf("SOT-ARCH-030: 取消 error = %v", err)
	}
}

func TestPrioritizeは対象内外の順序と入力を保持する(t *testing.T) {
	t.Parallel()

	target, err := lawtarget.NewResolvedLawTarget(
		"law-target",
		"対象法",
		lawtarget.MatchKindRegisteredTerm,
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-030: target を構築できません: %v", err)
	}
	input := []string{"other-1", "law-target", "other-2", "law-target", "other-3"}
	original := append([]string(nil), input...)

	got, changed, err := lawtarget.Prioritize(
		input,
		target,
		func(value string) string { return value },
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-030: stable partition のエラー = %v", err)
	}
	want := []string{"law-target", "law-target", "other-1", "other-2", "other-3"}
	if !changed || !reflect.DeepEqual(got, want) {
		t.Fatalf("SOT-ARCH-030: prioritized = %#v, changed=%t", got, changed)
	}
	if !reflect.DeepEqual(input, original) {
		t.Fatalf("SOT-ARCH-030: 入力 slice を変更しました: %#v", input)
	}
}

func newEmbeddedResolver(t *testing.T) lawtarget.PreprocessResolver {
	t.Helper()
	preprocessor, err := querypreprocess.NewEmbedded(nil)
	if err != nil {
		t.Fatalf("SOT-ARCH-030: 共通前処理器を構築できません: %v", err)
	}
	lexicon, err := lawnamelexicon.LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ENG-022: 法令名辞書を読み込めません: %v", err)
	}
	resolver, err := lawtarget.NewPreprocessResolver(preprocessor, lexicon.Entries())
	if err != nil {
		t.Fatalf("SOT-ARCH-030: law-target resolver を構築できません: %v", err)
	}
	return resolver
}
