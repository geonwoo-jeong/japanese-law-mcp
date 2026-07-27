package lawnamelexicon

import (
	"strings"
	"testing"
)

const validOfficialFixture = `{
  "schemaVersion": 1,
  "datasetId": "test-official",
  "source": {
    "providerId": "e-gov-law-api-v2",
    "operation": "GET /laws",
    "url": "https://laws.e-gov.go.jp/api/2/laws",
    "retrievedAt": "2026-07-27T09:00:00Z"
  },
  "statistics": {"lawCount": 2, "aliasCount": 2},
  "entries": [
    {
      "lawId": "415AC0000000057",
      "revisionId": "415AC0000000057_20250601_504AC0000000068",
      "lawNumber": "平成十五年法律第五十七号",
      "title": "個人情報の保護に関する法律",
      "titleKana": "こじんじょうほうのほごにかんするほうりつ",
      "aliases": ["個人情報保護法"]
    },
    {
      "lawId": "425AC0000000027",
      "revisionId": "425AC0000000027_20250601_504AC0000000068",
      "lawNumber": "平成二十五年法律第二十七号",
      "title": "行政手続における特定の個人を識別するための番号の利用等に関する法律",
      "titleKana": "ぎょうせいてつづきにおけるとくていのこじんをしきべつするためのばんごうのりようとうにかんするほうりつ",
      "aliases": ["マイナンバー法"]
    }
  ]
}`

const validSupplementalFixture = `{
  "schemaVersion": 1,
  "datasetId": "test-supplemental",
  "entries": [
    {
      "lawId": "415AC0000000057",
      "lawNumber": "平成十五年法律第五十七号",
      "title": "個人情報の保護に関する法律",
      "alias": "個情法",
      "kind": "common-abbreviation",
      "sourceName": "国立国会図書館 日本法令索引",
      "sourceUrl": "https://hourei.ndl.go.jp/simple/detail?lawId=0000095240",
      "confirmedAt": "2026-07-27"
    }
  ]
}`

func TestLoadBuildsImmutableEntriesFromOfficialAndSupplementalData(
	t *testing.T,
) {
	t.Parallel()

	lexicon, err := Load(
		[]byte(validOfficialFixture),
		[]byte(validSupplementalFixture),
	)
	if err != nil {
		t.Fatalf("SOT-ENG-022: Load() のエラー = %v", err)
	}
	entries := lexicon.Entries()
	if len(entries) != 2 {
		t.Fatalf("SOT-ENG-022: entries = %d", len(entries))
	}
	if entries[0].ResourceID != "415AC0000000057" ||
		entries[0].Canonical != "個人情報の保護に関する法律" ||
		!containsString(entries[0].Terms, "個人情報保護法") ||
		!containsString(entries[0].Terms, "個情法") {
		t.Fatalf("SOT-ENG-022: entry = %#v", entries[0])
	}

	entries[0].Canonical = "変更後"
	entries[0].Terms[0] = "変更後"
	reloaded := lexicon.Entries()
	if reloaded[0].Canonical != "個人情報の保護に関する法律" ||
		containsString(reloaded[0].Terms, "変更後") {
		t.Fatalf("SOT-ENG-022: 辞書が変更されました: %#v", reloaded[0])
	}
	terms := lexicon.Terms()
	if !containsString(terms, "個情法") {
		t.Fatalf("SOT-ENG-022: terms = %#v", terms)
	}
	terms[0] = "変更後"
	if containsString(lexicon.Terms(), "変更後") {
		t.Fatal("SOT-ENG-022: 登録語一覧が変更されました")
	}

	var nilLexicon *Lexicon
	if nilLexicon.Entries() != nil || nilLexicon.Terms() != nil {
		t.Fatal("nil Lexicon が値を返しました")
	}
}

func TestLoadRejectsMalformedOrInconsistentData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		official     string
		supplemental string
	}{
		{
			name: "未知のschema",
			official: strings.Replace(
				validOfficialFixture,
				`"schemaVersion": 1`,
				`"schemaVersion": 2`,
				1,
			),
			supplemental: validSupplementalFixture,
		},
		{
			name:         "後続JSON",
			official:     validOfficialFixture + `{}`,
			supplemental: validSupplementalFixture,
		},
		{
			name: "UTC以外の取得日時",
			official: strings.Replace(
				validOfficialFixture,
				`2026-07-27T09:00:00Z`,
				`2026-07-27T18:00:00+09:00`,
				1,
			),
			supplemental: validSupplementalFixture,
		},
		{
			name: "aliasの重複",
			official: strings.Replace(
				validOfficialFixture,
				`"aliases": ["個人情報保護法"]`,
				`"aliases": ["個人情報保護法", "個人情報保護法"]`,
				1,
			),
			supplemental: validSupplementalFixture,
		},
		{
			name: "未知の項目",
			official: strings.Replace(
				validOfficialFixture,
				`"datasetId": "test-official"`,
				`"datasetId": "test-official", "unknown": true`,
				1,
			),
			supplemental: validSupplementalFixture,
		},
		{
			name: "統計不一致",
			official: strings.Replace(
				validOfficialFixture,
				`"lawCount": 2`,
				`"lawCount": 3`,
				1,
			),
			supplemental: validSupplementalFixture,
		},
		{
			name:     "補足の未知法令",
			official: validOfficialFixture,
			supplemental: strings.Replace(
				validSupplementalFixture,
				`"415AC0000000057"`,
				`"999AC0000009999"`,
				1,
			),
		},
		{
			name:     "補足の法令名不一致",
			official: validOfficialFixture,
			supplemental: strings.Replace(
				validSupplementalFixture,
				`"個人情報の保護に関する法律"`,
				`"別の法律"`,
				1,
			),
		},
		{
			name:     "HTTPS以外の出典",
			official: validOfficialFixture,
			supplemental: strings.Replace(
				validSupplementalFixture,
				`https://hourei.ndl.go.jp/`,
				`http://hourei.ndl.go.jp/`,
				1,
			),
		},
		{
			name:     "未知の補足種別",
			official: validOfficialFixture,
			supplemental: strings.Replace(
				validSupplementalFixture,
				`"kind": "common-abbreviation"`,
				`"kind": "guessed"`,
				1,
			),
		},
		{
			name:     "不正な確認日",
			official: validOfficialFixture,
			supplemental: strings.Replace(
				validSupplementalFixture,
				`"confirmedAt": "2026-07-27"`,
				`"confirmedAt": "2026-02-30"`,
				1,
			),
		},
		{
			name:     "正規化後に別法令と衝突する補足略称",
			official: validOfficialFixture,
			supplemental: `{
  "schemaVersion": 1,
  "datasetId": "test-supplemental",
  "entries": [
    {
      "lawId": "425AC0000000027",
      "lawNumber": "平成二十五年法律第二十七号",
      "title": "行政手続における特定の個人を識別するための番号の利用等に関する法律",
      "alias": "個 人 情 報 保 護 法",
      "kind": "common-abbreviation",
      "sourceName": "国立国会図書館 日本法令索引",
      "sourceUrl": "https://hourei.ndl.go.jp/simple/detail?current=1&lawId=0000129572",
      "confirmedAt": "2026-07-27"
    }
  ]
}`,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load(
				[]byte(testCase.official),
				[]byte(testCase.supplemental),
			); err == nil {
				t.Fatalf("SOT-ENG-022: 不正な辞書を受理しました")
			}
		})
	}
}

func TestLoadRejectsMissingOrOversizedDatasets(t *testing.T) {
	t.Parallel()

	if _, err := Load(nil, []byte(validSupplementalFixture)); err == nil {
		t.Fatal("SOT-ENG-022: 空の公式辞書を受理しました")
	}
	if _, err := Load(
		[]byte(validOfficialFixture),
		make([]byte, maxDatasetBytes+1),
	); err == nil {
		t.Fatal("SOT-ENG-022: 上限を超える補足辞書を受理しました")
	}
}

func TestEmbeddedLexiconContainsCompleteEGovSnapshotAndSupplements(
	t *testing.T,
) {
	t.Parallel()

	lexicon, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ENG-022: LoadEmbedded() のエラー = %v", err)
	}
	entries := lexicon.Entries()
	if len(entries) != 9536 {
		t.Fatalf("SOT-ENG-022: e-Gov 法令数 = %d, want 9536", len(entries))
	}

	assertLexiconTerm(
		t,
		entries,
		"335AC0000000105",
		"道路交通法",
		"道交法",
	)
	assertLexiconTerm(
		t,
		entries,
		"415AC0000000057",
		"個人情報の保護に関する法律",
		"個情法",
	)
	assertLexiconTerm(
		t,
		entries,
		"425AC0000000027",
		"行政手続における特定の個人を識別するための番号の利用等に関する法律",
		"マイナ法",
	)
	assertLexiconTerm(
		t,
		entries,
		"322AC0000000049",
		"労働基準法",
		"労基法",
	)
	assertLexiconTerm(
		t,
		entries,
		"322AC0000000054",
		"昭和二十二年法律第五十四号（私的独占の禁止及び公正取引の確保に関する法律）",
		"独禁法",
	)
}

func assertLexiconTerm(
	t *testing.T,
	entries []Entry,
	resourceID string,
	canonical string,
	term string,
) {
	t.Helper()
	for _, entry := range entries {
		if entry.ResourceID != resourceID {
			continue
		}
		if entry.Canonical != canonical || !containsString(entry.Terms, term) {
			t.Fatalf("SOT-ENG-022: entry = %#v", entry)
		}
		return
	}
	t.Fatalf("SOT-ENG-022: lawId %q がありません", resourceID)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
