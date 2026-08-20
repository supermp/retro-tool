package main

import (
	"context"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"retro-tool/internal/drives"
	"retro-tool/internal/gamelist"
	"retro-tool/internal/install"
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

type Model struct {
	screen      Screen
	width       int
	screenErr   string
	spin        spinner.Model
	progressBar progress.Model

	srcRoot     string
	scanErr     error
	dirs        []install.DirInfo
	checkedDirs map[string]bool
	dirIdx      int
	scanned     bool

	actionIdx int
	install   installState
	gamelist  gamelistState

	workerCh chan tea.Msg
	cancel   context.CancelFunc
}

type installState struct {
	sysIdx          int
	sysRule         install.InstallRule
	showFixedDrives bool
	drives          []drives.Drive
	driveIdx        int
	progress        install.InstallProgress
	result          install.InstallResult
	started         time.Time
	lastRateAt      time.Time
	lastDoneBytes   int64
	lastDoneFiles   int64
}

type gamelistState struct {
	dirNames   []string
	currentIdx int
	startedAt  time.Time
	results    []gamelist.Result
}

func initialModel(src string) Model {
	return Model{
		screen:      scrDirSelect,
		spin:        spinner.New(spinner.WithSpinner(spinner.Points)),
		srcRoot:     src,
		checkedDirs: map[string]bool{},
		progressBar: progress.New(
			progress.WithWidth(40),
			progress.WithFillCharacters('█', '░'),
			progress.WithDefaultBlend(),
			progress.WithoutPercentage(),
		),
		workerCh: make(chan tea.Msg, 256),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		scanDirsCmd(m.srcRoot),
		rescanDrivesCmd(m.install.showFixedDrives),
		spinTickCmd(m.spin),
	)
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
