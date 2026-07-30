# SOT-IF-053: MCP `search_laws` v3

- 状態: 有効

## 規定

`search_laws` は、境界を明示した法令名、略称、法令名を一つ含む自然文または
軽微な誤記を受け取り、選択済みの公式情報源で確認できる
`LawSearchResult` を返す MCP ツールとする。

本規定は `SOT-IF-049` を置き換える。入力と出力の JSON schema
および二回以下の情報源呼出しを維持し、解決済み法令対象の page 内安定優先を
追加する。

対応する利用シナリオは、廃止した `SOT-SCN-008` を置き換える
`SOT-SCN-011` とする。

## 入力

| 名前 | 型 | 必須 | 制約 | 意味 |
|---|---|---:|---|---|
| `query` | string | はい | 正規化後 1 文字以上、UTF-8 で 512 byte 以下 | 法令を特定するための検索語 |
| `asOf` | string | いいえ | `2017-04-01` 以降の `YYYY-MM-DD` | 指定日以前で最新のリビジョンを検索する基準日 |
| `limit` | integer | いいえ | 1 以上 100 以下、既定値 20 | 一ページに返す法令数 |
| `offset` | integer | いいえ | 0 以上 2147483647 以下、既定値 0 | 取得を開始する法令位置 |

`query` は、先頭と末尾に連続する U+0020 を除いた値を検証済み原文として、
byte 数の判定と最初の検索に使用する。U+0000 から U+001F および U+007F の
ASCII 制御文字を含む値は `invalid_argument` とする。

検証済み `query` の先頭と末尾がともに U+002F `/` である値は、e-Gov の
正規表現指定と区別できないため受け付けない。この公開ツールは正規表現検索を
契約にせず、該当値を `invalid_argument` とする。

欠落、`null`、検証後の空値、上限超過、日付不正および定義していない入力項目は
受け付けない。

## 検索語の解決

検証済み原文から、情報源を呼ぶ前に `SOT-ARCH-030` の
`ResolvedLawTarget` を作れるか判定する。対象を作れなくても原検索を拒否しない。

対象の照合順、自然文の span 境界、誤記距離、一意性および解決しない条件は
`SOT-ARCH-030` を定義元とする。比較用の正規化値、読み、token または
編集途中の文字列を情報源へ送らず、確認検索には解決済み対象の
`officialTitle` だけを使用する。

## 検索と結果順

`offset` が 0 の場合もそれ以外の場合も、検証済み原文を変更せず最初に検索する。
原検索がエラーを返した場合は、別の検索語を試さずそのエラーを返す。

原検索が正常な非空結果を返した場合は、解決済み対象が同じ page にあるときだけ
`SOT-ARCH-030` の安定優先を適用し、それ以外は provider 順を保持して返す。
対象が page にないことを理由に確認検索を行わない。

原検索が正常な空結果を返し、解決済み対象が一つある場合だけ、
`officialTitle` で同じ provider を確認検索する。確認検索には原検索と同じ
`asOf`、`limit` および `offset` を使用し、正常な page に同じ安定優先を
適用して返す。

解決済み対象がない場合または確認検索も正常な空結果の場合は、最初の空結果を
返す。確認検索がエラーを返した場合は、そのエラーを返す。原検索と確認検索を
結合せず、総件数、取得位置または item を独自に合成しない。

候補の law ID が現在の page にない場合は item を作らず、別 page を取得せず、
provider 順の page をそのまま返す。安定優先は page 内の item 順だけを変更し、
`totalCount` と `nextOffset` を変更しない。

## offset

原検索の空判定は `items` の長さではなく、ページ位置と独立した
`totalCount` が 0 であるかによって判定する。`offset` が 0 より大きく、
現在の page の `items` が空でも `totalCount` が 1 以上なら、確認検索へ
切り替えない。

すべての検索は一つの公開リクエスト期限と context cancellation に従う。
二回上限の単位は、原検索と確認検索として facade が provider port へ渡す
論理検索 operation とし、合わせて最大二回とする。一つの論理検索 operation
内で e-Gov client が行う HTTP の自動再試行は `SOT-IF-004` に従い、この
二回へ数え直さない。実際の HTTP attempt の上限は同 SOT だけを定義元とする。

## 出力

`SOT-MODEL-006` の `LawSearchResult` を返す。結果がない場合は
`totalCount: 0` と空の `items` を返す。

`nextOffset` は 0 以上 2147483647 以下で、選択した一つの検索条件に対して
e-Gov が `next_offset` として返した場合、または `SOT-IF-054` に従い
e-Gov が返した `count` と `total_count` から `offset + count` を導出できる
場合だけ返す。法令名解決または page 内安定優先から `nextOffset` を作らない。

解決方法、内部 token、辞書の出典、補正前後の検索語および
`ResolvedLawTarget` を出力へ追加しない。

## エラー

- 入力が制約を満たさない場合は `invalid_argument` を返す。
- 外部情報源の制限、一時障害または現在のローカル同時実行上限は、原因に応じて
  `rate_limited`、`source_timeout`、`source_unavailable` または
  `source_busy` を返す。
- 外部契約、応答または安全上限の問題は、原因に応じて
  `source_contract_changed`、`invalid_source_response`、
  `source_response_too_large`、`source_processing_limit` または
  `unsafe_source_content` を返す。
- 起動時に検証済みの前処理がリクエスト中に予期せず失敗した場合を含み、
  上記へ分類できない内部処理の失敗は `internal_error` を返す。

各 code の `retryable`、`details` および秘密情報の禁止は `SOT-IF-027` に従う。

## 互換 facade

このツールは、e-Gov の数値 `offset` を維持する provider 固有の互換 facade
とする。`SOT-IF-022` の内部 capability と同じ入力検証、明示された `asOf`、
`limit`、応答 parser、`LawSummary` mapping および page 不変条件を再利用する。
公開 `offset` を `continuationToken` に変換して内部 capability を直接呼ばない。

法令名解決と安定優先は adapter の外側で行い、選択した各検索語は既存 facade へ
独立した request として渡す。facade と e-Gov adapter に辞書、Kagome、
`ResolvedLawTarget` または誤記規則を持ち込まない。無設定起動の選択済み
provider は e-Gov 法令 API Version 2 とする。

## 公開 schema

`inputSchema` は `type: object`、`additionalProperties: false` とし、
property は `query`、`asOf`、`limit` および `offset` の四つだけとする。
`required` は `query` だけとし、各 property の型、範囲および既定値は
本規定の入力表に一致させる。byte 数、実在する暦日および制御文字のように
JSON Schema だけで完全に表せない制約も handler の入力検証から省略しない。

`outputSchema` は `type: object`、`additionalProperties: false` とし、
`totalCount` と `items` を必須、`nextOffset` を省略可能とする。
各項目の型と制約は `SOT-MODEL-006`、各 item は `SOT-MODEL-001`、
item の `source` は `SOT-MODEL-003` に従う。これらの object も
各モデルが定義する項目だけを持ち、`additionalProperties: false` とする。
省略可能な値は `null` にせず、`SOT-MODEL-009` に従い property 自体を省略する。

順位以外の公開項目を追加、削除または改名せず、stdio と Streamable HTTP は
同じ `inputSchema`、`outputSchema` および結果形を使用する。

## 確認

原検索の error を保存すること、正常な空結果だけが確認検索を許すこと、
二回の論理検索 operation 上限、および原検索と確認検索を集約しないことを
fake provider で確認する。HTTP の自動再試行回数は e-Gov client の契約 test で
別に確認し、facade の論理検索回数へ加算しない。

正式名称、公式略称、補足略称、自然文、互換表記、挿入、削除、置換、
隣接文字の転置、短い語、衝突する略称、同率候補および複数法令を含む文を
fixture にする。原検索または確認検索の三件目に解決済み law ID がある場合は
先頭へ移し、ほかの item の相対順、件数、次位置および出典を保持することを
確認する。

`著作券法` と同じ誤記を一つ含む自然文は、解決先が一つの場合だけ
`著作権法` の law ID を優先する。対象 law ID が page にない場合は結果を
捏造せず、別 page を取得しないことも確認する。

MCP 契約 test では、本規定の公開 schema に対して構造的な完全一致を確認する。
入力の四項目、required、型、範囲、既定値および未知項目の拒否と、出力の
`totalCount`、`items`、任意の `nextOffset`、各 item の `LawSummary`、
`LegalSource` および全 object の `additionalProperties: false` を固定する。
stdio と Streamable HTTP で同じ schema と結果形を使用することも確認する。

## 関連

- [SOT-SCN-001: 法令名から法令を検索する](../10-scenarios/01-search-laws.md)
- [SOT-SCN-011: 解決済み法令を検索結果で優先する](../10-scenarios/11-prioritize-resolved-law-search-result.md)
- [SOT-MODEL-001: LawSummary](../20-model/01-law-summary.md)
- [SOT-MODEL-003: LegalSource](../20-model/03-legal-source.md)
- [SOT-MODEL-006: LawSearchResult](../20-model/06-law-search-result.md)
- [SOT-MODEL-009: JSON シリアライズ](../20-model/09-json-serialization.md)
- [SOT-ARCH-021: プロバイダー非依存の検索語前処理](../30-architecture/21-provider-independent-query-preprocessing.md)
- [SOT-ARCH-030: 解決済み法令対象の検索結果優先順位](../30-architecture/30-canonical-law-target-priority.md)
- [SOT-IF-022: law.search capability v1](22-law-search-capability.md)
- [SOT-IF-027: 公開情報源エラー契約](27-public-source-error-contract.md)
- [SOT-IF-004: e-Gov 法令 API Version 2](04-source-egov-law-api-v2.md)
- [SOT-IF-054: e-Gov 法令名検索マッピング v3](54-egov-law-search-mapping-v3.md)
- [SOT-ENG-022: 法令名検索辞書](../50-engineering/22-law-name-search-lexicon.md)
