package searchquery

func boundedDamerauLevenshtein(
	left []rune,
	right []rune,
	maximum int,
) int {
	if difference := absolute(len(left) - len(right)); difference > maximum {
		return maximum + 1
	}
	infinity := maximum + 1
	previousPrevious := filledDistances(len(right)+1, infinity)
	previous := filledDistances(len(right)+1, infinity)
	for index := 0; index <= len(right) && index <= maximum; index++ {
		previous[index] = index
	}

	for leftIndex := 1; leftIndex <= len(left); leftIndex++ {
		current := filledDistances(len(right)+1, infinity)
		if leftIndex <= maximum {
			current[0] = leftIndex
		}
		start := max(1, leftIndex-maximum)
		end := min(len(right), leftIndex+maximum)
		for rightIndex := start; rightIndex <= end; rightIndex++ {
			cost := 1
			if left[leftIndex-1] == right[rightIndex-1] {
				cost = 0
			}
			current[rightIndex] = min(
				previous[rightIndex]+1,
				current[rightIndex-1]+1,
				previous[rightIndex-1]+cost,
			)
			if leftIndex > 1 &&
				rightIndex > 1 &&
				left[leftIndex-1] == right[rightIndex-2] &&
				left[leftIndex-2] == right[rightIndex-1] {
				current[rightIndex] = min(
					current[rightIndex],
					previousPrevious[rightIndex-2]+1,
				)
			}
		}
		previousPrevious, previous = previous, current
	}
	if previous[len(right)] > maximum {
		return maximum + 1
	}
	return previous[len(right)]
}

func filledDistances(length int, value int) []int {
	values := make([]int, length)
	for index := range values {
		values[index] = value
	}
	return values
}

func absolute(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
