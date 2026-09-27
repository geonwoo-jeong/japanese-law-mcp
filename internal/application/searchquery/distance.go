package searchquery

func boundedDamerauLevenshtein(
	left []rune,
	right []rune,
	maximum int,
) int {
	if difference := absolute(len(left) - len(right)); difference > maximum {
		return maximum + 1
	}
	// SOT-ARCH-021/SOT-ARCH-030: 閾値内の帯だけを三行保持し、検索語を呼出し間で共有しない。
	const inlineMaximum = 3
	width := 2*maximum + 3
	var inlineDistances [3 * (2*inlineMaximum + 3)]int
	distances := inlineDistances[:]
	if width*3 > len(distances) {
		distances = make([]int, width*3)
	}
	infinity := maximum + 1
	for index := range distances {
		distances[index] = infinity
	}
	previousPrevious := distances[:width]
	previous := distances[width : width*2]
	current := distances[width*2 : width*3]
	for index := 0; index <= len(right) && index <= maximum; index++ {
		previous[index+maximum+1] = index
	}
	return bandedAdjacentDistance(left, right, maximum, previousPrevious, previous, current)
}

func bandedAdjacentDistance(left, right []rune, maximum int, previousPrevious, previous, current []int) int {
	for leftIndex := 1; leftIndex <= len(left); leftIndex++ {
		for index := range current {
			current[index] = maximum + 1
		}
		if leftIndex <= maximum {
			current[maximum+1-leftIndex] = leftIndex
		}
		start := max(1, leftIndex-maximum)
		end := min(len(right), leftIndex+maximum)
		for rightIndex := start; rightIndex <= end; rightIndex++ {
			// 帯の中央は主対角線で、左右の一要素は帯の外側の無限大を保持する。
			index := rightIndex - leftIndex + maximum + 1
			cost := 1
			if left[leftIndex-1] == right[rightIndex-1] {
				cost = 0
			}
			current[index] = min(
				previous[index+1]+1,
				current[index-1]+1,
				previous[index]+cost,
			)
			if leftIndex > 1 &&
				rightIndex > 1 &&
				left[leftIndex-1] == right[rightIndex-2] &&
				left[leftIndex-2] == right[rightIndex-1] {
				current[index] = min(
					current[index],
					previousPrevious[index]+1,
				)
			}
		}
		previousPrevious, previous, current = previous, current, previousPrevious
	}
	return min(previous[len(right)-len(left)+maximum+1], maximum+1)
}

func absolute(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
