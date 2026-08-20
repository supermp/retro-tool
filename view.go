package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"

	"retro-tool/internal/gamelist"
	"retro-tool/internal/install"
	"retro-tool/internal/ui"
)

type Screen int

const (
	scrDirSelect Screen = iota
	scrActionSelect
	scrInstallSystemSelect
	scrInstallDriveSelect
	scrInstallConfirm
	scrInstallProgress
	scrInstallSummary
	scrGamelistProgress
	scrGamelistSummary
)

const (
	maxFailWidth       = 34
	maxDirWidth        = 20
	maxDirColW         = 18
	minShowFailuresNum = 5
)

var styles = ui.DefaultStyles()

func actionNames() []string { return []string{"安装游戏", "生成 Gamelist"} }

type dirStatCell struct {
	Src, Dst string
	Files    int64
	Bytes    int64
}

func (m Model) View() tea.View {
	if m.width == 0 {
		m.width = 80
	}
	var content string
	switch m.screen {
	case scrDirSelect:
		content = m.dirSelectView()
	case scrActionSelect:
		content = m.actionSelectView()
	case scrInstallSystemSelect:
		content = m.installSystemSelectView()
	case scrInstallDriveSelect:
		content = m.installDriveSelectView()
	case scrInstallConfirm:
		content = m.installConfirmView()
	case scrInstallProgress:
		content = m.installProgressView()
	case scrInstallSummary:
		content = m.installSummaryView()
	case scrGamelistProgress:
		content = m.gamelistProgressView()
	case scrGamelistSummary:
		content = m.gamelistSummaryView()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) dirSelectView() string {
	var b strings.Builder
	b.WriteString(ui.PageHeader(styles, "📁", "选择游戏目录", "空格切换，A 全选，Enter 确认"))
	switch {
	case !m.scanned:
		b.WriteString(ui.BoxedHint(styles, m.spin.View(), "正在载入游戏目录"))
		b.WriteString("\n\n")
	case m.scanErr != nil:
		scanMsg := "⚠ 扫描源目录失败: " + m.scanErr.Error()
		b.WriteString(styles.Err.Render(scanMsg))
		b.WriteString("\n")
		b.WriteString(styles.Hint.Render("确认目录可访问，按 R 重试"))
		b.WriteString("\n\n")
	case len(m.dirs) == 0:
		b.WriteString(styles.Normal.Render("当前目录下没有子目录"))
		b.WriteString("\n\n")
	}

	nameW, sizeW := 0, 0
	sizeStrs := make([]string, len(m.dirs))
	for i, d := range m.dirs {
		if w := runewidth.StringWidth(d.Name); w > nameW {
			nameW = w
		}
		sizeStrs[i] = ui.HumanBytes(d.Bytes)
		if w := runewidth.StringWidth(sizeStrs[i]); w > sizeW {
			sizeW = w
		}
	}
	cellW := nameW + 4
	for i, d := range m.dirs {
		mark := "[ ]"
		markStyle := styles.Normal
		if m.checkedDirs[d.Name] {
			mark = "[x]"
			markStyle = styles.Selected
		}
		cur := "  "
		if i == m.dirIdx {
			cur = styles.Cursor.Render("> ")
		}
		name := markStyle.Render(ui.PadRight(mark+" "+d.Name, cellW))
		info := styles.Hint.Render(fmt.Sprintf("%10d 文件  %s", d.Files, ui.PadLeft(sizeStrs[i], sizeW)))
		fmt.Fprintf(&b, "%s%s%s\n", cur, name, info)
	}

	selected := m.selectedDirs()
	var total int64
	for _, d := range selected {
		total += d.Bytes
	}
	if m.screenErr != "" {
		b.WriteString("\n")
		b.WriteString(styles.Err.Render(m.screenErr))
		b.WriteString("\n")
	}
	if m.scanned {
		b.WriteString("\n")
		b.WriteString(ui.StatLine(styles, len(selected), len(m.dirs), ui.HumanBytes(total)))
	}
	return b.String()
}

func (m Model) actionSelectView() string {
	var b strings.Builder
	b.WriteString(ui.PageHeader(styles, "👆", "选择操作", "Enter 确认，Esc 返回"))
	items := actionNames()
	b.WriteString(ui.RenderMenu(styles, items, m.actionIdx))
	return b.String()
}

func (m Model) installSystemSelectView() string {
	var b strings.Builder
	b.WriteString(ui.PageHeader(styles, "📟", "选择系统类型", "Enter 确认，Esc 返回"))
	systems := install.AllSystems()
	items := make([]string, len(systems))
	for i, s := range systems {
		items[i] = string(s)
	}
	b.WriteString(ui.RenderMenu(styles, items, m.install.sysIdx))
	return b.String()
}

func (m Model) installDriveSelectView() string {
	var b strings.Builder
	b.WriteString(ui.PageHeader(styles, "💾", "选择目标盘符", "Enter 确认，Esc 返回，R 重新扫描，F 显示全部磁盘"))
	if len(m.install.drives) == 0 {
		b.WriteString(styles.Err.Render("⚠ 未发现可移动磁盘"))
		b.WriteString("\n")
		b.WriteString(styles.Hint.Render("插入磁盘后按 R 重新扫描；仍无法识别可按 F 查看全部磁盘（含硬盘）"))
		b.WriteString("\n\n")
	}

	type driveRow struct {
		label    string
		typeName string
		cap      string
		warn     bool
	}
	rows := make([]driveRow, len(m.install.drives))
	labelW, typeW, capW := 0, 0, 0
	for i, d := range m.install.drives {
		label := d.Label
		if label == "" {
			label = "无卷标"
		}
		typeName := d.TypeName()
		cap := "（容量未知）"
		if d.Total > 0 {
			cap = fmt.Sprintf("（剩余 %s）", ui.HumanBytes(d.Free))
		}
		rows[i] = driveRow{label: label, typeName: typeName, cap: cap, warn: !d.IsRemovable()}
		if w := runewidth.StringWidth(label); w > labelW {
			labelW = w
		}
		if w := runewidth.StringWidth(typeName); w > typeW {
			typeW = w
		}
		if w := runewidth.StringWidth(cap); w > capW {
			capW = w
		}
	}

	for i, d := range m.install.drives {
		cur := "  "
		if i == m.install.driveIdx {
			cur = styles.Cursor.Render("> ")
		}
		r := rows[i]
		warn := ""
		if r.warn {
			warn = styles.Err.Render("  ⚠")
		}
		line := fmt.Sprintf("%s  %s  %s  %s",
			styles.Normal.Bold(true).Render(d.Root),
			ui.PadRight("["+r.label+"]", labelW+2),
			ui.PadRight(r.typeName, typeW),
			ui.PadLeft(r.cap, capW),
		)
		b.WriteString(cur)
		b.WriteString(styles.Normal.Render(line))
		b.WriteString(warn)
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) installConfirmView() string {
	var b strings.Builder
	b.WriteString(ui.PageHeader(styles, "📦", "安装游戏确认", "Enter 确认，Esc 返回"))
	b.WriteString(ui.Field(styles, "系统类型", styles.Normal.Render(string(install.AllSystems()[m.install.sysIdx]))))
	b.WriteString("\n")

	selected := m.selectedDirs()
	totalBytes := int64(0)
	for _, d := range selected {
		totalBytes += d.Bytes
	}

	if len(m.install.drives) > m.install.driveIdx && m.install.driveIdx >= 0 {
		d := m.install.drives[m.install.driveIdx]
		label := d.Label
		if label == "" {
			label = "无卷标"
		}
		driveVal := d.Root + "  [" + label + "]  " + d.TypeName() + "（剩余 " + ui.HumanBytes(d.Free) + "）"
		b.WriteString(ui.Field(styles, "目标盘符", styles.Normal.Render(driveVal)))
		b.WriteString("\n")
		if d.Free > 0 && d.Free < totalBytes {
			warnText := fmt.Sprintf("⚠ 警告: 目标剩余空间小于需要安装的 %s", ui.HumanBytes(totalBytes))
			b.WriteString(styles.Warn.Render(warnText))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	cells := make([]dirStatCell, 0, len(selected))
	for _, d := range selected {
		cells = append(cells, dirStatCell{
			Src: d.Name, Dst: formatDir(m.install.sysRule, d.Name),
			Files: d.Files, Bytes: d.Bytes,
		})
	}
	b.WriteString(renderInstallStateTable(cells, dirsFiles(selected), totalBytes))
	b.WriteString("\n")
	if m.screenErr != "" {
		b.WriteString("\n")
		b.WriteString(styles.Err.Render(m.screenErr))
		b.WriteString("\n")
	}
	return b.String()
}

func dirsFiles(dirs []install.DirInfo) int64 {
	var n int64
	for _, d := range dirs {
		n += d.Files
	}
	return n
}

func formatDir(rule install.InstallRule, name string) string {
	if rule == nil {
		return name
	}
	return rule.DirName(name)
}

func (m Model) installProgressView() string {
	var b strings.Builder
	b.WriteString(ui.PageHeader(styles, "📤", "安装游戏中", "Esc 中止"))
	p := m.install.progress
	if p.TotalBytes == 0 {
		b.WriteString(ui.BoxedHint(styles, m.spin.View(), "正在扫描游戏文件"))
		b.WriteString("\n")
		return b.String()
	}
	frac := float64(p.DoneBytes) / float64(p.TotalBytes)
	bar := m.progressBar.ViewAs(frac)
	b.WriteString(bar)
	b.WriteString(" ")
	b.WriteString(styles.Sub.Render(fmt.Sprintf("%5.1f%%", frac*100)))
	b.WriteString("\n\n")

	cur := p.CurFile
	if cur != "" {
		if rel, err := filepath.Rel(m.srcRoot, cur); err == nil {
			cur = rel
		}
		srcLine := "安装 " + cur
		dstLine := "  至 " + p.CurDstDir
		maxW := m.width - 6
		if maxW > 10 {
			if runewidth.StringWidth(srcLine) > maxW {
				srcLine = ui.Truncate(srcLine, maxW)
			}
			if runewidth.StringWidth(dstLine) > maxW {
				dstLine = ui.Truncate(dstLine, maxW)
			}
		}
		b.WriteString(styles.Hint.Render("  " + srcLine))
		b.WriteString("\n")
		b.WriteString(styles.Hint.Render("  " + dstLine))
	} else {
		b.WriteString(styles.Hint.Render("  准备中"))
		b.WriteString("\n")
	}
	b.WriteString("\n\n")
	b.WriteString(renderInstallProgressTable(p))
	return b.String()
}

func (m Model) installSummaryView() string {
	var b strings.Builder
	b.WriteString(ui.PageHeader(styles, "📊", "安装游戏汇总", "Esc 返回"))
	r := m.install.result
	errCount := len(r.Errors)
	switch {
	case r.Aborted:
		b.WriteString(styles.Err.Render("✘ 已中止"))
		b.WriteString("\n")
	case errCount == 0:
		b.WriteString(styles.OK.Render("✔ 安装完成"))
		b.WriteString("\n")
	default:
		errText := fmt.Sprintf("✘ 安装完成，但有 %d 个错误", errCount)
		b.WriteString(styles.Err.Render(errText))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	cells := make([]dirStatCell, 0, len(r.PerDir))
	for _, pr := range r.PerDir {
		cells = append(cells, dirStatCell{Src: pr.SrcName, Dst: pr.DstName, Files: pr.Files, Bytes: pr.Bytes})
	}
	b.WriteString(renderInstallStateTable(cells, r.Files, r.Bytes))
	if errCount > 0 {
		b.WriteString("\n")
		b.WriteString(styles.Sub.Render("错误详情:"))
		b.WriteString("\n")
		for _, e := range r.Errors {
			item := "  - " + e
			b.WriteString(styles.Err.Render(item))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n\n")
	return b.String()
}

func (m Model) gamelistProgressView() string {
	var b strings.Builder
	b.WriteString(ui.PageHeader(styles, "🔍", "生成 Gamelist 中", "Esc 中止"))
	total := len(m.gamelist.dirNames)
	dir := ""
	if m.gamelist.currentIdx < total {
		dir = m.gamelist.dirNames[m.gamelist.currentIdx]
	}
	n := min(m.gamelist.currentIdx, total)
	frac := 0.0
	if total > 0 {
		frac = float64(n) / float64(total)
	}
	bar := m.progressBar.ViewAs(frac)
	b.WriteString(bar)
	b.WriteString("  ")
	num := ui.PadLeft(fmt.Sprint(n), len(fmt.Sprint(total)))
	b.WriteString(styles.Sub.Render(fmt.Sprintf("%s/%d", num, total)))
	b.WriteString("\n\n")

	status := ""
	if dir != "" {
		status = "生成 " + dir + "/gamelist.xml"
	}
	b.WriteString(styles.Hint.Render("  " + status))
	elapsed := time.Since(m.gamelist.startedAt).Round(time.Second)
	right := ui.PadLeft(elapsed.String(), 10)
	pad := m.width - 2 - runewidth.StringWidth(status) - runewidth.StringWidth(elapsed.String())
	if pad > 0 {
		b.WriteString(strings.Repeat(" ", pad))
	}
	b.WriteString(styles.Hint.Render(right))
	b.WriteString("\n\n")
	return b.String()
}

func (m Model) gamelistSummaryView() string {
	var b strings.Builder
	b.WriteString(ui.PageHeader(styles, "📊", "生成 Gamelist 汇总", "Enter 重新生成，Esc 返回"))
	if len(m.gamelist.results) < len(m.gamelist.dirNames) {
		b.WriteString(styles.Err.Render("✘ 已中止"))
		b.WriteString("\n\n")
	} else {
		b.WriteString(styles.OK.Render("✔ 生成完成"))
		b.WriteString("\n\n")
	}
	b.WriteString(renderGamelistTable(m.gamelist.results))
	b.WriteString("\n\n")
	return b.String()
}

func renderInstallStateTable(rows []dirStatCell, sumFiles, sumBytes int64) string {
	headers := []string{"源目录", "目标目录", "数量", "大小"}
	caps := []int{maxDirColW, maxDirColW, 10, 10}
	right := []bool{false, false, true, true}

	tbl := make([][]string, 0, len(rows)+2)
	for _, r := range rows {
		tbl = append(tbl, []string{
			r.Src, r.Dst,
			styles.Hint.Render(fmt.Sprint(r.Files)),
			styles.Hint.Render(ui.HumanBytes(r.Bytes)),
		})
	}
	tbl = append(tbl, make([]string, len(headers)))
	tbl = append(tbl, []string{
		"合计", "",
		styles.Hint.Render(fmt.Sprint(sumFiles)),
		styles.Hint.Render(ui.HumanBytes(sumBytes)),
	})
	return ui.RenderTable(headers, caps, right, tbl)
}

func renderInstallProgressTable(p install.InstallProgress) string {
	sz := func(b int64) string { return ui.HumanBytes(b) }
	const (
		colLabel = 10
		colNum   = 10
		colSpeed = 12
	)
	num := func(v int64) string { return ui.PadLeft(fmt.Sprint(v), colNum) }
	num2 := func(v int64) string { return ui.PadLeft(sz(v), colNum) }
	rate := fmt.Sprintf("%.0f 个/s", p.FilesPerSec)
	speed := "--"
	if p.SpeedBps > 0 {
		speed = ui.HumanBytes(int64(p.SpeedBps)) + "/s"
	}
	rows := [][]string{
		{
			ui.PadRight("数量", colLabel),
			styles.Hint.Render(num(p.DoneFiles)),
			styles.Hint.Render(num(p.DirFiles)),
			styles.Hint.Render(num(p.TotalFiles)),
			styles.Hint.Render(ui.PadLeft(rate, colSpeed)),
		},
		{
			ui.PadRight("大小", colLabel),
			styles.Hint.Render(num2(p.DoneBytes)),
			styles.Hint.Render(num2(p.DirBytes)),
			styles.Hint.Render(num2(p.TotalBytes)),
			styles.Hint.Render(ui.PadLeft(speed, colSpeed)),
		},
	}
	return ui.RenderTable([]string{"", "已完成", "当前目录", "所有目录", "实时速度"},
		[]int{colLabel, colNum, colNum, colNum, colSpeed},
		[]bool{false, true, true, true, true}, rows)
}

func renderGamelistTable(results []gamelist.Result) string {
	headers := []string{"目录", "总数", "成功", "失败列表(无图片)"}
	caps := []int{maxDirWidth, 8, 8, maxFailWidth}
	right := []bool{false, true, true, false}

	rows := make([][]string, 0, len(results))
	for _, r := range results {
		if r.Skipped {
			rows = append(rows, []string{
				r.Dir, "-", "-",
				styles.Normal.Render("街机游戏，跳过"),
			})
			continue
		}
		errs := len(r.Failed) > 0
		row := make([]string, len(headers))
		row[0] = r.Dir
		row[1] = styles.Sub.Render(fmt.Sprint(r.Total))
		row[2] = styles.OK.Render(fmt.Sprint(r.Success))
		row[3] = formatFailedList(r.Failed, maxFailWidth)
		if errs {
			row[3] = styles.Err.Render(row[3])
		}
		rows = append(rows, row)
	}
	return ui.RenderTable(headers, caps, right, rows)
}

func formatFailedList(failed []string, maxWidth int) string {
	if len(failed) == 0 {
		return "-"
	}
	total := len(failed)
	showAtLeast := min(minShowFailuresNum, total)
	for n := total; n >= showAtLeast; n-- {
		prefix := strings.Join(failed[:n], ", ")
		if n == total {
			if runewidth.StringWidth(prefix) <= maxWidth {
				return prefix
			}
			continue
		}
		suffix := fmt.Sprintf(" +%d", total-n)
		if runewidth.StringWidth(prefix+suffix) <= maxWidth {
			return prefix + suffix
		}
	}
	prefix := strings.Join(failed[:showAtLeast], ", ")
	if total > showAtLeast {
		prefix += fmt.Sprintf(" +%d", total-showAtLeast)
	}
	if runewidth.StringWidth(prefix) > maxWidth {
		return ui.Truncate(prefix, maxWidth)
	}
	return prefix
}
