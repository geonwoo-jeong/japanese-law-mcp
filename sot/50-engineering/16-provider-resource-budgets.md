# SOT-ENG-016: プロバイダー資源予算

- 状態: 有効

## 規定

各プロバイダー operation は、実装、fixture および契約テストを追加する前に、レスポンス取得、展開および解析で使用できる資源予算を数値で定義し、共通の安全上限を超えない範囲で provider-specific の lower budget を持つ。

## 必須予算

各 operation を採用する能力別または mapping SOT は、その operation が扱う artifact 種別ごとに、次の項目を整数または duration で定義する。

- `budgetKey`: provider 内で一意かつ変更しない小文字 ASCII の kebab-case 識別子
- `responseBytes`: HTTP の transfer framing を除いた後、`Content-Encoding` を復号する前に受信を許可する response body の最大 bytes
- `decompressedBytes`: HTTP content coding、圧縮 stream および container の各復号または展開段階が生成する bytes の累計上限。復号または展開がない場合は parser へ渡す body bytes の上限
- `entriesOrObjects`: artifact 種別ごとに後述する構造単位の上限
- `depth`: artifact 種別ごとに後述する入れ子または参照深さの上限
- `parseTimeout`: 取得済みレスポンスの展開と解析に使える最大時間
- `concurrencyGroup`: 同じ provider 内で同時実行枠を共有する小文字 ASCII の kebab-case 識別子
- `concurrency`: 一つの process 内で、その `providerId + concurrencyGroup` を同時に実行してよい件数

provider-specific の予算は、次の絶対 ceiling 以下とする。絶対 ceiling と同じ値を採用する場合は、その operation の SOT に lower budget をこれ以上下げられない理由を明記する。

| artifact 種別 | `responseBytes` ceiling | `decompressedBytes` ceiling | `entriesOrObjects` ceiling | `depth` ceiling | `parseTimeout` ceiling |
|---|---:|---:|---:|---:|---:|
| JSON / XML / HTML / GeoJSON / XBRL | 16 MiB | 32 MiB | 500000 | 128 | 5s |
| gzip 単体ストリーム | 16 MiB | 64 MiB | 500000 | 128 | 5s |
| ZIP アーカイブ | 32 MiB | 128 MiB | 1024 | 16 | 10s |
| PDF | 32 MiB | 32 MiB | 200000 | 64 | 10s |
| PBF または他のバイナリー record container | 32 MiB | 64 MiB | 1000000 | 64 | 10s |

## 計測規則

すべての byte 数は octet 数とし、ceiling と同じ値までは受理し、一 byte でも超えた時点で処理を中止する。`Content-Length` だけを信頼せず、stream を読みながら実測する。HTTP header、chunk framing および trailer は `responseBytes` に含めない。

`decompressedBytes` は、一つの operation 内で復号または展開の段階ごとに生成した byte 数を加算し、段階の切替えで 0 に戻さない。例えば HTTP gzip で送られた ZIP は、gzip が生成した ZIP bytes と ZIP の各 entry が生成した bytes の両方を累計する。復号も展開もない JSON、XML、HTML、PDF または binary は、parser に渡す body bytes を一回だけ数える。

`entriesOrObjects` は次のように数える。

- JSON と GeoJSON は、root value を一つとし、object の各 member value と array の各 element を再帰的に一つずつ数える。member name は別に数えない。scalar と container はともに一 value とする。
- XML、HTML および XBRL は、各 element、attribute、namespace declaration、空でない text または CDATA node、comment および processing instruction を一つずつ数える。document node は数えない。DTD と entity declaration は `SOT-ARCH-014` により受理しない。
- ZIP は、展開対象の選別より前に central directory の directory を含む全 entry を一つずつ数える。central directory を安全に確定できない archive は `unsafe_source_content` とする。
- PDF は、object stream 内の object を含む各 indirect object を object number と generation number の組ごとに一つ数える。同じ object への複数参照は重複して数えない。
- PBF または他の binary record container は、schema が定義する各 top-level record と再帰的な nested message を一つずつ数える。framing と record 境界は provider operation の SOT で固定する。
- 構造を持たない byte stream は全体を一単位とし、`entriesOrObjects: 1`、`depth: 1` とする。

`depth` は次のように数える。

- JSON と GeoJSON は root value を 1 とし、member value または array element へ一段下りるごとに 1 を加える。
- XML、HTML および XBRL は root element を 1 とし、子 element へ一段下りるごとに 1 を加える。attribute とその他の node は深さを増やさない。
- ZIP は、separator を `/` に正規化して安全な相対 path であることを検証した後の path component 数とする。
- PDF は trailer または catalog の root から indirect reference を辿る非循環 path の最大 edge 数に 1 を加えた値とする。循環参照は訪問済み集合で停止し、循環そのものが parser の安全性を損なう場合は `unsafe_source_content` とする。
- PBF または他の binary record container は top-level record を 1 とし、nested message へ一段下りるごとに 1 を加える。

artifact を複合する場合は、各段階に該当する数え方を適用し、operation SOT に artifact の順序と最終 parser 種別を記載する。実装ライブラリー固有の node 数、memory allocation 数または callback 回数へ置き換えない。

`concurrency` の絶対 ceiling は、一つの `providerId + concurrencyGroup` について一つの process あたり 4 件とする。外部情報源の制限、CPU 使用量またはメモリー使用量に応じて、これより小さい値を定義する。

同じ provider の複数 operation が同じ外部同時実行上限または同じ重い parser 枠を共有する場合は、各 budget row に同じ `concurrencyGroup` と同じ `concurrency` を設定する。独立した上限を持つ operation は異なる group を使用する。同じ group で異なる `concurrency` を定義してはならない。

## 超過時の扱い

取得中または取得後に予算を超えた場合は、実行を継続しない。

- `responseBytes` または `decompressedBytes` の超過は `source_response_too_large` とする。
- `entriesOrObjects` または `depth` の超過は、構造上の危険が原因であれば `unsafe_source_content`、単純なサイズ超過として検出した場合は `source_response_too_large` とする。
- `parseTimeout` の超過は `source_processing_limit` とし、同じ取得済み内容を自動再試行しない。
- `concurrency` の超過は、外部呼出しまたは解析を開始せず `source_busy` とする。判定には現在実行中の同じ `providerId + concurrencyGroup` だけを数え、完了済み呼出しの履歴を使用しない。

同時実行枠は最初の外部呼出し前に一回取得し、自動再試行、レスポンス取得、展開、解析および mapping が終了するまで保持してから必ず解放する。キャンセル、timeout、panic を含むすべての終了経路で解放し、一つのリクエストの再試行ごとに枠を取り直さない。

エラー、診断、ログおよび test failure の出力に、外部本文、検索語、認証情報または展開済み内容を含めない。

## 検証

各 provider operation は、少なくとも次を fixture または合成入力で検証する。

- ceiling より小さい正常系
- provider-specific lower budget の直前
- `responseBytes` 超過
- 展開後だけが `decompressedBytes` を超える case
- `entriesOrObjects` 超過
- `depth` 超過
- `parseTimeout` 超過
- 同じ operation の `concurrency` 超過時に外部呼出しへ到達しないこと
- 同じ `concurrencyGroup` の異なる operation を同時実行して共有上限を超えた場合も、後から開始した処理が外部呼出しへ到達しないこと
- キャンセル、timeout および parser failure の後に枠が解放されること

すべての artifact は `responseBytes`、`decompressedBytes`、`entriesOrObjects`、`depth`、`parseTimeout`、`concurrencyGroup` および `concurrency` の数値または識別子を持つ。適用しないとして `0`、`n/a` または未記載にせず、構造を持たない場合は一単位、深さ 1 として明示する。

conformance matrix は `budgetSotId` と `budgetKey` の組で一つの budget row を参照する。同じ SOT に複数の operation または artifact の予算がある場合も、組が一意にならなければならない。

## 関連

- [SOT-ENG-013: プロバイダー契約の検証](13-provider-contract-verification.md)
- [SOT-ENG-017: プロバイダー適合性 matrix](17-provider-conformance-matrix.md)
- [SOT-IF-017: 情報源エラーの正規化](../40-interfaces/17-source-error-normalization.md)
- [SOT-ARCH-010: プロバイダーの分離](../30-architecture/10-provider-isolation.md)
