# SOT-IF-015: 情報源操作の共通契約

- 状態: 有効

## 規定

能力別情報源ポートの各操作は、能力 SOT が定義する型付き入力と型付き出力を使用し、共通の呼出し生存期間、出典および失敗の契約に従う。

## 呼出し

- 操作は `context` を受け取り、キャンセルと期限を外部呼出しおよび解析へ伝播する。
- 入力値を変更せず、ページ取得、再試行またはプロバイダー変換に必要な値は新しい値として作る。
- プロバイダーが宣言していない能力は、ネットワーク呼出し前に `unsupported_capability` とする。
- 外部呼出しは、プロバイダー SOT が定める固定間隔、同時実行上限、`Retry-After` および backoff に従う。
- 外部リクエスト型、レスポンス型、エラー型および parser の型を、ポートの引数または戻り値に使用しない。
- 一つの資源を返す能力は `SourcedResource<T>` を内部結果とする。
- 一覧または検索を返す能力は、`SourcedResource<T>[]` と `SourcePage` を内部結果とする。

`SourcedResource<T>` は、次の従属構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `ref` | `SourceResourceRef` | はい | 取得に使用したプロバイダーと情報源上の資源と版 |
| `provenance` | `Provenance[]` | はい | 取得、抽出、正規化または加工の経路 |
| `data` | `T` | はい | 能力 SOT が定義する一つの具体的な型付き情報モデル |

`T` を無型の辞書、外部レスポンス型またはプロバイダー固有 DTO にしない。

`provenance` は一件以上とし、各要素の `resourceKey.sourceId` は、別の入力資源を明示する `derived` の `inputKeys` を除き `ref.key.sourceId` と一致する。最後の要素は `data` を生成した変換を表し、その `resourceKey` は `ref.key` と一致する。

`ref` は `data` の根拠となる主資源を示し、一覧または検索結果の item ID を兼ねるとは限らない。item identity は能力別 SOT を定義元とし、共通処理は `ref` だけを使って item を重複排除、上書きまたは結合しない。

この共通契約は情報取得だけを対象とする。外部システムへの登録、変更、削除、提出または処理開始は、利用シナリオと副作用を別の SOT で採用するまで共通能力へ含めない。

この共通契約は情報源ポートとアプリケーションの内部境界に適用し、既存 MCP ツールの公開モデルを変更しない。既存法令ツールへの変換は、現在の法令モデルと mapping SOT に従う。

個別の能力を実装する前に、その能力の ID、メジャーバージョン、利用目的、入力型、出力型、欠落時の扱い、継続取得、エラーおよび検証方法を一つ以上の能力別 SOT で定義する。この共通契約または能力群の一覧だけから個別の入出力を実装しない。

## 欠落

検索結果がない場合は、能力別の空の結果として返す。正確な識別子による取得対象がない場合は `not_found` とする。

外部情報源に存在しない項目を既定値、別の項目または推測値で補わない。省略可能な項目として能力別モデルが許可する場合だけ省略する。

## 関連

- [SOT-ARCH-017: 採用可能な能力群](../30-architecture/17-approved-capability-families.md)
- [SOT-ARCH-018: 拡張パック単位の正規化境界](../30-architecture/18-pack-scoped-normalization-boundary.md)
- [SOT-MODEL-011: SourceResourceKey](../20-model/11-source-resource-key.md)
- [SOT-MODEL-012: Provenance](../20-model/12-provenance.md)
- [SOT-MODEL-014: SourcePage](../20-model/14-source-page.md)
- [SOT-MODEL-016: SourceResourceRef](../20-model/16-source-resource-ref.md)
- [SOT-ENG-010: コンテキストによるキャンセル](../50-engineering/10-context-cancellation.md)
