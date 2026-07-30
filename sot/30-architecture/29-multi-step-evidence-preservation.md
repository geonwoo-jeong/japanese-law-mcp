# SOT-ARCH-029: 複数 step 候補の根拠保持

- 状態: 有効

## 規定

一つの query profile が、照会文で独立に明示された複数の取得意図を一つの
`LegalQueryCandidate` へまとめる場合は、各 logical step を成立させた検証済みの
根拠を候補全体へ保持する。

## 根拠の統合

複数 step 候補の `evidenceCodes` は、各 step の生成に使用した根拠の和集合を
`SOT-MODEL-022` の強い順に並べ、重複を除いた値とする。

一つの step または同じ対象の同値な生成経路について、公式識別子と正式名称が
同じ対象を重複して根拠付ける場合は、profile の版付き規則によって弱い根拠を
省略できる。一方、強い根拠が別の step を成立させただけである場合は、それを
理由に、独立した step を成立させた弱い根拠を削除しない。

例えば、正式名称による法令検索と、法令 ID による法令読取りを一候補へまとめる
場合は、`official_identifier` と `official_alias` をともに保持する。法令 ID
だけで一つの読取り step を作り、同じ対象の正式名称が同じ step の別経路として
重複した場合は、`official_identifier` だけに正規化できる。

法概念または形態素文脈が独立した条文検索を成立させた場合も同じ規則を適用する。
最終候補へ step ごとの内部根拠対応を公開せず、profile は候補の materialize 前に
根拠を統合する。

## 境界

本規定は、一 profile 内で複数 step を組み立てる際の根拠保持を定める。
profile 横断の候補合成は `SOT-ARCH-027` に従う。根拠の保持によって明示されて
いない step を追加したり、同じ入力から別の取得対象を推測したりしない。

根拠の統合規則を変更する場合は `SOT-ENG-024` に従って新しい profile version
を割り当てる。

## 確認

正式名称による検索と公式識別子による読取り、公式参照による読取りと法概念検索、
同じ step の重複根拠、および三 step 以上の組合せを、ネットワークを使わない
profile test で確認する。

複数 step では独立した根拠を保持し、同じ step の重複根拠は版付き規則どおりに
正規化し、最終順序が `SOT-MODEL-022` と一致することを確認する。

## 関連

- [SOT-MODEL-022: LegalQueryCandidate](../20-model/22-legal-query-candidate.md)
- [SOT-MODEL-025: LegalQueryPreprocessResult](../20-model/25-legal-query-preprocess-result.md)
- [SOT-ARCH-025: 統合照会の複数主題分離](25-unified-query-multi-topic-separation.md)
- [SOT-ARCH-027: 統合照会の profile 横断候補合成](27-unified-query-cross-profile-composition.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](../50-engineering/24-unified-query-evaluation-gate.md)
