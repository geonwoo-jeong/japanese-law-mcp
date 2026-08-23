# SOT-IF-026: プロバイダールーティング設定

- 状態: 有効

## 規定

プロバイダーの有効化、認証参照および能力別ルーティングは、`SOT-IF-029` の実行設定とは独立した型付きの起動時設定として定義し、`providers` と `providerRoutes` の名前空間で扱う。

## 構造

起動時設定は、`SOT-IF-029` が定義する項目に加えて、次の最上位項目を持てる。

| 名前 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `providers` | object | いいえ | `providerId` ごとの有効化と認証参照 |
| `providerRoutes` | object | いいえ | `{capabilityId}@{majorVersion}` ごとの選択方法 |

`providers.{providerId}` は、次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `enabled` | boolean | はい | その `providerId` を起動時に利用可能にするか |
| `settings` | object | いいえ | プロバイダー SOT が型、既定値および検証を定義する非秘密の設定 |
| `credentialEnvRefs` | object | いいえ | プロバイダー SOT が定義した credential slot ごとの環境変数参照 |

`credentialEnvRefs.{slot}` は、次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `type` | string | はい | `env` |
| `name` | string | はい | 秘密値を読む実行環境の環境変数名 |

`providerRoutes.{capabilityId}@{majorVersion}` は、次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `selection` | string | はい | `explicit`、`primary` または `aggregate` |
| `defaultProviderId` | string | 条件付き | `selection` が `primary` の場合の既定プロバイダー |
| `aggregateProviderIds` | string[] | 条件付き | `selection` が `aggregate` の場合に順序付きで使用する `providerId` |
| `rollbackProviderId` | string | いいえ | `primary` route の既定プロバイダーを利用者が起動時に一時的に置き換える明示 override |

## 組込み既定値

設定ファイルに `providers` または `providerRoutes` がない場合は、次の組込み既定値を使用する。

```yaml
providers:
  e-gov-law-api-v1:
    enabled: true
    settings: {}
    credentialEnvRefs: {}
  e-gov-law-api-v2:
    enabled: true
    settings: {}
    credentialEnvRefs: {}
providerRoutes:
  law.article.read@1:
    selection: primary
    defaultProviderId: e-gov-law-api-v2
  law.content.search@1:
    selection: primary
    defaultProviderId: e-gov-law-api-v2
  law.document.read@1:
    selection: primary
    defaultProviderId: e-gov-law-api-v2
  law.revision.list@1:
    selection: primary
    defaultProviderId: e-gov-law-api-v2
  law.search@1:
    selection: primary
    defaultProviderId: e-gov-law-api-v2
  law.update.list@1:
    selection: primary
    defaultProviderId: e-gov-law-api-v1
```

この既定値の `providerId`、descriptor、credential slot、接続範囲および capability は `SOT-IF-004`、`SOT-IF-037` および `SOT-IF-057` を定義元とする。利用者設定がない通常起動では、この既定値だけで e-Gov Version 2 の五つの法令機能と `law.update.list@1` に必要な binding を構成できなければならない。

新しく追加した provider は、その provider の公開採用と組込み既定値の変更を定義する別の SOT が有効になるまで、この組込み `providers` または `providerRoutes` へ自動追加しない。追加しただけの provider は無設定時に有効化せず、既存の primary route を置き換えない。

`SOT-PROD-009` の選択型拡張パックに属する provider が既存 capability ID を再利用する場合、pack-aware な有効化方法または route 選択方法を定義する後継 SOT が有効になるまで、その provider を既存の組込み `providerRoutes.{capabilityId}@{majorVersion}` に参加させてはならない。無設定起動では法令コアの既定 route を保持し、拡張 provider は独立した公開機能、`selection: explicit` の route、または provider-specific capability を採用したときにだけ到達可能にする。

## 制約

`providerId` は `SOT-IF-014` に従う。`providers` に存在しない `providerId`、コンパイル時に登録されていない `providerId`、または同じ `providerId` への重複定義を許可しない。

`settings` の key、型、既定値、上限、組合せ、接続先の変更可否および provider configuration scope に含める key は、プロバイダー固有のインターフェース SOT が定義する。共通設定境界は登録済み provider の設定 schema で検証してから、provider-specific の型付き設定へ変換する。未知の key、無型の値およびプロバイダー SOT が定義しない組合せを許可しない。

`settings` に秘密値、credential、署名鍵、session、利用者入力または外部レスポンスを含めない。秘密値は `credentialEnvRefs` が示す環境変数からだけ解決する。

`credentialEnvRefs` の `slot` は、そのプロバイダーのインターフェース SOT が定義した credential slot だけを許可する。未知の slot、未知の `type`、空の `name` および同一 slot への複数参照は起動時設定エラーとする。

`credentialEnvRefs.{slot}.name` は 1 文字以上 128 byte 以下で、ASCII の英字または `_` で始まり、その後を ASCII の英数字または `_` だけで構成する。参照先の環境変数が存在しない場合または値が空の場合は、provider の必須性に応じて起動時の設定エラーまたは `configuration_required` とし、空の credential をアダプターへ渡さない。

`credentialEnvRefs` は秘密値そのものを保持しない参照だけを表す。秘密値を設定ファイル、コマンドライン引数、`providerRoutes`、継続トークン、`ProviderDescriptor`、エラー、ログまたは診断へ含めない。解決した秘密値は起動後のメモリー内だけで保持し、設定として再シリアライズしない。

`selection` が `explicit` の route は、対応するツールまたはユースケースが provider 指定入力を別の SOT で定義している場合にだけ使用する。`defaultProviderId`、`aggregateProviderIds` または `rollbackProviderId` を併用しない。

`selection` が `primary` の route は、`defaultProviderId` を必須とする。`defaultProviderId` と `rollbackProviderId` は、どちらも同じ `{capabilityId, majorVersion}` を実装し、`enabled: true` である `providerId` でなければならない。`rollbackProviderId` を指定した場合は、運用者が明示的に rollback を選択したものとして、その route の実効既定プロバイダーを `rollbackProviderId` に置き換える。これは起動時 override であり、実行時の暗黙 fallback ではない。

`selection` が `aggregate` の route は、`aggregateProviderIds` を必須とし、配列は空を許可せず、順序を保持し、重複を許可しない。配列内の各 `providerId` は同じ `{capabilityId, majorVersion}` を実装し、`enabled: true` でなければならない。

集約検索の入力、部分結果、情報源別エラー、情報源別継続位置、決定的な順序および再開条件を定義する利用シナリオ、結果モデルおよび公開インターフェース SOT が採用されるまでは、`selection: aggregate` の route を定義してはならない。該当する後継 SOT がない状態で `aggregate` を指定した場合は起動時設定エラーとする。

`providerRoutes` の key は `{capabilityId}@{majorVersion}` の形式だけを許可し、`capabilityId` は `SOT-MODEL-013`、`majorVersion` は 1 以上の整数に従う。同じ key の重複、宣言されていない capability、実装していない major version、未登録 provider を参照する route、および route と provider 実装の不一致は起動時設定エラーとする。

既存の公開機能に必要な `primary` route が欠落している、または実効既定プロバイダーが構成できない場合は起動を失敗させる。公開機能へ未採用の capability は、route を定義しても MCP ツールの追加を意味しない。

未知の最上位項目、未知の `providers.{providerId}` 項目、未知の `providerRoutes` 項目、型の不一致および成立しない組合せは、`SOT-IF-039` と同じくサーバー起動前の設定エラーとする。

## 優先順位

`providers` と `providerRoutes` の入力元は、選択された一つの設定ファイルと、この文書の組込み既定値だけとする。個別のコマンドラインフラグまたは設定構造を表す環境変数を定義しない。設定ファイルに名前空間自体がない場合は、その名前空間の組込み既定値を使用する。

秘密値そのものはコマンドラインフラグまたは設定ファイルで受け取らない。設定ファイルで指定できるのは `enabled`、`settings`、route 構造および `credentialEnvRefs` の参照名だけとする。実際の秘密値は `credentialEnvRefs` が指す環境変数から起動時に読み取る。

設定ファイルに名前空間がある場合は、`providers.{providerId}` と `providerRoutes.{capabilityId}@{majorVersion}` の key ごとに組込み既定値へ上書きまたは追加する。provider object、route object および配列は全体を atomic に採用し、組込み既定値の子項目と部分的に結合しない。明示した空の `providers` または `providerRoutes` は、その名前空間の組込み既定値をすべて削除する。

## 継続取得の設定 scope

継続トークンに使用する provider configuration scope の fingerprint は、結果または取得位置の意味を変え得る有効設定を JSON object として表し、RFC 8785 の JSON Canonicalization Scheme で UTF-8 byte 列にした後、継続トークン鍵を使う HMAC-SHA-256 で生成する。HMAC 入力は ASCII の `provider-config-scope-v1`、値 `0x00`、canonical JSON の順に連結し、出力は padding なしの base64url とする。

fingerprint の object は、`providerId`、`origin`、`dataset`、`tenant`、`account`、`proxy`、`semanticConfig` および `credentialSlots` の八つの key を必須とする。該当しない scalar は文字列 `n/a`、該当しない object は空の object とし、`null` と key の省略を許可しない。

各 `credentialSlots.{slot}` は、秘密値を UTF-8 byte 列として、同じ継続トークン鍵を使う HMAC-SHA-256 で生成する。HMAC 入力は ASCII の `credential-scope-v1`、`0x00`、`providerId`、`0x00`、`slot`、`0x00`、秘密値の順に連結し、出力は padding なしの base64url とする。`credentialSlots` の key は provider SOT が定義した slot 名とし、秘密値を持たない provider は空の object とする。

`semanticConfig` には、結果または取得位置の意味を変える provider 固有設定だけを入れる。timeout、diagnostics、transport、製品版その他の非意味的な実行設定を含めない。プロバイダー SOT は、この八項目の具体的な値と `semanticConfig` の key を定義する。

秘密値、環境変数名、origin の認証情報、proxy credential または fingerprint の入力を、継続トークン、ログ、エラーまたは診断へ含めない。実効設定または credential が変わった場合は fingerprint が変わり、変更前の継続トークンを `invalid_argument` として拒否する。

## 関連

- [SOT-IF-029: ローカル実行設定](29-local-runtime-configuration.md)
- [SOT-IF-018: プロバイダー設定境界](18-provider-configuration.md)
- [SOT-IF-039: 設定ソースと優先順位 v2](39-configuration-sources-and-precedence-v2.md)
- [SOT-IF-016: 情報源の継続取得](16-source-continuation-contract.md)
- [SOT-ARCH-012: プロバイダーの登録](../30-architecture/12-provider-registry.md)
- [SOT-ARCH-013: 情報源の選択と組合せ](../30-architecture/13-source-composition.md)
- [SOT-ARCH-019: 拡張パックの有効化境界](../30-architecture/19-extension-pack-activation-boundary.md)
