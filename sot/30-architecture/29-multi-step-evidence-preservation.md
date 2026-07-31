# SOT-ARCH-029: 複数 step 候補の根拠保持

- 状態: 廃止
- 廃止理由: 同じ step 内で弱い根拠を省略する条件が選択的で、候補の
  `evidenceCodes` と score を一意に決められなかったため
- 後継: [SOT-ARCH-036: 複数 step 候補の step 内根拠正規化と保持](36-multi-step-evidence-normalization.md)

## 規定

一つの query profile が、照会文で独立に明示された複数の取得意図を一つの
`LegalQueryCandidate` へまとめる場合は、各 logical step を成立させた検証済みの
根拠を候補全体へ保持する。

## 根拠の統合

複数 step 候補の `evidenceCodes` と `semanticScore` の入力に使う根拠 code は、
まず各 step の内部で、その step と同じ対象を同じ意味で生成した同値経路だけを
正規化する。その後、正規化済みの各 step の根拠を候補全体へ和集合し、
`SOT-MODEL-022` の強い順に並べ、重複する code を除いた同じ集合とする。
profile は、この正規化前の候補全体の集合から score を計算した後で
`evidenceCodes` だけを正規化してはならない。code ごとの weight と score の
計算式は版付き profile metadata を定義元とし、本規定では変更しない。

一つの step または同じ対象の同値な生成経路について、公式識別子と正式名称が
同じ対象を重複して根拠付ける場合は、profile の版付き規則によって弱い根拠を
省略できる。一方、強い根拠が別の step を成立させただけである場合は、それを
理由に、独立した step を成立させた弱い根拠を削除しない。

候補全体の code 和集合を先に作り、その中の強い code を理由に全 step の弱い
code を一括で削除してはならない。正規化の単位は
`{step の意味署名, 同じ対象, 同値な生成経路}` とし、別の `topicOrdinal`、
別の logical input または別の取得対象に属する根拠を同値経路として扱わない。
同じ span に異なる kind の位置付き事実がある場合は、
`SOT-ARCH-031` と profile 固有 SOT が同じ意味の同値経路と定めた組だけを
正規化し、近接、同一文字列または候補全体の code 一致だけで統合しない。

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

複数 step を構築する各 profile は、profile 固有の固定検証 ID を一つ持ち、
同じ step の `official_identifier` と同値な `official_alias` は版付き規則どおりに
正規化できる一方、別 step を成立させた `official_alias`、`legal_concept`、
`morphological_context` または `general_term` は、他の step に強い code が
あることを理由に失われないことを確認する。三 step 以上でも step 内の正規化を
先に行い、その後の和集合と最終順序が `SOT-MODEL-022` に一致し、同じ集合から
`semanticScore` を計算することを確認する。
異なる profile の変更単位で同じ固定検証 ID を再利用しない。

## 関連

- [SOT-MODEL-022: LegalQueryCandidate](../20-model/22-legal-query-candidate.md)
- [SOT-MODEL-025: LegalQueryPreprocessResult](../20-model/25-legal-query-preprocess-result.md)
- [SOT-ARCH-025: 統合照会の複数主題分離](25-unified-query-multi-topic-separation.md)
- [SOT-ARCH-027: 統合照会の profile 横断候補合成](27-unified-query-cross-profile-composition.md)
- [SOT-ENG-024: 統合照会の評価コーパスと受入基準](../50-engineering/24-unified-query-evaluation-gate.md)
