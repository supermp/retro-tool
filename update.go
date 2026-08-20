package main

import (
	"context"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"retro-tool/internal/drives"
	"retro-tool/internal/gamelist"
	"retro-tool/internal/install"
)

const (
	actInstall = iota
	actGamelist

	minGamelistDisplayTime = 3000 * time.Millisecond
)

type dirListMsg []install.DirInfo
type dirScanErrMsg struct{ err error }
type driveListMsg []drives.Drive
type installProgressMsg install.InstallProgress
type installDoneMsg install.InstallResult
type installPlanErrMsg struct{ err error }
type gamelistResultMsg struct{ r gamelist.Result }
type gamelistDoneMsg struct{}
type gamelistHoldMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)
	case dirListMsg:
		return m.handleDirList(msg)
	case dirScanErrMsg:
		return m.handleDirScanErr(msg)
	case driveListMsg:
		return m.handleDriveList(msg)
	case installProgressMsg:
		return m.handleInstallProgress(msg)
	case installPlanErrMsg:
		return m.handleInstallPlanErr(msg)
	case installDoneMsg:
		return m.handleInstallDone(msg)
	case gamelistResultMsg:
		return m.handleGamelistResult(msg)
	case gamelistDoneMsg:
		return m.handleGamelistDone(msg)
	case gamelistHoldMsg:
		return m.handleGamelistHold()
	default:
		return m, nil
	}
}

func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.progressBar.SetWidth(min(max(m.width-55, 20), 55))
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := strings.ToLower(msg.String())

	if key == "ctrl+c" {
		if m.cancel != nil {
			m.cancel()
			return m, nil
		}
		return m, tea.Quit
	}

	switch m.screen {
	case scrDirSelect:
		switch key {
		case "up":
			moveSelectionIndex(&m.dirIdx, len(m.dirs), -1)
		case "down":
			moveSelectionIndex(&m.dirIdx, len(m.dirs), 1)
		case "space":
			if len(m.dirs) > 0 {
				m.checkedDirs[m.dirs[m.dirIdx].Name] = !m.checkedDirs[m.dirs[m.dirIdx].Name]
				m.screenErr = ""
			}
		case "a":
			allSelected := len(m.selectedDirs()) == len(m.dirs)
			for _, d := range m.dirs {
				m.checkedDirs[d.Name] = !allSelected
			}
			m.screenErr = ""
		case "r":
			if m.scanErr != nil {
				m.scanErr = nil
				return m, scanDirsCmd(m.srcRoot)
			}
		case "enter":
			if len(m.dirs) == 0 {
				break
			}
			if len(m.selectedDirs()) == 0 {
				m.screenErr = "至少勾选一个目录"
				break
			}
			m.screenErr = ""
			m.screen = scrActionSelect
		}
		return m, nil
	case scrActionSelect:
		switch key {
		case "up":
			moveSelectionIndex(&m.actionIdx, len(actionNames()), -1)
		case "down":
			moveSelectionIndex(&m.actionIdx, len(actionNames()), 1)
		case "esc":
			m.screen = scrDirSelect
		case "enter":
			switch m.actionIdx {
			case actInstall:
				m.install.sysIdx = 0
				m.install.sysRule = install.RuleForSystem(install.AllSystems()[0])
				m.screen = scrInstallSystemSelect
			case actGamelist:
				return m, m.beginGamelist(m.selectedNames())
			}
		}
		return m, nil
	case scrInstallSystemSelect:
		switch key {
		case "up":
			moveSelectionIndex(&m.install.sysIdx, len(install.AllSystems()), -1)
		case "down":
			moveSelectionIndex(&m.install.sysIdx, len(install.AllSystems()), 1)
		case "esc":
			m.screen = scrActionSelect
		case "enter":
			m.install.sysRule = install.RuleForSystem(install.AllSystems()[m.install.sysIdx])
			m.screen = scrInstallDriveSelect
			m.install.driveIdx = 0
			return m, rescanDrivesCmd(m.install.showFixedDrives)
		}
		return m, nil
	case scrInstallDriveSelect:
		switch key {
		case "up":
			moveSelectionIndex(&m.install.driveIdx, len(m.install.drives), -1)
		case "down":
			moveSelectionIndex(&m.install.driveIdx, len(m.install.drives), 1)
		case "r":
			return m, rescanDrivesCmd(m.install.showFixedDrives)
		case "f":
			m.install.showFixedDrives = !m.install.showFixedDrives
			m.install.driveIdx = 0
			return m, rescanDrivesCmd(m.install.showFixedDrives)
		case "esc":
			m.screen = scrInstallSystemSelect
		case "enter":
			if len(m.install.drives) == 0 {
				break
			}
			m.screen = scrInstallConfirm
			m.screenErr = ""
		}
		return m, nil
	case scrInstallConfirm:
		switch key {
		case "enter":
			names := m.selectedNames()
			if len(names) == 0 {
				m.screenErr = "先选择至少一个目录"
				m.screen = scrDirSelect
				break
			}
			if len(m.install.drives) == 0 {
				m.screenErr = "未选择目标盘符"
				m.screen = scrInstallDriveSelect
				break
			}
			dstRoot := m.install.drives[m.install.driveIdx].Root
			return m, m.beginInstall(dstRoot, names)
		case "esc":
			m.screen = scrInstallDriveSelect
		}
		return m, nil
	case scrInstallProgress:
		if key == "esc" && m.cancel != nil {
			m.cancel()
		}
		return m, nil
	case scrInstallSummary:
		if key == "esc" {
			m.restart()
		}
		return m, nil
	case scrGamelistProgress:
		if key == "esc" {
			if m.cancel != nil {
				m.cancel()
			} else {
				m.screen = scrGamelistSummary
			}
		}
		return m, nil
	case scrGamelistSummary:
		switch key {
		case "enter":
			return m, m.beginGamelist(m.selectedNames())
		case "esc":
			m.restart()
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)

	switch m.screen {
	case scrDirSelect:
		if !m.scanned {
			return m, cmd
		}
	case scrInstallProgress:
		if m.install.progress.TotalBytes == 0 {
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) handleDirList(msg dirListMsg) (tea.Model, tea.Cmd) {
	m.dirs = msg
	m.scanned = true
	if m.dirIdx >= len(m.dirs) {
		m.dirIdx = 0
	}
	return m, nil
}

func (m Model) handleDirScanErr(msg dirScanErrMsg) (tea.Model, tea.Cmd) {
	m.scanErr = msg.err
	m.scanned = true
	return m, nil
}

func (m Model) handleDriveList(msg driveListMsg) (tea.Model, tea.Cmd) {
	m.install.drives = msg
	if m.install.driveIdx >= len(m.install.drives) {
		m.install.driveIdx = 0
	}
	return m, nil
}

func (m Model) handleInstallProgress(msg installProgressMsg) (tea.Model, tea.Cmd) {
	p := install.InstallProgress(msg)
	now := time.Now()
	if m.install.started.IsZero() {
		m.install.started = now
	}
	if now.Sub(m.install.lastRateAt) >= 400*time.Millisecond {
		dt := now.Sub(m.install.lastRateAt).Seconds()
		if dt > 0 {
			p.SpeedBps = float64(p.DoneBytes-m.install.lastDoneBytes) / dt
			p.FilesPerSec = float64(p.DoneFiles-m.install.lastDoneFiles) / dt
		}
		m.install.lastRateAt = now
		m.install.lastDoneBytes = p.DoneBytes
		m.install.lastDoneFiles = p.DoneFiles
	} else {
		p.SpeedBps = m.install.progress.SpeedBps
		p.FilesPerSec = m.install.progress.FilesPerSec
	}
	p.Elapsed = now.Sub(m.install.started)
	m.install.progress = p
	return m, nextMsgCmd(m.workerCh)
}

func (m Model) handleInstallPlanErr(msg installPlanErrMsg) (tea.Model, tea.Cmd) {
	m.cancel = nil
	m.screenErr = msg.err.Error()
	m.screen = scrInstallConfirm
	return m, nil
}

func (m Model) handleInstallDone(msg installDoneMsg) (tea.Model, tea.Cmd) {
	m.install.result = install.InstallResult(msg)
	m.cancel = nil
	m.screen = scrInstallSummary
	m.install.progress = install.InstallProgress{}
	return m, nil
}

func (m Model) handleGamelistResult(msg gamelistResultMsg) (tea.Model, tea.Cmd) {
	m.gamelist.results = append(m.gamelist.results, msg.r)
	m.gamelist.currentIdx++
	return m, nextMsgCmd(m.workerCh)
}

func (m Model) handleGamelistDone(_ gamelistDoneMsg) (tea.Model, tea.Cmd) {
	aborted := len(m.gamelist.results) < len(m.gamelist.dirNames)
	m.cancel = nil
	if aborted {
		m.screen = scrGamelistSummary
		return m, nil
	}
	wait := minGamelistDisplayTime - time.Since(m.gamelist.startedAt)
	if wait > 0 {
		return m, func() tea.Msg {
			time.Sleep(wait)
			return gamelistHoldMsg{}
		}
	}
	m.screen = scrGamelistSummary
	return m, nil
}

func (m Model) handleGamelistHold() (tea.Model, tea.Cmd) {
	if m.screen == scrGamelistProgress {
		m.screen = scrGamelistSummary
	}
	return m, nil
}

func moveSelectionIndex(current *int, total int, delta int) {
	if total <= 0 {
		*current = 0
		return
	}
	newIdx := min(max(*current+delta, 0), total-1)
	*current = newIdx
}

func (m *Model) selectedDirs() []install.DirInfo {
	var out []install.DirInfo
	for _, d := range m.dirs {
		if m.checkedDirs[d.Name] {
			out = append(out, d)
		}
	}
	return out
}

func (m *Model) selectedNames() []string {
	dirs := m.selectedDirs()
	names := make([]string, len(dirs))
	for i, d := range dirs {
		names[i] = d.Name
	}
	return names
}

func (m *Model) beginGamelist(names []string) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.resetGamelistState()
	m.gamelist.dirNames = names
	m.gamelist.startedAt = time.Now()
	m.screen = scrGamelistProgress
	m.screenErr = ""

	return tea.Batch(
		startGamelistCmd(ctx, m.srcRoot, names, m.workerCh),
		nextMsgCmd(m.workerCh),
	)
}

func (m *Model) beginInstall(dstRoot string, names []string) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.resetInstallState()
	m.install.lastRateAt = time.Now()
	m.screen = scrInstallProgress
	m.screenErr = ""

	return tea.Batch(
		startInstallCmd(ctx, cancel, m.srcRoot, dstRoot, m.install.sysRule, names, m.workerCh),
		nextMsgCmd(m.workerCh),
		spinTickCmd(m.spin),
	)
}

func (m *Model) restart() {
	m.screen = scrDirSelect
	m.checkedDirs = map[string]bool{}
	m.dirIdx = 0
	m.actionIdx = 0
	m.screenErr = ""
	m.resetInstallState()
	m.resetGamelistState()
	m.cancel = nil
}

func (m *Model) resetInstallState() {
	m.install.started = time.Time{}
	m.install.lastRateAt = time.Time{}
	m.install.lastDoneBytes = 0
	m.install.lastDoneFiles = 0
	m.install.progress = install.InstallProgress{}
	m.install.result = install.InstallResult{}
}

func (m *Model) resetGamelistState() {
	m.gamelist.dirNames = nil
	m.gamelist.currentIdx = 0
	m.gamelist.startedAt = time.Time{}
	m.gamelist.results = nil
}
