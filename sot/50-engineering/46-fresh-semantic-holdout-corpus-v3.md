# SOT-ENG-046: 空入力境界を分離する fresh 意味 holdout corpus schema version 3

- 状態: 有効

## 規定

新しい候補の fresh 意味 holdout は corpus schema version 3 とし、
`input-query-empty` の意味家族だけを holdout から分離する。空入力拒否は
同じ commit に対する必須品質ゲートの入力境界検証として維持し、過去の意味家族を
名前の変更で fresh に戻さない。

## 適用範囲と旧版の保持

本規定は、`SOT-ENG-026` の corpus schema version 2 に対する最小の後継契約である。
version 3 の空入力の集合所属、holdout 必須 coverage および版選択だけを置き換え、
その他の field、型、正規形、順序、意味署名、checksum、digest 算出、集合分離、
資源上限、期待値検証、fixture 不変性と loader の安全境界は同 SOT を継承する。

schema version 1 と 2、その schema 原 byte、corpus、fixture、manifest と履歴を
変更、再生成又は version 3 へ変換しない。旧版の holdout における
`input-query-empty` の必須条件も保持する。本規定は、新しい corpus、baseline、
content manifest、review、request 又は pointer の予約成果物を作る指示ではない。

## schema と loader

`testdata/legalquery/schemas/legal-query-corpus-v3.schema.json` を追加し、
`corpus_manifest`、`semantic_case`、`execution_case` の三 variant を閉じて扱う。
manifest とすべての fixture の `schemaVersion` は integer の `3` で一致させる。
版の省略、未知版、混在、近似版又は current・最新版への fallback を拒否する。
manifest 専用 loader、development 専用 loader、全 corpus loader と constructor は、
明示した版に対応する schema と validator を使う。

version 3 の manifest は version 2 の `holdoutLeakageGroupDigests` と
`requiredDevelopmentAssertionIds` を引き続き必須とし、計算方法と閉じた ID 集合を
変更しない。development に要求する十一 assertion と execution の八 scenario を
維持する。version 3 に旧 corpus 名を指定しても execution scenario を七件へ戻さない。
旧版の historical corpus に対する七 scenario の互換規則はその宣言版だけで保持する。

## 空入力だけの分離

version 3 の holdout は、次のいずれかに当たる semantic case を拒否する。

- `coverageIds` に `input-query-empty` を持つ。
- `request.query` が空文字列である。
- `request.query` が Unicode の空白だけであり、製品 request constructor と同じ
  Unicode 空白の前後除去後に空文字列になる。

この判定は期待値、ほかの coverage、case ID、leakage group ID、ref 又は limit の
付替えから独立して行う。空入力の coverage を削除し、別の拒否入力の coverage を
名乗っても受理しない。同じ入力意味家族を同じ `leakageGroupId` にする
`SOT-ENG-026` の規則と、過去の全 request が予約した家族を再利用しない
`SOT-ENG-042`、`SOT-ENG-043` および `SOT-ENG-045` の規則は変更しない。

version 3 の development は、入力境界の校正又は回帰確認のために空入力と
`input-query-empty` を保持できる。development での収録は任意とし、後述する
同 commit の必須品質ゲートを代替しない。execution は従来どおり受理可能な
semantic plan を参照し、空入力を実行対象へ移さない。

version 3 の holdout 必須 coverage は、`SOT-ENG-026` の version 2 の六十一件から
`input-query-empty` 一件だけを除いた六十件とする。残る各 coverage の category、
最小件数と safety pair 条件は変えない。とくに `input-invalid-ref`、
`input-limit-above-maximum`、`input-limit-below-minimum`、
`input-query-ascii-control`、`input-query-too-long` の五拒否入力 coverage を維持する。

holdout の二百四十件以上、全十二 category 各二十件以上、四百件以下の上限、
安全性の ordinary/adversarial pair と四つの派生観測の存在検証も維持する。
`SOT-ENG-024` の全 metric、母集団の定義、計算式及び受入閾値、
`SOT-ENG-036` の report schema version 1 を変更しない。

## 同 commit の入力境界品質ゲート

空文字列及び Unicode 空白だけの query を製品 request constructor と公開 MCP が
ともに `invalid_argument` として拒否し、provider を呼び出さないことを固定の
入力境界 test で検証する。対象入力の拒否率は `100%`、provider 呼出し件数は `0` とし、
application と MCP の入力エラー分類を一致させる。

この検証は fresh holdout の消費、意味家族の新規予約又は report の新 metric に
しない。`SOT-ENG-020` と `SOT-ENG-027` の同じ対象 commit に対する CI 全 test の
必須項目として実行し、一件の失敗も許容しない。後続の候補評価では、対象 commit の
権威 CI によるこの検証の成功を必要とする。development fixture の有無又は意味
holdout の成功だけで省略しない。

## 確認

実 holdout 内容を開かず合成 fixture で次を確認する。

- `TestCorpusV3は空入力だけをHoldoutから分離する`: 空・Unicode 空白、coverage 付替え、
  development の受理及び schema version 1 と 2 の空入力互換
- `TestCorpusV3は残る60Coverageと旧版契約を維持する`: 残る全 coverage の個別不足拒否、
  旧版の空入力必須条件及び未知版の拒否
- `TestCorpusV3は合成Corpusを版別Loaderで受理する`: 三 variant、manifest 専用及び
  development 専用 loader、十一 assertion と八 scenario を含む合成 corpus の受理

入力境界は `TestNewRequestRejectsInvalidQuery` と
`TestQueryLegalInformationToolRejectsInvalidInputBeforeApplication` の空文字及び
Unicode 空白 case で確認する。これらの test、SOT 構造、開発原則 checksum と
対象解析をローカルで確認し、
全体の品質・coverage は `SOT-ENG-027` の権威 CI へ委ねる。

## 関連

- [SOT-ENG-020](20-verification-gate.md)
- [SOT-ENG-024](24-unified-query-evaluation-gate.md)
- [SOT-ENG-026](26-legal-query-corpus-artifact-contract.md)
- [SOT-ENG-027](27-resource-aware-verification-stages.md)
- [SOT-ENG-036](36-unified-query-evaluation-baseline-artifact-contract.md)
- [SOT-ENG-038](38-content-bound-candidate-evaluation-handoff.md)
- [SOT-ENG-042](42-candidate-evaluation-handoff-schema-v3.md)
- [SOT-ENG-043](43-candidate-evaluation-readiness-separation.md)
- [SOT-ENG-045](45-candidate-evaluation-handoff-schema-v4.md)
