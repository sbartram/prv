package ui

// RED stubs: replaced by the real layout in the GREEN commit.

const colCount = 7

func columnWidths(cells [][]string, total int) []int { return make([]int, colCount) }

func truncateCell(s string, w int) string { return s }

func (m Model) visibleRows() int { return len(m.snap.PRs) }
