# SOT-IF-064: 国立国会図書館の国会発言検索マッピング

- 状態: 有効

## 規定

`ndl-diet-speech-api` の `parliament.speech.search@1` は、検証済みの検索条件を発言単位 API の固定 query parameter へ対応させ、JSON の一つの `speechRecord` を一つの `SourcedResource<ParliamentSpeech>` へ決定的に変換する。

## 要求

正規化済み入力を次のように対応させる。

| `ParliamentSpeechSearchRequestV1` | API query parameter |
|---|---|
| `query` | `any` |
| `speaker` | `speaker` |
| `meetingName` | `nameOfMeeting` |
| `house: house_of_representatives` | `nameOfHouse=衆議院` |
| `house: house_of_councillors` | `nameOfHouse=参議院` |
| `house: both_houses` | `nameOfHouse=両院` |
| `house: conference_of_both_houses` | `nameOfHouse=両院協議会` |
| `fromDate` | `from` |
| `untilDate` | `until` |
| 実効 `limit` | `maximumRecords` |

`startRecord=1` と `recordPacking=json` を常に加える。初回取得以外の開始位置を送らない。空でない `continuationToken` は要求 URL を作る前に `invalid_argument` とする。

存在する parameter だけを `any`、`speaker`、`nameOfMeeting`、`nameOfHouse`、`from`、`until`、`startRecord`、`maximumRecords`、`recordPacking` の順に一回ずつ付ける。名前と値を UTF-8 の RFC 3986 query component として一回だけ percent-encode し、半角空白は `%20` とする。入力が持つ内部の半角空白を分割、並べ替えまたは別の論理演算へ変換しない。

要求 URL 全体が `SOT-IF-063` の 2000 byte 上限を超える場合は、parameter を削除または短縮せず `invalid_argument` とする。

## JSON 契約

top-level object は次を持つ。

| JSON field | 必須 | 制約 |
|---|---:|---|
| `numberOfRecords` | はい | 0 以上の整数 |
| `numberOfReturn` | はい | 0 以上、実効 `limit` 以下の整数 |
| `startRecord` | はい | 整数 `1` |
| `nextRecordPosition` | いいえ | `null` または 2 以上の整数 |
| `speechRecord` | 条件付き | object の配列。`numberOfReturn` が 1 以上の場合は必須 |

`speechRecord.length` は `numberOfReturn` と一致し、`numberOfReturn` は `numberOfRecords` 以下とする。`numberOfReturn: 0` では `speechRecord` の欠落または空配列を受理し、内部では空配列へ正規化する。`numberOfRecords` が 1 以上で `numberOfReturn: 0` の応答は受理しない。

非 `null` の `nextRecordPosition` は `startRecord + numberOfReturn` と一致し、`numberOfReturn` が 0 ではなく、`numberOfReturn < numberOfRecords` とする。`nextRecordPosition` の欠落または `null` は末尾を表し、`numberOfReturn < numberOfRecords` と同時に現れた場合は `invalid_source_response` とする。

未知の top-level field と `speechRecord` field は、公式 API が同じ意味の項目を追加できるよう無視できる。ただし、既知 field の欠落、型不一致、重複 key、矛盾した件数、非前進の次位置または一つの JSON object の後に続く data を受理しない。

## 項目対応

| API `speechRecord` | `ParliamentSpeech` |
|---|---|
| `speechID` | `speechId` |
| `speechOrder` | `speechOrder` |
| `speaker` | `speaker.name` |
| `speakerYomi` | `speaker.reading` |
| `speakerGroup` | `speaker.group` |
| `speakerPosition` | `speaker.position` |
| `speakerRole` | `speaker.role` |
| `speech` | `speechText` |
| `startPage` | `startPage` |
| `speechURL` | `speechUrl` |
| `issueID` | `meeting.meetingRecordId` |
| `imageKind` | `meeting.imageKind` |
| `session` | `meeting.session` |
| `nameOfHouse` | `meeting.houseName` |
| `nameOfMeeting` | `meeting.meetingName` |
| `issue` | `meeting.issue` |
| `date` | `meeting.meetingDate` |
| `closing` | `meeting.closing`。`null` なら省略 |
| `meetingURL` | `meeting.meetingUrl` |
| `pdfURL` | `meeting.pdfUrl` |
| `SOT-IF-063` の情報源 | `source` |

空文字または `null` の `speakerYomi`、`speakerGroup`、`speakerPosition`、`speakerRole` および `pdfURL` は設定しない。`startPage` が `null` なら省略し、0 以上の整数なら 0 を含めて保持する。その他の必須 field の欠落、`null`、空文字または型不一致は `invalid_source_response` とする。

`searchObject` は発言単位 JSON に存在する場合も、確認済みの公式説明と実応答から provider-independent な型付き意味を確定できないため出力へ対応させない。`speechOrder`、`imageKind` または別の発言・会議項目へ読み替えず、安全な JSON value として資源予算に含めた後に破棄する。

`date` は実在する ISO `YYYY-MM-DD` として検証する。発言本文は先頭末尾の Unicode whitespace だけを除き、内部の改行と文字を保持する。その他の文字列は先頭末尾の Unicode whitespace だけを除き、空になった必須値を受理しない。

三つの URL は絶対 URL として解析し、userinfo、fragment および `https://kokkai.ndl.go.jp/` 以外の origin を拒否する。PDF の content を取得しない。

## 出典とページ

各 item は次を持つ。

- `ref.providerId`: `ndl-diet-speech-api`
- `ref.key.sourceId`: `ndl-diet-records`
- `ref.key.resourceType`: `parliament-speech`
- `ref.key.resourceId`: `speechID`
- `ref.key.versionId`: 省略

`provenance.url` は実際に取得した発言単位 API の要求 URL、`mediaType` は `application/json`、`transformation` は `normalized`、`methodId` は `SOT-IF-064`、`location` は配列内の該当 `speechRecord` を一意に示す値とする。最後の `resourceKey` は `ref.key` と一致させる。発言本文の公式参照先は `data.speechUrl` に保持する。

API の配列順を変更しない。`SourcePage.returnedCount` は `numberOfReturn`、`totalCount` は `numberOfRecords`、`totalRelation` は `exact` とする。`nextRecordPosition` は page 不変条件の検証にだけ使用し、外部取得位置、公開数値または継続トークンへ変換しない。`nextToken` は常に省略する。

## 確認

六つの検索条件、四つの院名、parameter 順、percent-encoding、2000 byte 境界、空でない token の事前拒否、空結果、1 件と 30 件、総数と返却数、次位置の有無と不変条件、未知 field、重複 key、欠落・null・型不一致、全文改行、日付、三つの URL、省略可能項目、配列順、重複 item の保持、resource key、provenance および exact `SourcePage` を JSON fixture で確認する。

## 関連

- [SOT-MODEL-034: ParliamentSpeech](../20-model/34-parliament-speech.md)
- [SOT-IF-062: `parliament.speech.search` capability v1](62-parliament-speech-search-capability.md)
- [SOT-IF-063: 国立国会図書館の国会発言検索 API 情報源](63-source-ndl-diet-speech-api.md)
