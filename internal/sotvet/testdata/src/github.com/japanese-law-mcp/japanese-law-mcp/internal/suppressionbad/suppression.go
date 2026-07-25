package suppressionbad

var (
	missingEverything int //nolint:gosec // want "SOT.ENG.019"
	missingJapanese   int //nolint:gosec // SOT-ENG-019: fixed test value // want "SOT.ENG.019"
	missingSOT        int //nolint:gosec // 固定したテスト値である。 // want "SOT.ENG.019"
	malformedSOT      int //nolint:gosec // SOT-ENG-19: 固定したテスト値である。 // want "SOT.ENG.019"

	staticcheckMissingJapanese int //lint:ignore SA1000 SOT-ENG-019: fixed test value // want "SOT.ENG.019"
	staticcheckMissingSOT      int //lint:ignore SA1000 固定したテスト値である。 // want "SOT.ENG.019"
	staticcheckMissingCheck    int //lint:ignore all SOT-ENG-019: 固定したテスト値である。 // want "SOT.ENG.019"

	gosecBare            int // #nosec // want "SOT.ENG.019"
	gosecAll             int // #nosec all -- SOT-ENG-019: 固定したテスト値である。 // want "SOT.ENG.019"
	gosecMissingJapanese int // #nosec G101 -- SOT-ENG-019: fixed test value // want "SOT.ENG.019"
	gosecMissingSOT      int // #nosec G101 -- 固定したテスト値である。 // want "SOT.ENG.019"
	gosecBlockMissingSOT int /* #nosec G101 -- 固定したテスト値である。 */ // want "SOT.ENG.019"
	gosecGarbageRule     int // #nosec G101 garbage -- SOT-ENG-019: 固定したテスト値である。 // want "SOT.ENG.019"
	gosecDisableBare     int //gosec:disable // want "SOT.ENG.019"
	gosecDisableSpecific int //gosec:disable G101 -- SOT-ENG-019: 固定したテスト値である。 // want "SOT.ENG.019"
)

//lint:file-ignore SA1000 SOT-ENG-019: ファイル全体の抑制は禁止される。 // want "SOT.ENG.019"
func fileWideSuppression() {}
