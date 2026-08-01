# corpus-v11 候補 target seed

この文書は、`corpus-v11` の新しい holdout と development へ使う候補 target を、
公式の一次資料だけから集めた作業用 seed 集である。定義元ではなく、最終的な
fixture の採否、件数、category、`leakageGroupId` および期待値は
`SOT-ENG-026`、`SOT-ENG-038` および
[統合照会の意図判定導入順](30-unified-query-intent-rollout.md) に従って別に固定する。

## 目的

- `corpus-v10` で消費済みの holdout と topic を安易に再利用せず、`corpus-v11` の
  seed 候補を先に広く確保する
- 法令名、法令番号、条文本文、更新、裁判例検索、裁判例詳細および法務省 guidance
  を横断する query family を、公式 URL 付きで整理する
- uncommon な target を多めに確保し、略称、表記揺れ、日付、条番号、case keyword
  の variation を後で組みやすくする

## 収集方針

- 公式 URL は e-Gov 法令検索、裁判所および法務省の一次資料だけを使う
- 個人名を前面に出す query は seed にしない
- 裁判例は検索 page、匿名化された事件名または裁判所 PDF を優先する
- 同じ law を development と holdout の両方へ使う必要はなく、fixture family を
  分けて使う

## 候補一覧

| 種別 | target | 公式 URL | 向く fixture family | 使いどころ |
|---|---|---|---|---|
| 法令 | 医薬品、医療機器等の品質、有効性及び安全性の確保等に関する法律 | https://elaws.e-gov.go.jp/document?lawid=335AC0000000145_20230526_505AC0000000036 | `law-name-and-concept` `surface-variation` `content-search` | 正式名称が長く、略称・俗称・本文検索の分離に向く |
| 法令 | 官民データ活用推進基本法 | https://elaws.e-gov.go.jp/document?keyword=%E5%AE%98%E6%B0%91%E3%83%87%E3%83%BC%E3%82%BF&lawid=428AC1000000103_20191216_501AC0000000016 | `law-name-and-concept` `content-search` | 行政 DX 系の語と law 名検索の境界を作りやすい |
| 法令 | 計量法 | https://elaws.e-gov.go.jp/document?lawid=404AC0000000051 | `law-name-and-concept` `official-reference` | 短い law 名、法令番号、条文読取りの基本ケースに向く |
| 法令 | 家事事件手続法 | https://elaws.e-gov.go.jp/document?lawid=423AC0000000052_20260521_504AC0000000048 | `structured-location-and-date` `law.article.read` | 条・項・手続系 query、家事事件まわりの文脈分離に向く |
| 法令 | 民事調停法 | https://elaws.e-gov.go.jp/document?lawid=326AC1000000222_20260521_504AC0000000048 | `law-name-and-concept` `law.article.read` | 家事事件手続法との近接 topic を fail-closed にしやすい |
| 法令 | 住民基本台帳法 | https://elaws.e-gov.go.jp/document?lawid=342AC0000000081_20260527_507AC0000000046 | `law-name-and-concept` `content-search` `official-reference` | マイナンバー周辺と別 law として切り分けやすい |
| 法令 | 個人情報の保護に関する法律 | https://elaws.e-gov.go.jp/document?lawid=415AC0000000057_20260521_504AC0000000048 | `law-name-and-concept` `content-search` `surface-variation` | よく知られた俗称と正式名称、本文 keyword の境界確認に向く |
| 法令 | 情報通信技術を活用した行政の推進等に関する法律 | https://elaws.e-gov.go.jp/document?lawid=414AC0000000151_20250523_507AC0000000043 | `surface-variation` `content-search` | 長い正式名称と「デジタル手続」系の語の分離に向く |
| 法令 | 健康増進法 | https://elaws.e-gov.go.jp/document?lawid=414AC0000000103_20230401_504AC0000000076 | `law-name-and-concept` `law.article.read` | 短い law 名、受動喫煙や施設管理といった本文 query を作りやすい |
| 法令 | 重要電子計算機に対する不正な行為による被害の防止に関する法律 | https://elaws.e-gov.go.jp/document?lawid=507AC0000000042_20250701_000000000000000 | `surface-variation` `law-name-and-concept` `typo-variation` | 新しめで長い名称のため、省略・誤記・語順揺れに向く |
| 法令 | 沿岸漁場整備開発法施行令 | https://elaws.e-gov.go.jp/document?lawid=351CO0000000051 | `official-reference` `structured-location-and-date` | uncommon な政令名、法令番号、施行時点 query に向く |
| 法令 | 補助金等に係る予算の執行の適正化に関する法律施行令 | https://elaws.e-gov.go.jp/document?lawid=330CO0000000255_20241120_506CO0000000345 | `surface-variation` `law-name-and-concept` `content-search` | 非常に長い正式名称のため略称・途中語検索の境界に向く |
| 法令 | 国立大学法人法施行令 | https://elaws.e-gov.go.jp/document?lawid=415CO0000000478_20240401_505CO0000000362 | `official-reference` `content-search` | 法人法と施行令の取り違え防止に向く |
| 法令 | 鳥獣の保護及び管理並びに狩猟の適正化に関する法律 | https://elaws.e-gov.go.jp/document?lawid=414AC0000000088_20250425_507AC0000000028 | `surface-variation` `content-search` | 長い名称と wildlife 系の本文 keyword の両方を試せる |
| 法令 | 不正競争防止法 | https://elaws.e-gov.go.jp/document?lawid=405AC0000000047_20250722_507AC0000000026 | `law-name-and-concept` `content-search` `official-reference` | 家畜遺伝資源、不正取得など近接概念との分離に向く |
| 法令 | 国家公務員法 | https://elaws.e-gov.go.jp/document?lawid=322AC0000000120_20241225_506AC0000000072 | `official-reference` `content-search` | 「秘密保持」「服務」など generic keyword と law 名の切り分けに向く |
| 法令 | 民事訴訟法 | https://elaws.e-gov.go.jp/document?keyword=%E6%B0%91%E4%BA%8B%E8%A8%B4%E8%A8%9F%E6%B3%95&lawid=408AC0000000109_20201001_502AC0000000022 | `official-reference` `law.article.read` | 条番号 query と procedure law 同士の近接性確認に向く |
| 法令 | 人事訴訟法 | https://elaws.e-gov.go.jp/document?lawid=415AC0000000109_20251001_505AC0000000053 | `law-name-and-concept` `law.article.read` | 家事事件手続法、民事調停法との近接混同を試せる |
| 法令 | 裁判所法 | https://elaws.e-gov.go.jp/document?lawid=322AC0000000059_20251001_505AC0000000053 | `official-reference` `content-search` | 裁判所組織と裁判例検索 intent の混同防止に向く |
| 法令 | 出入国管理及び難民認定法 | https://elaws.e-gov.go.jp/document?lawid=326CO0000000319_20250523_507AC0000000039 | `official-reference` `law.article.read` `content-search` | 永住許可 guidance と law 本文の intent を切り分けやすい |
| guidance | 永住許可（入管法第22条） | https://www.moj.go.jp/isa/applications/procedures/eizyuu_00001.html | `official-reference` `unsupported-scope` `content-search` | law 本文ではなく手続 guidance であることを明確に扱う seed |
| guidance | 永住許可制度の適正化Q＆A | https://www.moj.go.jp/isa/immigration/faq/kanri_qa_00003.html | `unsupported-scope` `content-search` | 法令本文と行政 Q&A を区別する query に向く |
| guidance | 永住許可に関するガイドライン（令和8年2月24日改訂） | https://www.moj.go.jp/isa/applications/resources/nyukan_nyukan50.html?lang=so | `structured-location-and-date` `unsupported-scope` | 日付付き guidance と law read intent の分離に向く |
| 裁判例検索 | 裁判例検索トップ | https://www.courts.go.jp/hanrei/search1/index.html | `judicial-search` `input-boundary` | AND/OR、事件番号、裁判年月日、参照法条の検索 UI 契約を確認できる |
| 裁判例 | 平成28年(ネ)第2993号 地位確認等請求控訴事件 | https://www.courts.go.jp/assets/hanrei/hanrei-pdf-86961.pdf | `judicial-read` `judicial-search` | 労働契約法20条を含む匿名化 PDF の read seed として使える |
| 裁判例 | 第1191号 損害賠償等請求事件 | https://www.courts.go.jp/assets/hanrei/hanrei-pdf-89768.pdf | `judicial-read` `official-reference` | 「第1191号」という裁判所 PDF title で ref/read の boundary を試せる |
| 裁判例 | 労契法20条を含む控訴審 PDF | https://www.courts.go.jp/assets/hanrei/hanrei-pdf-88455.pdf | `judicial-read` `content-search` | 労契法20条 keyword を持つ別 PDF として search/read の差分に使える |
| 裁判例 | 第2100号 未払賃金等支払請求事件 | https://www.courts.go.jp/assets/hanrei/hanrei-pdf-87784.pdf | `judicial-read` `structured-location-and-date` | 号数と事件名の両方を含む read seed に向く |
| 裁判例 | 労契法20条を含む別控訴審 PDF | https://www.courts.go.jp/assets/hanrei/hanrei-pdf-88296.pdf | `judicial-read` `content-search` | 同一条文 topic の複数 case を持たせ、単一 read と search を分けやすい |

## 初期の使い分け案

### law 側で先に作りやすいもの

- 長い正式名称
  - 医薬品、医療機器等の品質、有効性及び安全性の確保等に関する法律
  - 補助金等に係る予算の執行の適正化に関する法律施行令
  - 情報通信技術を活用した行政の推進等に関する法律
  - 重要電子計算機に対する不正な行為による被害の防止に関する法律
- 近接手続 law
  - 家事事件手続法
  - 民事調停法
  - 人事訴訟法
- uncommon な行政・海事・大学系
  - 沿岸漁場整備開発法施行令
  - 国立大学法人法施行令
  - 鳥獣の保護及び管理並びに狩猟の適正化に関する法律

### judicial 側で先に作りやすいもの

- 裁判例検索トップの AND/OR、事件番号、日付検索
- 労契法20条を含む複数 PDF を使った `search` と `read` の分離
- 「第1191号」「第2100号」のような title を使う ref/read seed

## 次の作業

1. 上の seed から、`corpus-v10` の消費済み holdout と leakage group が重なりにくい
   law・case 群を選び、`corpus-v11` 用の target roster を閉じる
2. roster ごとに `law-name-and-concept`、`official-reference`、
   `structured-location-and-date`、`judicial-search`、`judicial-read` の最低件数を
   割り当てる
3. その後にだけ `internal/legalquerycorpus/repository_corpus_v11_test.go` の RED を
   本物の fixture へ接続する
