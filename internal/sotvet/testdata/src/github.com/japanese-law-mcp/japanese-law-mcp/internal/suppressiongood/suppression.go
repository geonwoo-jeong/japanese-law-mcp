package suppressiongood

var (
	nolintSingle   int //nolint:gosec // SOT-ENG-019: 固定したテスト値である。
	nolintMultiple int //nolint:gosec,noctx // SOT-ENG-019: 固定したテスト値である。
	nolintSOTLast  int //nolint:gosec // 固定したテスト値である。SOT-ENG-019

	staticcheckSingle   int //lint:ignore SA1000 SOT-ENG-019: 固定したテスト値である。
	staticcheckMultiple int //lint:ignore SA1000,U1000 SOT-ENG-019: 固定したテスト値である。

	gosecSingle         int // #nosec G101 -- SOT-ENG-019: 固定したテスト値である。
	gosecMultiple       int // #nosec G101 G102 -- SOT-ENG-019: 固定したテスト値である。
	gosecBlock          int /* #nosec G101 -- SOT-ENG-019: 固定したテスト値である。 */
	gosecMultilineBlock int /* #nosec G101 --
	SOT-ENG-019: 固定したテスト値である。 */
)

// 通常の説明では nolint という語を引用する。
func ordinaryNolintSentence() {}

// #nosecurity は gosec の抑制指示ではない。
func ordinaryNosecSentence() {}

// 通常の説明では #nosec を引用する。
func ordinaryNosecToken() {}

//gosec:enable G101
func gosecEnableIsNotSuppression() {}

/*nolint:gosec // SOT-ENG-019: ブロックコメントは nolint 指示として解釈されない。*/
func blockNolintSentence() {}

//go:noinline
func compilerDirective() {}
