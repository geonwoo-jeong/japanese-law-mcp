# SOT-ARCH-032: 統合照会の限定分岐保持

- 状態: 有効

## 規定

統合照会は、誤認識の影響を抑えるために、各 query profile が composer へ
渡す前の候補集合で複数の意味候補を内部保持できるが、無条件の並列検索へ
広げず、独立した根拠を持つ少数の代替候補だけを決定的に残す。

## 分岐の単位

本規定で保持数を校正する分岐の最小単位は、一つの query profile が
`QueryProfileContribution` を構築する前に作る一つの候補 draft とする。
provider route、外部結果、速度または失敗は分岐の単位にしない。

各 profile は、`SOT-ARCH-025` のうち自身の採用済み task/resource に適用できる
複数主題規則を先に適用する。適用できる規則がない profile は、複数主題を
推測して step 化せず、一主題の候補 draft として扱う。その後、候補 draft を
evidence cluster ごとに校正し、上限と
`selectionMode` を確定してから contribution を構築する。
`SOT-ARCH-027` の composer は、その検証済み contribution を後段で合成する。
合成候補と合成後に残る non-member 候補を、再び profile 内 cluster へ戻して
本規定の margin で刈り込まない。合成後の置換、順位および全体上限は
`SOT-ARCH-027` と `SOT-MODEL-026` に従う。

別の主題、別の節または別の profile が作る候補は、互いの根拠を流用しない。
ただし、別 profile であることだけを保持理由にせず、各 profile 内では後述の
条件を満たさなければならない。照会文全体を一つの「広い検索語」にして
全 provider へ展開する分岐を作らない。

## 分岐を保持できる条件

ある候補を追加分岐として保持できるのは、少なくとも次の条件をすべて満たす場合だけとする。

1. 採用済みの task/resource に属する
2. `SOT-ARCH-031` の `boundary` により非実行境界へ落ちていない
3. 独立した正の根拠を一つ以上持つ
4. 同じ profile と `rankingVersion` に属する cluster 内の首位候補との差が、
   その profile set で校正した保持 margin の範囲内にある

「独立した正の根拠」は、次のいずれかとする。

- 公式識別子、法令番号、法令履歴 ID または検証済み `ref`
- 明示 task/resource cue と同じ節にある法令名、条項、事件番号または明示検索対象
- 公的根拠を持つ法概念辞書 entry
- `SOT-ARCH-025` の規則で独立主題と判定した一般検索語

弱い一般語だけ、対象外 relation の subject/predicate だけ、または別候補から流用した根拠だけでは追加分岐を保持できない。

保持 margin と分岐順は、`SOT-ENG-024` の評価で受け入れた校正値とする。profile 固有の候補保持規則だけを変える場合は profile version に属し、profile 横断の score policy と同じ尺度を変える場合は ranking version に属する。重みは順位付けのための内部値であり、確率、再試行優先度または公開 confidence そのものではない。文書化していない一時的なヒューリスティクス、実行時の観測件数または provider 固有の fallback 成否で分岐保持を増減させない。

本規定の「追加分岐」は、実行適格な代替候補だけを指す。`SOT-MODEL-026` が `unsupported` または `mixed_unsupported_intent` の監査用に保持する内部候補は、条件 2 と保持 margin の対象外とし、選択または実行しない。

## 上限

- 同じ evidence cluster から保持できる代替分岐は三件以下とする
- 一つの `QueryProfileContribution` が保持する候補総数は、既存の上限どおり十六件以下とする
- planner が実行対象として選べる候補は `SOT-ARCH-023` に従い最大二件とする
- capability 呼出し総数は `SOT-MODEL-023` の固定予算を超えない

ここでいう evidence cluster は、同じ profile の候補 draft が持つ各 step について、
原文順に `{topicOrdinal, evidenceSpan}` を並べた一時的な key が一致する候補群を
表す。`evidenceSpan` は、その step に属する最初の明示 task/resource 根拠 span、
それがない場合は最初の `target_anchor` span、それもない場合は、その step を
実際に生成した最初の `semantic_expansion` の法概念または一般検索語 span とする。
別候補の span を流用せず、一 step でもこの span を持たない候補は追加分岐に
できない。この key は候補 draft の校正にだけ使い、
`LegalQueryCandidate` または `QueryProfileContribution` へ保存しない。
別 cluster の候補は互いの三件上限へ含めないが、contribution 構築前と
profile 横断合成後のいずれも、全体の十六候補上限を超えてはならない。

一つの主題から law、law_provision、judicial_decision など複数 resource へ自動展開した全組合せを cluster として保持することはできない。cluster は独立根拠のまとまりであり、resource の総当たり集合ではない。条件 1 から 4 を満たす四件目の代替候補を同じ cluster で検出した場合、profile は同じ profile の完全順序による上位三件だけを clarification 用候補として保持し、`selectionMode=clarification_required`、`hedgePairs=[]` とする。四件目以降は実行候補へ含めない。selector はこの入力を通常の候補不足または内部エラーにせず、`SOT-MODEL-023` の `ambiguous_candidates` を持つ `needs_clarification` へ変換する。

## 保持と実行の分離

分岐保持は、誤認識に備えて候補を残す内部手続であり、外部検索の即時並列実行を許可するものではない。保持した候補のうち、実際に実行できるのは `SOT-ARCH-023` が許可する単独候補または明示 hedge pair だけとする。

したがって、次を禁止する。

- 保持した全候補をそのまま並列検索すること
- `hedgePairs` がない候補同士を「念のため」で二件実行すること
- pack 無効の候補を、利用可能な下位候補へ静かに置き換えて実行すること
- `unsupported` または `needs_clarification` の候補を内部実行すること

## 多義語と列挙

同じ語が法令本文検索と裁判例検索の両方へ対応し得る場合、公的根拠を持つ法概念辞書または明示 resource cue があるときだけ、二分岐まで保持できる。単なる列挙や裸の名詞だけでは、法令と裁判例の両方へ無条件に分岐しない。

`永住許可と帰化について教えてください。` のように主題が複数ある照会は、`SOT-ARCH-025` に従い、原則として一候補内の別 step として保持する。別候補に分かれるのは、同じ主題に対する代替意味がある場合だけとし、主題ごとの全組合せを分岐として増殖させない。

## 公開境界

公開 MCP 結果には、保持だけした全分岐、保持 margin、重みの生値または棄却した分岐理由の全列挙を出さない。公開面へ出るのは、選択済み interpretation、明確化理由、対象外理由および実行結果だけとする。

## 確認

少なくとも次を、planner test、profile test および統合評価 fixture で確認する。

- 一つの法概念が法令本文検索と裁判例検索の両候補を持つ場合、二分岐まで保持できても無条件に二件実行しない
- 明示 read、明示条文または検証済み `ref` がある場合、弱い一般語からの広い検索分岐を保持しない
- `民法を検索してください。裁判例も検索してください。` は、明示主題の分離として保持し、照会全体を一検索へ広げない
- `民法第103条を引用する裁判例の影響グラフを作成してください。` は、対象外との混在により実行分岐を零件とする
- 同じ evidence cluster から四件目の代替候補を黙って保持せず、上位三件を clarification 用候補として残したまま `selectionMode=clarification_required` にする

## 関連

- [SOT-MODEL-022: LegalQueryCandidate](../20-model/22-legal-query-candidate.md)
- [SOT-MODEL-023: LegalQueryPlan](../20-model/23-legal-query-plan.md)
- [SOT-MODEL-026: QueryProfileContribution](../20-model/26-query-profile-contribution.md)
- [SOT-MODEL-028: QueryCandidateCompositionMember](../20-model/28-query-candidate-composition-member.md)
- [SOT-ARCH-023: 統合照会の候補選択と制限付き実行](23-unified-query-selection-and-hedging.md)
- [SOT-ARCH-025: 統合照会の複数主題分離](25-unified-query-multi-topic-separation.md)
- [SOT-ARCH-027: 統合照会の profile 横断候補合成](27-unified-query-cross-profile-composition.md)
- [SOT-ARCH-031: 統合照会の意図根拠レイヤ](31-unified-query-intent-evidence-layer.md)
