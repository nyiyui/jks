package layout

import (
	"bytes"
	"log"
	"reflect"
	"testing"
)

type box struct {
	top    int
	height int
}

func (b box) Layout() (int, int) {
	return b.top, b.height
}

func toString(boxes []Box, nColumns int, columns []int) string {
	nRows := 0
	for _, box := range boxes {
		top, height := box.Layout()
		if top+height+1 > nRows {
			nRows = top + height + 1
		}
	}
	lines := make([][]byte, nRows)
	for i, box := range boxes {
		top, height := box.Layout()
		for j := 0; j < height; j++ {
			line := lines[top+j]
			if line == nil {
				line = make([]byte, nColumns)
			}
			log.Printf("line[%d]: %v", top+j, line)
			log.Printf("column: %d", columns[i])
			line[columns[i]] = byte('0' + i)
			lines[top+j] = line
		}
	}
	for _, line := range lines {
		if line == nil {
			continue
		}
		for j := 0; j < nColumns; j++ {
			if line[j] == 0 {
				line[j] = ' '
			} else {
				break
			}
		}
	}
	return string(bytes.Join(lines, []byte("\n")))
}

func TestLayout(t *testing.T) {
	boxes := []Box{
		box{0, 1},
		box{1, 2},
		box{2, 2},
		box{3, 1},
	}
	// 0
	// 1
	// 12
	// 32
	nColumns, columns := Layout(boxes, 0)
	for _, column := range columns {
		if column+1 > nColumns {
			t.Errorf("column %d is out of range", column)
		}
	}
	t.Logf("layout:\n%s", toString(boxes, nColumns, columns))
	if nColumns != 2 {
		t.Errorf("expected 2 columns, got %d", nColumns)
	}
	if !reflect.DeepEqual(columns, []int{0, 0, 1, 0}) {
		t.Errorf("unexpected columns: %v", columns)
	}
}
