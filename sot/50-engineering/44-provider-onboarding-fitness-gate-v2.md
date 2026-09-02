# SOT-ENG-044: プロバイダー追加 fitness gate v2

- 状態: 有効

## 規定

新しい provider または既存 provider の capability binding は、他の provider と共通 interface への不要な変更を伴わず、独立した package、fixture および test として追加できることを `provider-onboarding-fit` で確認する。

この gate は provider 実装の内部構造を完全に証明するものではない。変更範囲の分離、matrix との対応、共通契約 test および既存 test の非回帰を確認する。provider の機能変更と、廃止 SOT から直接の後継 SOT への追跡参照だけを更新する変更は区別する。

## 適用変更

merge base との差分に次のいずれかが含まれる場合に適用する。

- provider-specific SOT、provider package、fixture または `ProviderDescriptor`
- capability binding、provider route、provider 設定 schema または conformance matrix
- provider の追加または変更に伴う composition root の登録
- conformance matrix の `interfaceSotIds` にある廃止 SOT を直接の後継 SOT へ更新する追跡参照変更

共通 capability、共通 model または能力別 port の意味を変更する変更は、provider の追加と分離する。ただし、最初の provider を実装する前に必要な provider-neutral の registry、route、HTTP、continuation、予算または conformance 基盤は、独立した基盤変更として先に追加できる。

## 検査スナップショット

`--base-ref` と検査対象 `HEAD` の merge base を比較開始点とする。commit 差分、index、working tree および未追跡 file の和集合を検査し、選択した各層の最終内容を使用する。同じ path の index と working tree の内容が異なる場合、VCS 情報を取得できない場合、比較開始点または検査対象を commit として解決できない場合、および内容の由来を一意に決められない場合は成功として扱わない。

Git が返した object ID と repository 相対 path を検証し、shell を介さず固定した Git command の引数として使用する。repository 外の path、symlink または通常 file ではない canonical artifact を読み込まない。

## matrix の追跡参照更新

変更した provider matrix ごとに、merge base の内容と検査スナップショットの内容を `SOT-ENG-017` の同じ canonical schema、strict YAML decoder および row 検証で読み込む。次の条件をすべて満たす場合だけ、その matrix を追跡参照だけの更新として provider の機能変更 target から除外する。

1. 両方の matrix が同じ `schemaVersion` を持ち、schema、provider file 名、row の整列、case partition および重複禁止に適合する。
2. row の数、順序、および `(providerId, capabilityId, majorVersion, operation)` が一致する。
3. 各 row の `interfaceSotIds` 以外の全 field が一致する。
4. 各 `interfaceSotIds` の長さと順序が一致し、同じ位置で一つ以上の ID が変更されている。
5. 変更前の各 ID は現在の SOT で `廃止` であり、同じ文書の `後継` が変更後の ID を直接指す。
6. 変更後の各 ID は現在の SOT で `有効` である。

前記の条件を満たす複数 provider の matrix は、一つの変更に含めてよい。対象 provider 数の算定からは除外するが、追跡参照更新も gate の適用変更とし、canonical matrix loader、provider import 検証および全 `implemented` row の conformance test を実行する。

file または row の追加・削除、row の並べ替え、`interfaceSotIds` の追加・削除・並べ替え、直接の後継ではない ID への変更、他 field の変更、schema または YAML の不正、重複 key、重複 tuple、および比較元を読み取れない変更は、追跡参照だけの更新として除外しない。検証不能な内容を追跡参照更新として成功へ緩和しない。

追跡参照だけの matrix 更新と、provider package、provider fixture、provider descriptor、binding、route、provider 設定 schema、共通 model、共通 capability または provider conformance 基盤の変更を同じ変更へ混在させない。混在した場合は、provider の機能変更または基盤変更から追跡参照更新を分離する。

provider 制御変更の path 判定は、descriptor、binding、route、provider 設定 schema または composition root の責任を持つ artifact に限定する。provider、route または binding という文字列を file 名に含むだけの共通 loader、test または一般設定 artifact を、内容を確認せず provider 制御変更とみなさない。

## 実行

共通 command は次とする。

```text
go run ./cmd/provider-onboarding-fit --base-ref <git-revision>
```

`--base-ref` は一回だけ必須とし、commit として解決できる値を受け付ける。CI はこの command を中央の品質ゲートより先に実行し、`SOT-ENG-017` の canonical matrix loader と通常の Go test を再利用する。ローカルの Git hook はこの command または provider conformance test を実行しない。開発者が問題を切り分ける場合に限り、対象を限定して command または回帰 test を任意に実行できる。

## 検証

gate は次を確認する。

1. 変更対象外の provider package と provider 固有 fixture に意図しない差分がない。
2. provider 追加だけを目的とする変更では、既存の共通 model、能力別 port および共通 capability SOT の意味を変更していない。
3. provider package が他の provider package を import していない。
4. provider の外部 DTO、request builder、parser および mapping がその provider package 内に分離されている。
5. `implemented` row、production 公開する `ProviderDescriptor`、compiled binding inventory、fixture および test の対象が一致し、runtime registry と route は enabled かつ implemented である binding だけを参照する。`planned` row の test 用 descriptor、fixture および test は先に置けるが、production descriptor、compiled binding inventory、runtime registry または route へ含めない。
6. 新しい binding が同じ `(capabilityId, majorVersion)` の共通 conformance suite に合格する。
7. 変更前から存在する provider の unit・integration・conformance test が合格する。
8. 新しい provider または既存 provider の新しい binding は、別の有効な SOT で公開採用を決めない限り、組込み provider、無設定時の enabled set、既存 route の key と値、および primary route を変更しない。disabled provider では factory の呼出しと credential の解決を行わない。
9. 追跡参照だけの matrix 更新では、変更した全 ID が現在の SOT lifecycle と直接の後継関係に一致する。

検証には schema、import、package path、matrix snapshot、SOT lifecycle および matrix reference の静的確認と、通常の Go test を使用する。provider factory の関数 object、全 SSA dataflow、全 call graph、fixture の内部 counter または `go test -json` event の完全一致は必須としない。

## 段階的な実装

実装は次の順序に分けてよい。

1. canonical schema、matrix loader、fitness command および CI 接続
2. provider-neutral の registry、route、共通 helper および能力別 conformance suite
3. 一つの capability binding の provider package、fixture および test
4. 同じ provider の次の capability binding
5. provider 全体の descriptor の production 公開、matrix status、runtime registration および route の有効化

各 provider binding は一つずつ実装、review、commit する。準備中の row は `status=planned` のままにし、provider package と test が存在しても runtime から到達させない。

一つの変更単位は一 provider を原則とする。ただし、別の有効な SOT が複数 provider の prepared row、runtime binding および公開面を同じ最終接続で原子的に有効化することを明示した場合に限り、その SOT が列挙した provider の完全一致集合を一つの従属変更単位として検査できる。この例外では、対象 provider すべての matrix file を同じ差分に含め、provider package の分離、provider 間 import の禁止、共通 model と capability の非変更、および全 implemented row の conformance test を維持する。matrix を変更しない複数 provider package の同時 maintenance、列挙外 provider の追加、または一部 provider だけの組合せには適用しない。

`SOT-IF-074` が採用した `courts-hanrei-html` と `courts-hanrei-pdf` の最終接続は、この従属変更単位に該当する。これは provider identity、設定、descriptor、package、fixture または資源予算を統合する規定ではない。

planned の準備段階では test 用の descriptor 定義を置けるが、production へ公開しない。一つの `ProviderDescriptor` が複数 capability を宣言する場合は、各 binding を順に準備した後、descriptor の capability 集合と compiled binding inventory が一致する単位でまとめて有効化できる。e-Gov 法令 API Version 2 は、六つの binding と descriptor を一致させて `implemented` とする。

既存 provider に capability を追加する場合は、対象の planned row と新しい成果物だけを段階的に変更する。別の有効な SOT が同時変更を採用しない限り、既存の implemented または retired row、既存 descriptor capability、runtime binding および route を維持する。

既存 provider の maintenance は、対象 provider package と fixture の範囲に限定し、既存の共通 conformance suite を再実行する。外部仕様の更新で provider SOT、descriptor または複数 binding が同時に影響を受ける場合は、provider 単位の契約更新として扱ってよい。

## 初回導入

repository に canonical schema、matrix loader または `provider-onboarding-fit` がまだない場合は、次を一つの初回導入変更として追加できる。

- canonical schema
- 全 row が `status: planned` である最初の provider matrix
- matrix loader と test
- `provider-onboarding-fit` command と test
- CI から command を呼ぶ接続
- loader と command に必要な最小の module dependency
- 実装状況 Wiki の更新

初回導入には実際の provider adapter、provider fixture、runtime binding または public route を含めない。初回導入後は、前節の段階に従って一つずつ追加する。

## 成功条件

前記の九条件または適用可能な `SOT-ENG-020` の gate が失敗した場合は、provider 変更を完了としない。SOT と matrix だけを準備する変更では、SOT 静的検査と schema test を行い、provider 実装を開始した時点から conformance test を必須にする。

## 関連

- [SOT-ARCH-016: プロバイダーの段階的な追加](../30-architecture/16-incremental-provider-onboarding.md)
- [SOT-ENG-020: 変更の検証ゲート](20-verification-gate.md)
- [SOT-ENG-013: プロバイダー契約の検証](13-provider-contract-verification.md)
- [SOT-ENG-017: プロバイダー適合性 matrix](17-provider-conformance-matrix.md)
