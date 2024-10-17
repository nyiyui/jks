package ui

import (
	"fmt"
	"time"
)

func formatDuration(d time.Duration) string {
	if d == time.Duration(0) {
		return ""
	}
	d2 := d.Abs()
	return fmt.Sprintf("%dd %02d:%02d", int(d.Hours())/24, int(d2.Hours())%24, int(d2.Minutes())%60)
}
