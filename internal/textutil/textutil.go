package textutil

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

func PadRight(s string, w int) string {
	n := runewidth.StringWidth(s)
	if n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-n)
}

func PadLeft(s string, w int) string {
	n := runewidth.StringWidth(s)
	if n >= w {
		return s
	}
	return strings.Repeat(" ", w-n) + s
}

func Truncate(s string, w int) string {
	if runewidth.StringWidth(s) <= w {
		return s
	}
	budget := w - runewidth.StringWidth("…")
	var b strings.Builder
	width := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if width+rw > budget {
			break
		}
		b.WriteRune(r)
		width += rw
	}
	return b.String() + "…"
}

func HumanBytes(b int64) string {
	if b < 0 {
		b = 0
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
