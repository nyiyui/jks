package layout

import (
	"sort"
)

type Box interface {
	Layout() (top int, height int)
}

// Layout puts boxes in a multi-column layout.
// The boxes slice is mutated.
func Layout[T Box](boxes []T, minHeight int) (nColumns int, columns []int) {
	sort.SliceStable(boxes, func(i, j int) bool {
		_, heightA := boxes[i].Layout()
		_, heightB := boxes[j].Layout()
		return max(heightA, minHeight) > max(heightB, minHeight)
	})
	sort.SliceStable(boxes, func(i, j int) bool {
		topA, _ := boxes[i].Layout()
		topB, _ := boxes[j].Layout()
		return topA < topB
	})
	columns = make([]int, len(boxes))
	columnBottoms := make([]int, len(boxes))
	// make sure each box does not overlap with each other
	for i, box := range boxes {
		top, height := box.Layout()
		height = max(height, minHeight)
		column := 0
		for columnBottoms[column] > top {
			column++
		}
		columns[i] = column
		if column+1 > nColumns {
			nColumns = column + 1
		}
		columnBottoms[column] = top + height
	}
	return nColumns, columns
}
