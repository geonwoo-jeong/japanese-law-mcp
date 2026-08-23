# SOT-MODEL-034: ParliamentSpeech

- 状態: 有効

## 規定

`ParliamentSpeech` は、公式の国会会議録検索システムに登録された一つの発言と、その発言が属する会議の参照情報を、発言と会議の意味を混在させず表す。

## 構造

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `speechId` | string | はい | 情報源が示す発言 ID |
| `speechOrder` | integer | はい | 会議録内の発言番号 |
| `speaker` | `ParliamentSpeaker` | はい | 発言者について情報源が示す属性 |
| `speechText` | string | はい | 発言単位出力の発言本文 |
| `startPage` | integer | いいえ | 情報源が示す発言の掲載開始頁 |
| `speechUrl` | string | はい | 発言を確認できる公式 URL |
| `meeting` | `ParliamentMeetingReference` | はい | 発言が属する会議録の参照情報 |
| `source` | `InformationSource` | はい | 発言情報を取得した情報源 |

`ParliamentSpeaker` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `name` | string | はい | 情報源が示す発言者名 |
| `reading` | string | いいえ | 情報源が示す発言者よみ |
| `group` | string | いいえ | 情報源が示す発言者所属会派 |
| `position` | string | いいえ | 情報源が示す発言者肩書き |
| `role` | string | いいえ | 情報源が示す発言者役割 |

`ParliamentMeetingReference` は次の構造とする。

| 項目 | 型 | 必須 | 意味 |
|---|---|---:|---|
| `meetingRecordId` | string | はい | 情報源の会議録 ID |
| `imageKind` | string | はい | 会議録、目次、索引、附録または追録という情報源のイメージ種別 |
| `session` | integer | はい | 国会回次 |
| `houseName` | string | はい | 情報源が示す院名 |
| `meetingName` | string | はい | 情報源が示す会議名 |
| `issue` | string | はい | 情報源が示す号数 |
| `meetingDate` | date | はい | 会議開催日 |
| `closing` | boolean | いいえ | 情報源が値を示す場合の閉会中フラグ |
| `meetingUrl` | string | はい | 会議録テキストを確認できる公式 URL |
| `pdfUrl` | string | いいえ | 情報源が示す会議録 PDF の公式 URL |

## 制約

- `speechId`、`meeting.meetingRecordId`、`speaker.name`、`speechText`、`meeting.houseName` および `meeting.meetingName` は空文字にしない。
- `speechOrder`、`startPage` および `meeting.session` は 0 以上とする。
- `meeting.meetingDate` は情報源が示す完全な年月日を検証し、`YYYY-MM-DD` で表す。年月または年だけの値を補完しない。
- 発言 ID と会議録 ID は大文字小文字、先頭のゼロおよび区切りを変更しない。
- `imageKind`、`houseName`、`issue`、発言者属性および発言本文は、情報源の語彙と意味を保持し、別の状態、分類または法的評価へ変換しない。
- 発言本文の先頭末尾にある Unicode whitespace だけを除ける。本文内の改行、空白、文字、順序および注記を変更しない。
- 発言者名または肩書きから所属、役割、議員資格その他の属性を推測しない。
- `speechUrl`、`meeting.meetingUrl` および存在する `meeting.pdfUrl` は、認証情報を含まない `https://kokkai.ndl.go.jp/` の URL とする。
- 会議情報を発言固有の項目へ平坦化せず、発言を会議、議案、法律または法令改正の資源へ変換しない。
- `SourceResourceKey.resourceType` は `parliament-speech`、`resourceId` は `speechId` とし、`versionId` は設定しない。
- JSON はこの表の camelCase 項目名を使用し、省略可能な値がない項目は `SOT-MODEL-009` に従って省略する。

## 確認

全必須項目、省略可能な発言者属性・開始頁・PDF、完全な日付、識別子の保持、本文の改行、公式 URL、発言と会議の入れ子境界、議案・法令項目の非混入および JSON 表現を単体テストで確認する。

## 関連

- [SOT-MODEL-009: JSON シリアライズ](09-json-serialization.md)
- [SOT-MODEL-010: InformationSource](10-information-source.md)
- [SOT-MODEL-011: SourceResourceKey](11-source-resource-key.md)
- [SOT-ARCH-018: 拡張パック単位の正規化境界](../30-architecture/18-pack-scoped-normalization-boundary.md)
- [SOT-IF-064: 国立国会図書館の国会発言検索マッピング](../40-interfaces/64-ndl-diet-speech-search-mapping.md)
