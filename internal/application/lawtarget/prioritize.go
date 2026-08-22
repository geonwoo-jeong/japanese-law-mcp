package lawtarget

import "fmt"

// Prioritize は、対象 item と対象外 item の相対順を保つ新しい slice を返す。
func Prioritize[T any](
	items []T,
	target ResolvedLawTarget,
	identifier func(T) string,
) ([]T, bool, error) {
	if err := target.Validate(); err != nil {
		return nil, false, err
	}
	if identifier == nil {
		return nil, false, fmt.Errorf("法令対象を取得する関数は必須です")
	}
	matched := make([]T, 0, len(items))
	unmatched := make([]T, 0, len(items))
	requiresMove := false
	sawUnmatched := false
	for _, item := range items {
		if identifier(item) == target.LawID() {
			matched = append(matched, item)
			if sawUnmatched {
				requiresMove = true
			}
			continue
		}
		sawUnmatched = true
		unmatched = append(unmatched, item)
	}
	if len(matched) == 0 || !requiresMove {
		return append([]T(nil), items...), false, nil
	}
	prioritized := make([]T, 0, len(items))
	prioritized = append(prioritized, matched...)
	prioritized = append(prioritized, unmatched...)
	return prioritized, true, nil
}
