package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

type StyleSet struct {
	Title    lipgloss.Style
	Sub      lipgloss.Style
	Hint     lipgloss.Style
	OK       lipgloss.Style
	Err      lipgloss.Style
	Warn     lipgloss.Style
	Cursor   lipgloss.Style
	Selected lipgloss.Style
	Normal   lipgloss.Style
}

func DefaultStyles() StyleSet {
	sub := lipgloss.NewStyle().Foreground(lipgloss.Color("#87CEFA"))
	ok := lipgloss.NewStyle().Foreground(lipgloss.Color("#32CD32"))
	err := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6347"))
	normal := lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0"))
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#808080"))
	return StyleSet{
		Title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#98FB98")),
		Sub:      sub,
		Hint:     hint,
		OK:       ok,
		Err:      err,
		Warn:     lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")),
		Cursor:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF")),
		Selected: lipgloss.NewStyle().Foreground(lipgloss.Color("#32CD32")),
		Normal:   normal,
	}
}

func PageHeader(s StyleSet, icon, title, hint string) string {
	return s.Title.Render("🎮 复古游戏工具") + "\n\n" +
		s.Sub.Render(icon+" "+title) + "  " + s.Hint.Render(hint) + "\n\n"
}

func RenderMenu(s StyleSet, items []string, idx int) string {
	var b strings.Builder
	for i, it := range items {
		cur := "  "
		if i == idx {
			cur = s.Cursor.Render("> ")
		}
		if i == idx {
			b.WriteString(cur + s.Normal.Bold(true).Render(it))
		} else {
			b.WriteString(cur + s.Normal.Render(it))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func Field(s StyleSet, label, value string) string {
	return s.Sub.Render(PadRight(label+":", 9)) + "  " + value
}

func BoxedHint(s StyleSet, spin, text string) string {
	return s.Hint.Render(spin + " " + text + " " + spin)
}

func StatLine(s StyleSet, selected, total int, size string) string {
	return s.Selected.Render(fmt.Sprintf("已选择 %d / %d 个目录（%s）", selected, total, size))
}
