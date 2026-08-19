package tuiutil

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"retro-tool/internal/textutil"
)

var (
	StyleSub    = lipgloss.NewStyle().Foreground(lipgloss.Color("#87CEFA"))
	StyleOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("#32CD32"))
	StyleErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6347"))
	StyleNormal = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0"))
	StyleHint   = lipgloss.NewStyle().Foreground(lipgloss.Color("#808080"))
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
	return StyleSet{
		Title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#98FB98")),
		Sub:      StyleSub,
		Hint:     StyleHint,
		OK:       StyleOK,
		Err:      StyleErr,
		Warn:     lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")),
		Cursor:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF")),
		Selected: lipgloss.NewStyle().Foreground(lipgloss.Color("#32CD32")),
		Normal:   StyleNormal,
	}
}

func TitleBar(s StyleSet) string {
	return s.Title.Render("🎮 复古游戏工具")
}

func ScreenHeader(s StyleSet, icon, title, hint string) string {
	return s.Sub.Render(icon+" "+title) + "  " + s.Hint.Render(hint)
}

func PageHeader(s StyleSet, icon, title, hint string) string {
	return TitleBar(s) + "\n\n" + ScreenHeader(s, icon, title, hint) + "\n\n"
}

func MenuItem(s StyleSet, cur, label string, selected bool) string {
	if selected {
		return cur + s.Normal.Bold(true).Render(label)
	}
	return cur + s.Normal.Render(label)
}

func RenderMenu(s StyleSet, items []string, idx int, render func(string, int) string) string {
	var b strings.Builder
	for i, it := range items {
		cur := "  "
		if i == idx {
			cur = s.Cursor.Render("> ")
		}
		if render != nil {
			b.WriteString(cur)
			b.WriteString(render(it, i))
		} else {
			b.WriteString(MenuItem(s, cur, it, i == idx))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func Field(s StyleSet, label, value string) string {
	return s.Sub.Render(textutil.PadRight(label+":", 9)) + "  " + value
}

func BoxedHint(s StyleSet, spin, text string) string {
	return s.Hint.Render(spin + " " + text + " " + spin)
}

func StatLine(s StyleSet, selected, total int, size string) string {
	return s.Selected.Render(fmt.Sprintf("已选择 %d / %d 个目录（%s）", selected, total, size))
}
