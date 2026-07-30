# ADR-004: 製品辞書の根拠だけで評価するコーパス Version 4

- 状態: 採用
- 決定日: 2026-07-28

## 背景

法令コア profile の development 集合を独立 review したところ、三件が製品の組込み辞書に存在しない法令名事実を test 内で追加していた。

- `開示法`を二つの情報公開法へ対応させていた。
- `海商法`を法令名`商法`へ対応させていた。
- `ＡＰＰＩ`を個人情報保護法の略称として対応させていた。

これらの表記が別の文脈で用いられることと、当該法令への対応を組込み辞書の根拠として採用していることは同じではない。test だけに事実を追加すると、製品 binary が持たない語彙で profile の成立を確認することになる。

## 決定

`corpus-v1`から`corpus-v3`は変更せず、同じ schema version、seed および holdout を持つ`corpus-v4`を追加する。三件は、取得元を記録した組込み e-Gov 法令 API Version 2 スナップショットに実在する正式名称または略称へ置き換え、入力と期待意味を変えるため新しい case ID を割り当てる。

| Version 3 | Version 4 | 根拠 |
|---|---|---|
| `development-ambiguity-alias`の`開示法` | `development-ambiguity-provisional-law`の`暫定法` | e-Gov スナップショットで二法令に対応する略称 |
| `development-execution-timeout`の`海商法` | `development-execution-timeout-commercial-code`の`商法` | e-Gov スナップショットの正式名称 |
| `development-surface-width`の`ＡＰＰＩ` | `development-surface-width-jas-law`の`JAS法` | e-Gov スナップショットの略称`ＪＡＳ法`に対する文字幅差 |

`execution-timeout`は参照する semantic case ID だけを新しい ID へ変更し、scenario、action および期待結果は変えない。test 専用の法令名注入は削除し、製品と同じ`querypreprocess.NewEmbedded`だけで development の意味候補を生成する。

文字幅差の検証では、辞書一致済みの`JAS法`の内部部分`JAS`が誤記候補として重複していた。辞書一致 span と重なる誤記候補は作らず、比較用正規化一致を優先するようにする。

固定値は次のとおりである。

| 項目 | Version 3 | Version 4 |
|---|---:|---:|
| development 件数 | `31` | `31` |
| holdout 件数 | `240` | `240` |
| execution 件数 | `7` | `7` |
| seed | `20260727` | `20260727` |
| holdout digest | `25b06db5e29ada2922e970a8c569ee9d2e73ce22fae4ffe3ce5880043d3543b8` | `25b06db5e29ada2922e970a8c569ee9d2e73ce22fae4ffe3ce5880043d3543b8` |

この決定時点では初回 profile set と評価 command の既定 corpus を
`corpus-v4` とした。過去三版も loader の再現性試験を残す。後に
[ADR-005](05-reviewed-evaluation-corpus-v9.md) が既定 corpus の指定だけを
`corpus-v9` へ置き換えた。

## 検証結果

- `corpus-v4`は、loader による schema、fixture checksum、集合分離、最小件数、参照および holdout digest の検証に成功した。
- 法令コア profile は、test 専用の辞書事実なしで development の対象二十九件に期待する意味候補を生成した。
- `JAS法`は`comparison_normalized`の一件として抽出され、重なる`unique_typo_correction`を生成しない。
- holdout の byte 列と digest は変更していない。
- 評価 command と baseline は未実装であるため、この変更だけを全評価ゲートの成功とは扱わない。

## 帰結

- development test と製品 binary が同じ法令名辞書を使用する。
- 出典を採用していない表記を、test だけで公式略称のように扱わない。
- 過去の入力と期待値は`corpus-v3`から再現できる。
- 今後も入力または期待意味を変える場合は、[`SOT-ENG-026`](../../sot/50-engineering/26-legal-query-corpus-artifact-contract.md) に従って新しい corpus version と case ID を割り当てる。
