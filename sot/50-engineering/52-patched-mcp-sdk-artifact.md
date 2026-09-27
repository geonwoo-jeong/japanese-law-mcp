# SOT-ENG-052: 修正した MCP SDK の固定成果物

- 状態: 有効

## 規定

製品が使用する MCP SDK の局所的な JSON 最適化は、固定した上流 module の複製と許可した差分だけからなる、検証可能な一つの成果物集合として管理する。元の JSON 動作を保持し、元版、全 source、変更内容、権利表示および公式配布物の対応を追跡できなければならない。

## 適用境界と配置

対象は `github.com/modelcontextprotocol/go-sdk` の固定版に対する内部 JSON 処理の最適化と、その修正版の提供である。MCP の公開契約、検索結果の意味、SDK の公開 API、他の外部 module の置換規則を変更しない。

成果物は次の配置とする。

```text
third_party/
├── modelcontextprotocol-go-sdk/
│   └── 固定した元 module の全 file と許可した差分
└── modelcontextprotocol-go-sdk-patch/
    ├── MCP-SDK-PATCH.json
    ├── json-unmarshal.patch
    └── MCP-SDK-LICENSE
```

SDK の複製には、元 module の全 file を含める。実装の変更は `internal/json/json.go` 一件、追加は `internal/json/json_equivalence_test.go` 一件に限定し、その他の元 file は原 byte を保持する。元の module 定義と原権利表示を変更しない。複製に含める file 集合は manifest から決まり、未登録 file、余分な directory、欠落、symlink、特殊 file および未申告の改変を拒否する。

root の `go.mod` は元 module の exact version を require し、その版だけを `./third_party/modelcontextprotocol-go-sdk` へ置換する。版を限定しない replace、他の配置への置換、置換先の別 version、require と replace の不一致を認めない。元版の取得物識別は後述する manifest と実取得物で照合し、root の module 定義と `go.sum` の通常の整合性は `SOT-ENG-020` に従って検証する。

## manifest と固定 identity

`MCP-SDK-PATCH.json` の schema version 1 は一つの JSON object とし、次の項目だけを持つ。

| 項目 | 内容 |
|---|---|
| `schemaVersion` | 整数 `1` |
| `patchId` | `sha256-` と `patchSHA256` の連結 |
| `module` | 後述する元 module identity |
| `sourceDirectory` | `third_party/modelcontextprotocol-go-sdk` |
| `patchFile` | `json-unmarshal.patch` |
| `patchSHA256` | patch 原 byte の SHA-256 |
| `reason` | 空でない日本語の変更理由 |
| `upstreamFiles` | 元 module の全 file の digest 一覧 |
| `changes` | 許可した実装変更一件の変更前後 identity |
| `additions` | 許可した同等性試験一件の digest |

`module` は `path`、`version`、`originCommit`、`zipSum`、`goModSum`、`zipSHA256`、`goModSHA256` だけを持つ。`path` は対象 module、`version` は製品が require する固定版、`originCommit` は元版の commit identity とする。`zipSum` と `goModSum` は元取得物の Go module `h1:` checksum、後二項目は元 ZIP とその中の `go.mod` の raw SHA-256 とする。元取得 metadata が commit identity を返す場合は `originCommit` と一致させる。

file digest は `path`、`bytes`、`sha256` だけを持ち、元 module または複製 root からの相対 path、原 byte 数、原 byte の SHA-256 を表す。`upstreamFiles` は path の byte 昇順とし、重複を認めない。`changes` は `path`、`beforeSHA256`、`afterSHA256`、`bytes` だけを持ち、許可した実装 file の元 digest、変更後 digest、変更後 byte 数を記録する。変更前 digest は `upstreamFiles` の同 file と一致させる。追加試験は元 file 集合と重複させない。

SHA-256 は小文字十六進六十四桁、commit identity は小文字十六進四十桁とする。file path は正規の相対 POSIX path とし、絶対 path、空の要素、`.`、`..`、逆斜線、NUL、改行および colon を認めない。未知項目、重複 key、null、不正 UTF-8、末尾の別 JSON 値、未対応 schema version と、固定 identity または許可差分と矛盾する値を拒否する。

manifest は 1 byte 以上 64 KiB 以下、元 ZIP は 1 byte 以上 32 MiB 以下、一 source file と patch は 4 MiB 以下、元 file 一覧は 1 件以上 1024 件以下とする。元の空 file は保持できる。変更後の実装 file は空にしない。JSON は深さ 6、値数 4096 を上限として読み込む。file の種類と上限を読取り前後に確認し、repository 内の検証済み path に閉じて照合する。

具体的な上流 version、commit、checksum、原 file 件数および全 file digest の値は、固定成果物と module 定義で管理し、SOT 本文へ一覧を複写しない。manifest は元取得物を修正版であるかのように記録せず、元版と修正後を明確に区別する。

## 内容の照合と同等性

repository 内の checker は、検証対象の module 選択が版限定の固定 copy を指すこと、明示した元 module 取得物が置換物ではないこと、および manifest の取得 identity との一致を確認する。一度読み込んだ同じ元 ZIP に対し、raw SHA-256、Go module checksum、entry 集合、全 file の byte 数と digest、および元 `go.mod` の両 checksum を照合する。不正 path、重複 entry、特殊 file および上限超過を拒否する。

複製の全 file 集合を、元 file 一覧に宣言された実装変更と追加試験を反映した集合と照合する。patch の原 byte が `patchSHA256` と一致し、配布用 LICENSE が元 ZIP の LICENSE と全 byte 一致することを確認する。patch は採用時に確認した実装変更を表し、採用する source とその許可差分を固定する。checker は照合対象を自動修正せず、取消し、取得不能、不正入力または不一致を失敗として返す。

JSON 最適化は、元版の値、大小文字の扱い、数値、重複項目、RawMessage、error の種類と内容、部分的な target 変更、custom unmarshal の呼出しと副作用、および入力 byte の所有権を保持する。stream 処理と、最適化対象外の入力の既存経路を変更しない。buffer 容量などの private な割当て詳細を公開契約にしない。

固定した元版の処理を独立の参照として、境界長、前後の空白、完全な一値、空・途中切断・複数値、長い数値、Unicode、大小文字、重複、入れ子、map/interface、RawMessage および custom unmarshal の正常・error・入力変更を照合する。製品の実際の MCP 経路でも、同じ入力に対する公開応答全体を比較する。品質と資源使用の確認では、同じ source と入力条件を識別して、変更前との同等性と割当て削減を確認する。

## 権利表示と公式 archive

元 SDK の LICENSE と各 source に含まれる原権利表示を原文のまま保持する。配布用の `MCP-SDK-LICENSE` は SDK 複製内の LICENSE および検証した元 ZIP の LICENSE と全 byte 一致させる。原文を翻訳やプロジェクト独自の説明へ置き換えない。プロジェクトが追加する由来と変更理由は日本語とし、原文と区別する。

公式デスクトップ archive が受理する entry 集合は、次の組合せだけとし、全てを archive 直下に含める。

- `SOT-DEL-010` が定める実行ファイル一つ。
- `SOT-ENG-051` が定める IPA の原文通知二つ。
- 本規定の `MCP-SDK-LICENSE` と `MCP-SDK-PATCH.json`。

SDK の二文書は実行権限のない通常 file とし、それぞれ 1 byte 以上 64 KiB 以下とする。SDK 許諾文は前述の原文、patch 識別情報は検証対象 checkout の検証済み manifest を正本とする。IPA の名称・原文・上限は `SOT-ENG-051` を定義元とする。実行ファイルの数、名前、配布形式と対象環境は `SOT-DEL-010` を維持する。

配布検証は、この閉じた集合にない entry、不正 path、重複、不正な file type、文書への実行権限、上限超過、欠落および正本との全 byte 不一致を拒否する。文書の正本も上限内の通常 file として読み、取得できない場合に照合を省略しない。実行ファイルの展開は entry の順序に依存せず、検証済みの名前で選択する。SDK の内容を IPA 原文へ混入させず、別の配布 asset に移してこの集合の照合を迂回しない。

## 確認

中央品質ゲートの `ci` profile は同じ snapshot に対して、root で `go run ./cmd/sdk-patch-check --repository=.` を実行し、SDK 複製内で `go test -p=1 -count=1 ./internal/json` を実行する。nested module の試験を root の全 package 試験だけで成功扱いしない。元取得物と採用済み複製の実照合を、合成 fixture の checker 試験だけで代替しない。

`SOT-ENG-020` の製品と固定検証 tool の脆弱性検査を維持し、local replace によって元 SDK の module identity が検査から欠落しないよう、固定した元 module と版を現在の脆弱性 DB に照会する。元版の照会対象・固定 tool・DB 応答と実行完了を確認し、元版の未解決の報告、外部到達不能または不完全な応答を成功にしない。元版に関する報告だけから修正済み製品の到達性を断定しない。

- manifest の閉じた構造、固定元・選択 module・許可差分の不一致と、複製の不足・追加・改変を拒否することを確認する。
- 元 ZIP の raw digest、module checksum、全 entry と全 file の identity、および LICENSE の原文一致を確認する。
- 既存と追加の SDK JSON 試験、実際の MCP 応答比較で、元版との同等性を確認する。
- archive の閉じた構成、文書の正本一致、欠落・未知追加・重複・不正 path/type/権限/size の拒否、および順序に依存しない実行ファイル選択を確認する。
- CI plan に実 copy 照合、SDK 専用試験、元 SDK の脆弱性照会が含まれ、ローカル hook に外部 DB 検査または全試験が追加されないことを確認する。

完了ゲートと検証段階は `SOT-ENG-020`、`SOT-ENG-027`、同一 source からのリリースは `SOT-DEL-004` に従う。選択した semantic source closure と過去成果物の不変性は `SOT-ENG-038` を定義元とし、本規定を理由にその対象 closure、schema または固定済み成果物を書き換えない。

## 関連

- [SOT-ENG-020](20-verification-gate.md)
- [SOT-ENG-027](27-resource-aware-verification-stages.md)
- [SOT-ENG-038](38-content-bound-candidate-evaluation-handoff.md)
- [SOT-ENG-051](51-compact-ipa-generated-artifact-v2.md)
- [SOT-DEL-004](../60-delivery/04-release-consistency.md)
- [SOT-DEL-010](../60-delivery/10-desktop-binaries.md)
