# SOT-IF-018: プロバイダー設定境界

- 状態: 有効

## 規定

各プロバイダーアダプターは、自分に必要な接続設定と認証情報だけを持つ型付き設定を受け取り、他のプロバイダーの設定または共通情報モデルから分離する。

## 設定

プロバイダー設定はプロバイダー ID の名前空間で分け、接続先の変更可否、認証情報の種類、必須条件および検証方法を、採用するプロバイダーごとのインターフェース SOT で定義する。

認証情報は、実行環境の環境変数または秘密情報の注入機構から起動時に読み取り、該当アダプターへだけ渡す。MCP リクエスト、能力別入力、`ProviderDescriptor`、`InformationSource`、継続トークン、エラーまたは診断へ含めない。

設定値は起動後に変更しない。未知の設定、異なるプロバイダーの設定混在、無効な URL、必要な認証情報の欠落および許可しない組合せは、外部呼出し前に判別する。

外部接続先を設定可能にする場合は、プロバイダー SOT が許可する HTTPS origin の範囲に限定し、任意の URL をリクエストから指定させない。

HTTP クライアントは、許可した origin 以外への redirect を追従しない。名前解決後の接続先が loopback、private、link-local または multicast のアドレスである場合と、IP literal または Unix domain socket を接続先として指定した場合は拒否する。接続時にも解決先を検証し、検証後の名前解決差替えを許さない。

プロバイダー HTTP クライアントは、`HTTP_PROXY`、`HTTPS_PROXY`、`NO_PROXY` その他の実行環境が暗黙に与える proxy 設定を既定で使用しない。proxy を許可する場合は、そのプロバイダーのインターフェース SOT で採用条件、接続先、認証方式および検証方法を定義し、明示設定で有効化したときだけ使用する。

許可した proxy を経由する場合も、最終的に到達する origin、名前解決後のアドレス、redirect および認証情報の到達範囲へ、direct 接続と同じ検証を適用する。別の origin、別の利用主体または別の認証スコープへ認証情報を転送しない。

プロバイダーの有効化、credential の環境変数参照および能力別 route は `SOT-IF-026` に従う。各プロバイダーを有効にする前に、そのプロバイダー固有の接続先、credential slot、設定 scope fingerprint の入力および検証方法をインターフェース SOT で定義する。

## 関連

- [SOT-IF-029: ローカル実行設定](29-local-runtime-configuration.md)
- [SOT-IF-039: 設定ソースと優先順位 v2](39-configuration-sources-and-precedence-v2.md)
- [SOT-IF-026: プロバイダールーティング設定](26-provider-routing-configuration.md)
- [SOT-IF-014: ProviderDescriptor](14-provider-descriptor.md)
- [SOT-ARCH-012: プロバイダーの登録](../30-architecture/12-provider-registry.md)
- [SOT-ARCH-015: 起動時設定境界](../30-architecture/15-startup-configuration-boundary.md)
