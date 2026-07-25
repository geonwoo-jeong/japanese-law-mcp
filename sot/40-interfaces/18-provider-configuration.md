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

既存の実行設定へ公開する名前、環境変数および配布方式は、各プロバイダー機能を採用するときに `SOT-IF-005` と整合する別の設定 SOT で定義する。`SOT-IF-020` が許可する最上位項目にプロバイダー設定は含まれないため、認証または個別設定を必要とするプロバイダーを有効にする前に、設定項目と優先順位を定義する後継 SOT を採用する。

## 関連

- [SOT-IF-005: 実行設定](05-runtime-configuration.md)
- [SOT-IF-020: 設定ソースと優先順位](20-configuration-sources-and-precedence.md)
- [SOT-IF-014: ProviderDescriptor](14-provider-descriptor.md)
- [SOT-ARCH-012: プロバイダーの登録](../30-architecture/12-provider-registry.md)
- [SOT-ARCH-015: 起動時設定境界](../30-architecture/15-startup-configuration-boundary.md)
