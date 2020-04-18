package tz

func trimPathByRoot(AtoW []int, BtoW []int, root int) ([]int, []int) {
	for i, el := range AtoW {
		if el == root {
			AtoW = AtoW[:i]
		}
	}

	AtoW = append(AtoW, root)

	for i, el := range BtoW {
		if el == root {
			BtoW = BtoW[:i]
		}
	}

	return AtoW, BtoW
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func markAsVisited(traversed map[int]bool, el int) (wasVisited bool) {
	_, wasVisited = traversed[el]
	return
}

func trimPrefix(AtoW []int, BtoW []int) ([]int, []int) {
	awLength := len(AtoW)
	bwLength := len(BtoW)

	traversed := make(map[int]bool)

	for i := 0; i < max(awLength, bwLength); i++ {
		if i < awLength {
			if markAsVisited(traversed, AtoW[i]) {
				return trimPathByRoot(AtoW, BtoW, AtoW[i])
			}
		}
		if i < bwLength {
			if markAsVisited(traversed, BtoW[i]) {
				return trimPathByRoot(AtoW, BtoW, BtoW[i])
			}
		}
	}

	return AtoW, BtoW
}
