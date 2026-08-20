package ui

import (
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var tableHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#87CEFA"))

func RenderTable(headers []string, caps []int, right []bool, rows [][]string) string {
	cols := len(headers)

	widths := make([]int, cols)
	for c := range headers {
		widths[c] = ansi.StringWidth(headers[c])
	}
	for _, r := range rows {
		for c := 0; c < cols && c < len(r); c++ {
			if w := ansi.StringWidth(r[c]); w > widths[c] {
				widths[c] = w
			}
		}
	}
	for c, capW := range caps {
		if capW > 0 {
			if c < len(right) && right[c] {
				widths[c] = capW
			} else if widths[c] > capW {
				widths[c] = capW
			}
		}
	}

	columns := make([]table.Column, cols)
	for c := range headers {
		title := headers[c]
		if c < len(right) && right[c] {
			title = strings.Repeat(" ", widths[c]-ansi.StringWidth(title)) + title
		}
		columns[c] = table.Column{Title: title, Width: widths[c]}
	}

	btRows := make([]table.Row, 0, len(rows))
	for _, r := range rows {
		row := make(table.Row, cols)
		for c := 0; c < cols; c++ {
			if c < len(r) {
				cell := r[c]
				if c < len(right) && right[c] && cell != "" {
					if pad := widths[c] - ansi.StringWidth(cell); pad > 0 {
						cell = strings.Repeat(" ", pad) + cell
					}
				}
				row[c] = cell
			}
		}
		btRows = append(btRows, row)
	}

	totalW := 0
	for _, c := range columns {
		totalW += c.Width + 2
	}
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(btRows),
		table.WithHeight(len(btRows)+1),
		table.WithWidth(totalW),
		table.WithStyles(table.Styles{
			Header:   tableHeaderStyle.Padding(0, 1),
			Cell:     lipgloss.NewStyle().Padding(0, 1),
			Selected: lipgloss.NewStyle(),
		}),
	)
	return frame(t.View())
}

func frame(view string) string {
	const borderColor = "240"
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	if len(lines) == 0 {
		return view
	}
	w := 0
	for _, ln := range lines {
		if n := ansi.StringWidth(ln); n > w {
			w = n
		}
	}
	band := lipgloss.NewStyle().Foreground(lipgloss.Color(borderColor)).Render(strings.Repeat("─", w))

	var b strings.Builder
	b.WriteString(lines[0])
	b.WriteString("\n")
	b.WriteString(band)
	b.WriteString("\n")
	for _, ln := range lines[1:] {
		b.WriteString(ln)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
