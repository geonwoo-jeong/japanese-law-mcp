# SOT-IF-071: 裁判所 PDF の判例引用抽出マッピング

- 状態: 有効

## 規定

`courts-hanrei-pdf` は、詳細 HTML が直接示した `full_text` PDF の text layer だけを使い、判例参照とその根拠を `SOT-IF-068` の共通出力へ対応させる。

## 入力対応

- `decision` は同じ request の詳細取得結果をそのまま使用する。
- `document` はその詳細に一回だけ含まれる `full_text` の `JudicialDocumentLink` をそのまま使用する。
- PDF URL、判例番号パターン、正規化辞書または fuzzy 候補を入力へ追加しない。

所属、origin、path および宣言された media type は外向き取得前に検証する。取得後は response の media type と `%PDF-` magic を parser 起動前に検証する。PDF を一回だけ取得し、自動再試行しない。

## parser の採用と隔離

純粋 Go の `github.com/tsawler/tabula` を固定した版で先に評価し、日本語 fixture、context 取消および全資源予算 gate を満たす場合だけ採用する。満たさない場合は、固定した版の `github.com/dslipak/pdf` を同じ gate で評価する。両方が満たさない場合は capability 契約、予算又は fixture を緩和せず、provider の conformance row を `planned` のままにしてこの実装段階を中止する。複数 parser を production runtime fallback として同梱しない。

parser は MCP と同じ executable の非公開 worker mode を子 process として起動する。検証済み PDF は標準入力、上限付きの構造化結果は標準出力だけで受け渡し、shell、公開 CLI option、network 又は外部 resource を worker から使用しない。親は request context と 4 秒の短い方を期限とし、timeout、取消、panic、異常終了又は protocol 違反時に worker を終了して reap する。worker の失敗で MCP process を終了させない。

## 抽出規則

- 判例参照として扱うのは、完全な事件番号、裁判所と裁判日の組又は一意な判例集表記など、採用済みの厳密な構文で対象同一性を構成できる明示的記載だけとする。
- 一意に構成できない場合、または複数の公表裁判例候補へ一致する場合は `unresolvedMentions` とする。
- 同一 PDF 内の重複言及は occurrence として数えられるが、最終 graph edge では統合できるよう根拠を保持する。
- 言及位置は PDF ページ番号と、そのページ内の短い周辺文字列で表せる範囲に限る。
- PDF bytes の SHA-256 を `sha256:{hex}` として provenance の `contentDigest` に保持する。`transformation=extracted`、`methodId=SOT-IF-071` とし、位置を確認できる場合だけ `Provenance.location` に page を設定する。

text layer から有効な文字を一つも取得できない場合は `document_text_unavailable` の成功縮退とする。壊れた encoding、暗号化又は安全に辿れない参照構造を空 text に読み替えない。

## 法的意味の禁止

抽出時に次を推論しない。

- 先例性、拘束力、判例変更、破棄、上訴帰結または treatment
- 法令 revision、施行時点または法的評価
- OCR 補正や画像推定に基づく本文補完

## 確認

小さな日本語 fixture と注入可能な小さい test budget を用い、明示的参照、重複 occurrence、オフターゲット、page 順、digest と provenance、image-only、暗号化、不正 magic、過大・深すぎる PDF、timeout、panic、取消、抜粋長上限、text layer 不在の成功縮退および法的意味の非推論を mapping テストで確認する。終了後に worker、pipe、一時ファイルと同時実行枠が残らないことも確認する。

## 関連

- [SOT-IF-068: `judicial-decision.case-citation.extract` capability v1](68-judicial-case-citation-extract-capability.md)
- [SOT-IF-070: 裁判所「裁判例検索」PDF 情報源](70-source-courts-hanrei-pdf.md)
