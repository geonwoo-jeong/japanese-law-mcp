# ADR-005: 独立 review 済みの評価コーパス Version 9

- 状態: 採用
- 決定日: 2026-07-30

## 背景

`corpus-v4` に対して現行の default profile set を評価したところ、意味候補、
順位または実行動作ではなく fixture の期待 assertion に不一致が残っていた。
各不一致を製品実装へ合わせて機械的に変更せず、対象を限定した独立 review で
製品不具合か fixture 誤りかを判定した。

`corpus-v5` から `corpus-v9` は、その判定を append-only の版として順に残す。

| version | 訂正した期待値 |
|---|---|
| `corpus-v5` | 衝突する略称誤記の選択 |
| `corpus-v6` | profile 横断の予算統合で保持する法概念根拠 |
| `corpus-v7` | 複数 step 候補で保持する正式名称根拠 |
| `corpus-v8` | 独立した形態素検索 step の根拠 |
| `corpus-v9` | 法令名の明示 task・誤記根拠と同一意味へ収束する法概念出典 |

各版は直前の corpus を変更せず新しい directory、manifest checksum および
holdout digest を持つ。profile、辞書、ranking、request、意味署名および
execution fixture は、期待値訂正と同じ変更で改変しない。

## 決定

default profile set の標準評価 corpus を `corpus-v9` とし、過去版は loader と
訂正前後の再現用成果物として保持する。標準 command は `corpus-v9` と
`default-1` baseline の固定組合せだけを受け付ける。

固定値は次のとおりである。

| 項目 | 値 |
|---|---:|
| development 件数 | `32` |
| holdout 件数 | `240` |
| execution 件数 | `8` |
| seed | `20260727` |
| holdout digest | `c72a7a9504465f1d64a02ec9cb82d6a9faf6f382bee546c30439513f42d47557` |

`SOT-ENG-024` の主語、義務および適用範囲は変えず、同 SOT が固定する標準
corpus version だけを現在の review 済み成果物へ更新する。将来の corpus 更新も
同じく新しい version、独立 review、変更前後の評価および baseline 更新を
一つの変更として扱う。

## 検証結果

- `corpus-v9` は schema、fixture checksum、集合分離、最小件数、参照および
  holdout digest の検証に成功した。
- 同じ入力を二回評価した plan は `224/224` 一致し、期待 decision、reason
  および selection は `224/224` 一致した。
- 意味署名は `206/206`、top-1 と top-2 はそれぞれ `191/191`、
  high-confidence precision は `176/176` 一致した。
- 根拠 assertion は `206/206`、法概念 assertion は `19/19` 一致した。
- 導出した四種類の profile 横断観測と八件の execution fixture は、すべての
  指標が一致した。誤った resource 呼出し、予算違反、暗黙の第一件 read
  および空結果後の再分類はいずれも `0` 件だった。
- 各 corpus 訂正は、独立 review で 8 点を超え、blocker なしと判定された。

## 帰結

- 中央品質 gate は、実際の製品 profile がすべての期待 assertion を満たす
  `corpus-v9` を使用する。
- baseline は、現在の profile version、ranking version、corpus digest および
  すべての測定値を一つの決定的な JSON 文書へ固定する。
- 過去の誤った期待値と訂正理由は、`corpus-v4` から `corpus-v8` および
  専用の回帰 test で再現できる。
- 標準 corpus または baseline だけを、現在の実装結果に合わせて暗黙に
  上書きできない。
