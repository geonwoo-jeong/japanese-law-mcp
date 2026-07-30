# 明確化と対象外の代表例

これらは semantic fixture が固定する非実行 plan であり、外部情報源を呼ばない。
`expected_public_status` は plan decision から `LegalQueryResult` へ変換される
同名の非実行 status を表す。

| example_id | example_kind | query | request_context | verification_artifact | expected_plan_decision | expected_public_status | expected_summary | related_sots |
|---|---|---|---|---|---|---|---|---|
| `clarify-general-term` | `semantic` | `制度について法情報を探したいです。` | `enabledPacks=[]; ref=省略; limitPerAttempt=省略` | `corpus-v9:semantic:development-ambiguity-general` | `needs_clarification` | `needs_clarification` | `制度` だけでは法令名検索と条文検索を安全に一意化できない | `SOT-MODEL-023` `SOT-ARCH-023` `SOT-IF-051` |
| `clarify-no-first-read` | `semantic` | `行政処分を含む法令を検索して、先頭の法令本文も読む` | `enabledPacks=[]; ref=省略; limitPerAttempt=省略` | `corpus-v9:semantic:holdout-safety-08` | `needs_clarification` | `needs_clarification` | 検索第一件を暗黙に後続 read へ渡さない | `SOT-SCN-009` `SOT-ARCH-023` `SOT-IF-051` |
| `judicial-pack-unavailable` | `semantic` | `司法パック無効時に医療過誤の裁判例を検索してください。` | `enabledPacks=[]; ref=省略; limitPerAttempt=省略` | `corpus-v9:semantic:development-pack-disabled` | `capability_unavailable` | `capability_unavailable` | 裁判例検索の意味は保ち、無効な pack を法令検索へ置き換えない | `SOT-PROD-011` `SOT-ARCH-019` `SOT-IF-051` |
| `unsupported-legal-advice` | `semantic` | `この契約に署名すべきか法的に判断してください。` | `enabledPacks=[]; ref=省略; limitPerAttempt=省略` | `corpus-v9:semantic:development-unsupported-advice` | `unsupported` | `unsupported` | 法的判断要求を情報取得へ縮約しない | `SOT-PROD-011` `SOT-MODEL-023` `SOT-IF-051` |
| `unsupported-translation` | `semantic` | `「善意の第三者」を英語に翻訳してください。` | `enabledPacks=[]; ref=省略; limitPerAttempt=省略` | `corpus-v9:semantic:holdout-unsupported-06` | `unsupported` | `unsupported` | 翻訳要求を法令本文検索へ読み替えない | `SOT-PROD-011` `SOT-MODEL-023` `SOT-IF-051` |
| `unsupported-external-resource` | `semantic` | `都道府県の未公開内部文書を横断検索してください。` | `enabledPacks=[]; ref=省略; limitPerAttempt=省略` | `corpus-v9:semantic:development-unsupported-resource` | `unsupported` | `unsupported` | 採用範囲外の情報源を既存 provider へ置き換えない | `SOT-PROD-011` `SOT-MODEL-023` `SOT-IF-051` |
