package model

import "fmt"

// LawRevisionKind は、情報源固有の改正区分を共通化した値である。
type LawRevisionKind string

const (
	LawRevisionKindEnactment        LawRevisionKind = "enactment"
	LawRevisionKindPartialAmendment LawRevisionKind = "partial_amendment"
	LawRevisionKindAffectedLaw      LawRevisionKind = "affected_law"
	LawRevisionKindRepeal           LawRevisionKind = "repeal"
)

// LawRevisionRepealStatus は、廃止、失効その他の状態を共通化した値である。
type LawRevisionRepealStatus string

const (
	LawRevisionRepealStatusNone                LawRevisionRepealStatus = "none"
	LawRevisionRepealStatusRepealed            LawRevisionRepealStatus = "repealed"
	LawRevisionRepealStatusExpired             LawRevisionRepealStatus = "expired"
	LawRevisionRepealStatusSuspended           LawRevisionRepealStatus = "suspended"
	LawRevisionRepealStatusLossOfEffectiveness LawRevisionRepealStatus = "loss_of_effectiveness"
)

// LawRevisionCurrentStatus は、履歴と現時点との関係を共通化した値である。
type LawRevisionCurrentStatus string

const (
	LawRevisionCurrentStatusCurrent  LawRevisionCurrentStatus = "current"
	LawRevisionCurrentStatusFuture   LawRevisionCurrentStatus = "future"
	LawRevisionCurrentStatusPrevious LawRevisionCurrentStatus = "previous"
	LawRevisionCurrentStatusRepealed LawRevisionCurrentStatus = "repealed"
)

func validateLawRevisionKind(value LawRevisionKind) error {
	switch value {
	case "",
		LawRevisionKindEnactment,
		LawRevisionKindPartialAmendment,
		LawRevisionKindAffectedLaw,
		LawRevisionKindRepeal:
		return nil
	default:
		return fmt.Errorf("revisionKind が定義されていません")
	}
}

func validateLawRevisionRepealStatus(value LawRevisionRepealStatus) error {
	switch value {
	case "",
		LawRevisionRepealStatusNone,
		LawRevisionRepealStatusRepealed,
		LawRevisionRepealStatusExpired,
		LawRevisionRepealStatusSuspended,
		LawRevisionRepealStatusLossOfEffectiveness:
		return nil
	default:
		return fmt.Errorf("repealStatus が定義されていません")
	}
}

func validateLawRevisionCurrentStatus(value LawRevisionCurrentStatus) error {
	switch value {
	case "",
		LawRevisionCurrentStatusCurrent,
		LawRevisionCurrentStatusFuture,
		LawRevisionCurrentStatusPrevious,
		LawRevisionCurrentStatusRepealed:
		return nil
	default:
		return fmt.Errorf("currentStatus が定義されていません")
	}
}
