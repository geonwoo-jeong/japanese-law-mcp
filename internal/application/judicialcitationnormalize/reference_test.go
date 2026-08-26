package judicialcitationnormalize_test

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitationnormalize"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestExactLawAliasResolverは正式名称と登録別名だけを解決する(t *testing.T) {
	t.Parallel()

	resolver := mustResolver(t)
	for _, input := range []string{"民法第1条", "個情法第1条", "第五種共同漁業権法第1条"} {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			got, reason, err := judicialcitationnormalize.NormalizeReferencedProvision(
				context.Background(),
				input,
				resolver,
			)
			if err != nil || reason != "" || got.LawID() == "" {
				t.Fatalf("正確一致を解決できません: %#v %q %v", got, reason, err)
			}
		})
	}
}

func TestNormalizeReferencedProvisionは法令名と条文位置を正規化する(t *testing.T) {
	t.Parallel()

	resolver := mustResolver(t)
	tests := []struct {
		input        string
		provision    model.LawArticleProvision
		article      string
		paragraph    int
		hasParagraph bool
	}{
		{input: "個情法附則第２条第１項", provision: model.LawArticleProvisionSupplementary, article: "2", paragraph: 1, hasParagraph: true},
		{input: "民法第10条の2", provision: model.LawArticleProvisionMain, article: "10_2"},
		{input: "民法第百九条第二項", provision: model.LawArticleProvisionMain, article: "109", paragraph: 2, hasParagraph: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()

			got, reason, err := judicialcitationnormalize.NormalizeReferencedProvision(
				context.Background(),
				test.input,
				resolver,
			)
			if err != nil || reason != "" {
				t.Fatalf("NormalizeReferencedProvision() = (%#v, %q, %v)", got, reason, err)
			}
			if got.LawID() == "" || got.LawTitle() == "" {
				t.Fatalf("法令解決結果 = %#v", got)
			}
			location := got.Location()
			paragraph, exists := location.ParagraphNumber()
			if location.Provision() != test.provision ||
				location.ArticleNumber() != test.article ||
				exists != test.hasParagraph ||
				paragraph != test.paragraph {
				t.Fatalf("location = %#v", location)
			}
		})
	}
}

func TestNormalizeReferencedProvisionは未解決理由を区別する(t *testing.T) {
	t.Parallel()

	resolver := mustResolver(t)
	tests := []struct {
		input  string
		reason model.JudicialCitationUnresolvedReason
	}{
		{input: "同法709条", reason: model.JudicialCitationUnresolvedReasonUnsupportedReference},
		{input: "開示法第1条", reason: model.JudicialCitationUnresolvedReasonAmbiguousTarget},
		{input: "個人情報保護砲第1条", reason: model.JudicialCitationUnresolvedReasonFuzzyMatchOnly},
		{input: "個人 情報保護法第1条", reason: model.JudicialCitationUnresolvedReasonFuzzyMatchOnly},
		{input: "未登録架空特別法名第3条", reason: model.JudicialCitationUnresolvedReasonUnregisteredLawName},
		{input: "民法施行法第1条", reason: model.JudicialCitationUnresolvedReasonUnregisteredLawName},
		{input: "民法第1条第1項第1号", reason: model.JudicialCitationUnresolvedReasonAmbiguousLawLocation},
		{input: "民法第0条", reason: model.JudicialCitationUnresolvedReasonAmbiguousLawLocation},
	}
	for _, test := range tests {
		test := test
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()

			got, reason, err := judicialcitationnormalize.NormalizeReferencedProvision(
				context.Background(),
				test.input,
				resolver,
			)
			if err != nil || reason != test.reason || got.LawID() != "" {
				t.Fatalf("入力 %q = (%#v, %q, %v)", test.input, got, reason, err)
			}
		})
	}
}

func TestNormalizeReferencedProvisionは取消と未初期化resolverを拒否する(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := judicialcitationnormalize.NormalizeReferencedProvision(ctx, "民法第1条", mustResolver(t)); err == nil {
		t.Fatal("取消済み context を受理しました")
	}
	if _, _, err := judicialcitationnormalize.NormalizeReferencedProvision(
		context.Background(),
		"民法第1条",
		judicialcitationnormalize.ExactLawAliasResolver{},
	); err == nil {
		t.Fatal("未初期化 resolver を受理しました")
	}
}

func TestSplitReferencedProvisionTextは区切りを決定的に分割する(t *testing.T) {
	t.Parallel()

	got := judicialcitationnormalize.SplitReferencedProvisionText("民法709条、個情法第2条\n刑法199条")
	if len(got) != 3 || got[0] != "民法709条" || got[1] != "個情法第2条" || got[2] != "刑法199条" {
		t.Fatalf("SplitReferencedProvisionText() = %#v", got)
	}
}

func TestNormalizeLowerCourtDecisionは完全な原審事件番号だけを受理する(t *testing.T) {
	t.Parallel()

	got, ok, err := judicialcitationnormalize.NormalizeLowerCourtDecision(
		"東京高等裁判所",
		"令和7年(ネ)第12号",
	)
	if err != nil || !ok || got.CourtName() != "東京高等裁判所" || got.CaseNumberSearch() != "令和7年(ネ)第12号" {
		t.Fatalf("NormalizeLowerCourtDecision() = (%#v, %t, %v)", got, ok, err)
	}
	if _, ok, err := judicialcitationnormalize.NormalizeLowerCourtDecision("東京高等裁判所", "令和7年(ネ)第12号 補足"); err == nil || ok {
		t.Fatalf("後続文字つき事件番号を受理しました: ok=%t err=%v", ok, err)
	}
}

func mustResolver(t *testing.T) judicialcitationnormalize.ExactLawAliasResolver {
	t.Helper()

	resolver, err := judicialcitationnormalize.NewExactLawAliasResolver([]lawnamelexicon.Entry{
		{
			ResourceID: "civil",
			Canonical:  "民法",
		},
		{
			ResourceID: "appi",
			Canonical:  "個人情報の保護に関する法律",
			Terms:      []string{"個人情報保護法", "個情法"},
		},
		{
			ResourceID: "disclosure-a",
			Canonical:  "独立行政法人等の保有する情報の公開に関する法律",
			Terms:      []string{"開示法"},
		},
		{
			ResourceID: "disclosure-b",
			Canonical:  "行政機関の保有する情報の公開に関する法律",
			Terms:      []string{"開示法"},
		},
		{
			ResourceID: "penal",
			Canonical:  "刑法",
		},
		{
			ResourceID: "fifth-fishery",
			Canonical:  "第五種共同漁業権法",
		},
	})
	if err != nil {
		t.Fatalf("resolver を構築できません: %v", err)
	}
	return resolver
}
