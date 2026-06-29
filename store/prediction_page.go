package store

func clampPageLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func clampPageOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
