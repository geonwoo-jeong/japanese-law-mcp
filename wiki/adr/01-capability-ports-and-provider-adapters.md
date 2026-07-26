# ADR-001: 能力別ポートとプロバイダー別 anti-corruption layer

- 状態: 採用
- 決定日: 2026-07-25

## 背景

日本の法情報の提供方法は、版管理された API、版を持たない HTML、PDF、ZIP および一括 download に分かれる。外部項目名が似ていても、法令、議案、会議録、通達、裁決および裁判例では識別子、時点、欠落および本文の意味が異なる。

実装は provider ごとに段階的に行う。一方で、新しい provider を追加するたびに共通層または既存 provider を作り直す構造にはしない。現在の公開 MCP ツールとの互換性も維持する。

## 参照した方法論と事例

### Ports and Adapters

[AWS Prescriptive Guidance の Hexagonal architecture](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/hexagonal-architecture.html) は、application が技術非依存の port を持ち、外部 API や storage の技術的交換を adapter が変換することで、domain logic と infrastructure を分離する。複数の入力 provider または出力先を持ち、交換可能性と独立試験が必要な場合に適用する一方、adapter の保守費用も考慮する。

本設計では、すべての operation を一つにまとめず、利用目的と意味が一致する capability ごとに小さい typed port を置く。

### Anti-Corruption Layer

[Microsoft Azure Architecture Center の Anti-Corruption Layer](https://learn.microsoft.com/en-us/azure/architecture/patterns/anti-corruption-layer) は、意味を共有しない subsystem の間に facade または adapter を置き、外部 subsystem の model と API が内部設計を制限しないようにする。また、意味差がない場合まで layer を広げず、入力検証と安全化を境界で行うことを示す。

本設計では、外部 DTO、検索文法、selector、page 方式、認証および error を provider package と mapping SOT に閉じ込める。意味が一致しない項目は共通 model へ押し込まず、より小さい capability または provider-specific capability とする。

### Terraform provider

[HashiCorp Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework) は、provider を外部 API との translation layer とし、provider ごとの configuration、schema、resource または data source、および版を持つ plugin protocol によって core との境界を作る。

本設計は初期段階で動的 plugin process を採用しないが、provider descriptor、typed configuration、capability version、および provider ごとの test artifact を分ける考え方を採用する。動的読み込みを採用しないのは、利用者のローカル単一 binary として配布し、初期の供給網と互換性面を小さく保つためである。

### Kubernetes の out-of-tree provider

[Kubernetes Cloud Controller Manager](https://kubernetes.io/docs/tasks/administer-cluster/developing-cloud-controller-manager/) は、cloud 固有 logic を core から分離し、共通 Go interface を実装する provider が core と異なる周期で変更、公開できる構造を取る。[Kubernetes の provider 分離完了に関する記録](https://kubernetes.io/blog/2024/05/20/completing-cloud-provider-migration/) は、provider 固有 code を core に持ち続ける保守複雑性と vendor-neutrality を分離理由として挙げ、複数 provider へ同じ信頼を与える test framework の重要性も示す。

本設計では provider package 間 import を禁止し、共通 conformance suite、provider ごとの matrix、および onboarding fitness gate で同じ境界を検証する。

## 比較した案

### 一つの汎用 operation

`Execute(operation, map[string]any)` のような一つの入口へ集約する案は採用しない。

- compile time に必須項目と型を確認できない。
- 外部 field と provider 固有の検索文法が共通層へ漏れる。
- 一つの provider の仕様変更が共有 parser と分岐へ伝播する。
- 空結果、欠落、`not_found` および部分失敗の意味を operation 名だけでは固定できない。

### 一つの巨大な共通 model

多様な法情報を多数の optional field と `extensions` object へ入れる案は採用しない。

- 同じ名前で意味が異なる日付、状態および識別子を誤って同一視する。
- provider が持たない値を空値または推測値で埋める圧力が生じる。
- provider 追加のたびに共通 model を変更し、既存 consumer へ影響する。

### 能力別 typed port と provider 別 adapter

この案を採用する。

```text
新しい公開 tool ─┐
内部 use case ───┴─> use case ─> typed capability port
                                      ↑
                           provider-specific adapter
                           DTO / client / parser / mapper

既存 MCP facade ─> 互換入力境界 ─> 同じ adapter の client / parser / mapper
```

既存 facade も lossless に対応できる場合は typed capability port へ接続する。e-Gov 固有 DSL または任意の数値 `offset` のように共通契約へ持ち込めない入力は、provider 固有の互換入力境界に残し、応答 parser、mapping、資源予算およびエラー正規化だけを同じ adapter で共有する。

## 決定

1. 共通 interface は provider 名ではなく利用目的と意味で分けた typed capability とする。
2. capability は入力、出力、欠落、時点、継続取得、error および検証を能力別 SOT で固定する。
3. provider は descriptor、typed settings、credential slot、外部 operation mapping、resource budget、fixture および conformance row を独立して持つ。
4. provider 固有 DTO、検索 DSL、HTML selector、XML element および page key は adapter の外へ出さない。
5. `SourceResourceRef` と `Provenance` で provider、情報源、資源、版および変換を保持する。
6. 新 provider は意味が完全に一致する既存 capability だけへ binding する。一致しなければ別 capability とし、似た field 名だけで共通化しない。
7. 既存 MCP tool は互換 facade とし、内部 capability の provider/version roundtrip を公開契約へ暗黙に追加しない。
8. provider onboarding と built-in default の変更を分離し、追加しただけの provider が無設定時の挙動を変えないようにする。
9. provider ごとの package、SOT、fixture および conformance file を分け、他 provider の変更を onboarding gate で禁止する。
10. 利用者のローカル binary を前提とし、運用障害検知、`liveness`、`readiness` および常時監視を設計対象にしない。
11. 拡張パックの有効化、capability の意味、および provider route を独立した構成軸とし、パックの追加で法令コアの既定 route を変更しない。

採用した規定の定義元は、[SOT-ARCH-017](../../sot/30-architecture/17-approved-capability-families.md)、[SOT-ARCH-018](../../sot/30-architecture/18-pack-scoped-normalization-boundary.md)、[SOT-ARCH-019](../../sot/30-architecture/19-extension-pack-activation-boundary.md)、[SOT-ARCH-010](../../sot/30-architecture/10-provider-isolation.md)、[SOT-ARCH-016](../../sot/30-architecture/16-incremental-provider-onboarding.md)、[SOT-IF-014](../../sot/40-interfaces/14-provider-descriptor.md) から [SOT-IF-018](../../sot/40-interfaces/18-provider-configuration.md)、および [SOT-ENG-017](../../sot/50-engineering/17-provider-conformance-matrix.md) と [SOT-ENG-018](../../sot/50-engineering/18-provider-onboarding-fitness-gate.md) とする。

## 帰結

- e-Gov Version 2 は最初の実装であり、共通 model の無条件な同義語ではない。
- 同じ意味を提供する将来 provider は、既存 capability と test suite を変更せずに追加できる。
- HTML、PDF または source 固有の検索機能が意味的に一致しない場合は、既存 capability を膨らませず新しい小さい契約を先に採用する。
- adapter、mapping SOT、fixture および conformance row の追加費用は発生する。
- 公開 tool を増やす判断と、内部 provider binding を増やす判断は別になる。
- dynamic plugin が必要になった場合は、binary compatibility、署名、sandbox および distribution trust を別の ADR と SOT で採用する。
