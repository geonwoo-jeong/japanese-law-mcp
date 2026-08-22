package querypreprocess

import "fmt"

type typoCandidateLimitError struct {
	limit int
}

func (e typoCandidateLimitError) Error() string {
	return fmt.Sprintf(
		"誤記判定候補は %d 件以下でなければなりません",
		e.limit,
	)
}

func (typoCandidateLimitError) CandidateCaseFailure() {}
