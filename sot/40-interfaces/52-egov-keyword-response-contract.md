# SOT-IF-052: e-Gov キーワード検索 JSON 応答の受理契約

- 状態: 有効

## 規定

e-Gov 法令 API Version 2 の `GET /keyword` を使用する全入口は、一つの
provider 固有 parser によって、JSON 応答の構造、欠落、`null`、追加項目および
不正値を同じ規則で判定してから、公開 facade または共通 capability の
モデルへ変換する。

## 公式仕様との関係

本規定は、2026年7月30日に確認した
[法令 API Version 2 OpenAPI](https://laws.e-gov.go.jp/api/2/swagger-ui/lawapi-v2.yaml)
の `keyword_response`、公式 JSON 例および各項目の説明を外部仕様の基準とする。

同 OpenAPI の `keyword_response.required` は、property に存在しない `count`
だけを列挙しており、説明、properties および公式例と整合しない。そのため
adapter は `required` 配列を機械的な受理条件にせず、共通モデルを安全に
作るために必要な次節の意味上の必須項目を provider contract とする。

本規定は、現在の runtime parser が一部の個別応答の構造違反と省略可能項目の
`null` を `source_contract_changed` とする実装を変更する採用済み理想状態を
定義する。実装が本規定へ移行するまでの差分は Wiki で追跡する。

adapter は `response_format=json` を明示して要求する。成功応答は media type
が `application/json` である場合だけ JSON として解析し、同 endpoint が
提供できる XML を自動判定または fallback として解析しない。

## 受理する JSON 構造

top-level は一つの JSON object とし、次を検証する。

| 位置 | 必須 | 受理する値 |
|---|---:|---|
| `total_count` | はい | `0` 以上で実装の安全な integer 範囲内の整数 |
| `sentence_count` | はい | `0` 以上で実装の安全な integer 範囲内の整数 |
| `next_offset` | いいえ | `null` または安全な integer 範囲内の整数 |
| `items` | はい | `null` ではない JSON array |

`items` の各要素は object とし、`law_info`、`revision_info` および
`sentences` を持つ。`law_info` と `revision_info` は object、
`sentences` は一件以上の array とする。

共通モデルを識別する次の項目は、`null` ではない空でない string とする。

- `law_info.law_id`
- `revision_info.law_revision_id`
- `revision_info.law_title`
- `sentences[].position`
- `sentences[].text`

`LawSummary` の省略可能項目へ対応する `law_info.law_num`、
`law_info.promulgation_date` および
`revision_info.amendment_enforcement_date` は、欠落または `null` の場合に
値を推測せず共通モデルから省略する。string で存在する場合は、対応する
法令番号または暦日の型を満たさなければならない。

公式 schema に追加された未知の項目は、上記の必須項目、資源予算または
既知項目の型を変えない限り無視できる。未知項目を共通モデルの
`map[string]any`、provenance または公開 JSON へ移さない。

単一の item または sentence であっても object へ縮約した形は受理せず、
公式 schema どおり array を要求する。空結果は `total_count: 0`、
`sentence_count: 0` および空の `items` を持つ成功として受理する。

## 値の整合と正規化

`sentence_count` は全 `items[].sentences` の件数の総和と一致し、要求した
`limit` 以下でなければならない。`next_offset`、`offset`、
`sentence_count` および `total_count` の関係は `SOT-IF-010` と
`SOT-IF-028` が定める一致位置単位の同じページ不変条件で検証する。

`highlight_tag=mark` を送信した応答では、`sentences[].text` から API が
挿入した小文字の `<mark>` と `</mark>` だけを除去する。ほかの tag、文字、
空白または entity を変更、実行または HTML として解釈しない。

同じ parser 結果を、`search_law_content` facade では
`LawContentSearchResult`、`law.content.search@1` では
`SourcedResource<LawContentMatch>` の page へ変換する。入口ごとに
JSON shape、`null` または error 分類を実装し直さない。

検索語、法令名または返却件数に固有の例外を設けない。`永住許可` その他の
回帰語は fixture 名または任意の live 確認に使用できるが、runtime の分岐条件、
補完値または error 抑制条件にしない。

## エラー分類

一件の外部応答だけから、公式 schema 自体が変更されたと推測しない。
runtime parser と、公式一次資料を確認する provider contract 検証の分類を
次のように分ける。

| 検出位置 | 条件 | code |
|---|---|---|
| runtime parser | top-level、item、sentence、必須 object・array・scalar の欠落、`null` または型不一致 | `invalid_source_response` |
| runtime parser | 負の件数、空の必須文字列、不正な日付、件数不一致、limit 超過、offset または総件数の矛盾 | `invalid_source_response` |
| runtime parser | `response_format=json` に対する JSON 以外の成功時 media type、または JSON として解析できない body | `invalid_source_response` |
| provider contract 検証 | 現在の公式 OpenAPI、media type または公式例を一次資料で確認し、記録した受理契約の必須構造若しくは型が変更された | `source_contract_changed` |

したがって、個別応答の `law_info`、`revision_info`、`sentences` その他の
必須 container が欠落、`null` または別の JSON 型であっても、公式仕様の
変更を別途確認していない runtime では `invalid_source_response` とする。

省略可能な `law_num`、`promulgation_date` および
`amendment_enforcement_date` の欠落または `null` は error にせず、いずれも
値を持たない同じ共通モデルへ対応させる。存在する省略可能項目の型または
値が不正な場合は `invalid_source_response` とする。

`source_contract_changed` は、壊れた個別応答の別名として使用しない。
公式仕様の変更を確認した場合は、受理範囲を推測で広げず、本規定と fixture を
更新するまで fail closed とする。

- 2xx で error object、trailing JSON、壊れた圧縮 body または安全上限を
  満たさない body を成功結果へ変換しない。公開 code は
  `SOT-IF-017` と `SOT-IF-027` に従う。

外部応答本文、検索語、URL query、内部 decoder error および未知項目の値を
公開 error details に含めない。

## 確認

公式 JSON 例、正常な空結果、単一法令・複数 sentence、複数法令、
省略可能項目の欠落と `null`、未知の追加項目、および
複数法令に一致する `永住許可` の匿名化した固定 snapshot を fixture にする。

意味上の各必須項目と container について欠落、`null` および型不一致を別々に
確認し、いずれも `invalid_source_response` であることを固定する。
省略可能三項目の欠落と `null` は同じ共通モデルとなることを確認する。
単一 object への配列縮約、空の必須文字列、空の `sentences`、件数不一致、
limit 超過、残件がある `next_offset` の欠落・`null`、非前進、範囲外および
int32 上限超過を `invalid_source_response` として固定する。

provider contract 検証では、保存した公式 schema と取得した公式 schema の
必須構造または型を意図的に変えた fixture を使い、個別応答の失敗とは別に
`source_contract_changed` を確認する。

同じ fixture を facade と capability の両入口へ渡し、同じ parser 判定と
同じ item 展開順になることを確認する。外部ネットワークを使う確認は
`SOT-ENG-013` に従って任意の定期確認へ分離し、fixture test の代わりにしない。

`SOT-IF-010` と `SOT-IF-028` の `GET /keyword` 応答受理と error 分類は
本規定を定義元とする。request mapping と page 不変条件の定義元は、
それぞれの既存 SOT に維持する。

## 関連

- [SOT-IF-004: e-Gov 法令 API Version 2](04-source-egov-law-api-v2.md)
- [SOT-IF-010: e-Gov 本文検索マッピング](10-egov-content-search-mapping.md)
- [SOT-IF-017: 情報源エラーの正規化](17-source-error-normalization.md)
- [SOT-IF-023: law.content.search capability v1](23-law-content-search-capability.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [SOT-IF-028: e-Gov 構造化本文検索マッピング](28-egov-structured-content-search-mapping.md)
- [SOT-IF-033: MCP `search_law_content`](33-mcp-search-law-content.md)
- [SOT-ENG-013: プロバイダー契約の検証](../50-engineering/13-provider-contract-verification.md)
