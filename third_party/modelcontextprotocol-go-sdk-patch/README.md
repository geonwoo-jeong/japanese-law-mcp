# Go MCP SDK の固定複製と JSON 差分

`../modelcontextprotocol-go-sdk/` は `github.com/modelcontextprotocol/go-sdk@v1.6.1` の module ZIP に含まれる200 file を保存した複製である。元の module path、go.mod、go.sum、copyright、完全な LICENSE を保持する。これは無変更の v1.6.1 ではなく、[MCP-SDK-PATCH.json](MCP-SDK-PATCH.json) の差分を含む。root の version 限定 local replace だけがこの複製を選択する。

元の ZIP/go.mod の h1、raw SHA-256、取得元 commit、元200 file のサイズと hash、変更前後の hash、追加試験の hash、patch 識別を manifest に固定した。変更は `internal/json/json.go` 一つと `internal/json/json_equivalence_test.go` の追加だけである。[json-unmarshal.patch](json-unmarshal.patch) は実装差分を保持する。SDK 側の英文原文は第三者原文として保存し、この説明と追加差分コメントは日本語で記述する。

## 修正内容

元 SDK の内部 Unmarshal は短い完全な入力にも segmentio Decoder の32KiB buffer を確保していた。32KiB未満かつ妥当な完全JSON一値だけ、JSON四空白を除いた raw token を複製して Parse へ渡す。大小文字を厳密に扱う flag は維持する。その他の入力は元 Decoder を通る。末尾空白の error 内容、custom callback から見た入力所有権、数値境界、多値・不正入力の動作を差分試験で照合する。

実装の SHA-256 は `c3000b84e6507c84c1a56ed5edb76c15d0104dd64067417b4cc9e637bc96572a`、追加試験は `4c91435f71125383afe06bfe3b250c232341a39ce0c49fa5a7e198d4be536af2`。測定済み file の全 byte を保存しており、実装内へ配布用の追記は行っていない。差分識別は隣接文書で示す。

## 照合と保守

成果物と配布file集合は SOT-ENG-052、適用検証は SOT-ENG-019/020/027、依存方向は SOT-ARCH-007、配布は SOT-DEL-004/010/011 を定義元とする。開発用 `go run ./cmd/sdk-patch-check --repository=.` は元 module の exact version を明示して取得情報を読み、ZIP の raw hash/h1 と全 file 一覧、version 限定 replace、保存copy、許可差分、追加試験、配布用 LICENSE の byte 一致を検査する。元ZIPを原hash/h1で識別し、全200 fileを原一覧と照合し、JSON実装の変更前hashが原一覧と一致することを確認する。保存後hashはmanifestの許可値に一致しなければならず、他199 fileは原値を維持する。追加fileも固定試験一つとそのhashに限定する。この比較で変更範囲を固定する。unified patchはraw hashを照合する識別補助であり、checkerがpatchを自動再適用して意味や実行結果を証明するものではない。JSON動作の同値性は差分試験と公開契約で検証する。

対象repositoryのGo照会はworkspace・Go設定を切り離したreadonly環境で行う。明示版の `go mod download` はgo.sumを更新し得るため、一時module内で取得し終了時に破棄する。copy、対象repositoryのgo.mod/go.sum、manifest、既存module cacheをpatchする処理はない。元ZIPが未取得の場合の通常cacheへの取得可否は呼出し側のGOPROXY等に従う。CI以外の既存Git hookに取得・脆弱性照会を追加しない。

root の `go test ./...` はこの入れ子 module の試験を列挙しない。中央 CI 品質ゲートから SDK directory を cwd に `go test -p=1 -count=1 ./internal/json` を一回実行する。root 製品の契約・coverage・source mode の govulncheck は維持する。local replace の空版が元 SDK の脆弱性照会を欠落させるため、固定 govulncheck の query mode で元path/版を別途照会する。プロセス終了だけで成功にせず、固定protocol、対象identity、DB取得、完全JSON終端を照合し、OSVが返れば到達性またはSDK更新の確認が必要として止める。

SDKを更新するときは元200 file の一部を手直しせず、選択する原版、元取得identity、copy、差分、試験、manifest、限定replace、脆弱性照会のidentity、配布通知を同じ変更で照合する。適用patchとmanifestのhashだけを書き換えて検証済みと扱わない。root go.sum は通常の tidy 結果を使い、元SDKの取得identityは本manifestで保持する。

## 許諾文と配布識別

[MCP-SDK-LICENSE](MCP-SDK-LICENSE) は SDK の LICENSE と同一 byte であり、MITからApache-2.0への移行説明と両許諾原文を省略しない。公式binary archiveには本許諾文と `MCP-SDK-PATCH.json` を同梱する。元版、patch digest、releaseのrepository commitは別々の識別である。local replacement の build info だけを元module checksumの証明としない。
