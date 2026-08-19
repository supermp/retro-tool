package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"retro-tool/internal/drives"
	"retro-tool/pkg/gamelist"
	"retro-tool/pkg/install"
)

const (
	minGamelistDisplayTime = 3000 * time.Millisecond
)

func scanDirsCmd(src string) tea.Cmd {
	return func() tea.Msg {
		ds, err := install.ScanDirs(src)
		if err != nil {
			return dirScanErrMsg{err}
		}
		return dirListMsg(ds)
	}
}

func rescanDrivesCmd(showFixedDrives bool) tea.Cmd {
	return func() tea.Msg { return driveListMsg(drives.ListDrives(showFixedDrives)) }
}

func nextMsgCmd(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg { return <-ch }
}

func spinTickCmd(s spinner.Model) tea.Cmd {
	return func() tea.Msg { return s.Tick() }
}

func startInstallCmd(ctx context.Context, cancel context.CancelFunc, dstRoot string, m Model, names []string) tea.Cmd {
	return func() tea.Msg {
		ch := m.workerCh
		go func() {
			defer func() {
				if r := recover(); r != nil {
					cancel()
					sendWorkerMsg(ch, installPlanErrMsg{err: fmt.Errorf("安装过程异常: %v", r)})
				}
			}()
			plan, err := install.BuildPlan(m.srcRoot, dstRoot, m.install.sysRule, names)
			if err != nil {
				cancel()
				sendWorkerMsg(ch, installPlanErrMsg{err})
				return
			}
			installer := install.NewInstaller(ctx, m.install.sysRule, plan, func(p install.InstallProgress) {
				sendWorkerMsg(ch, installProgressMsg(p))
			})
			res := installer.Run()
			sendWorkerMsg(ch, installDoneMsg(res))
		}()
		return nil
	}
}

func startGamelistCmd(ctx context.Context, m Model, names []string) tea.Cmd {
	return func() tea.Msg {
		ch := m.workerCh
		go func() {
			defer func() {
				if r := recover(); r != nil {
					sendWorkerMsg(ch, gamelistResultMsg{r: gamelist.Result{Dir: "[错误]", Failed: []string{fmt.Sprintf("Gamelist 生成异常: %v", r)}}})
				}
			}()
			deadline := time.Now().Add(minGamelistDisplayTime)
			budget := func() time.Duration {
				if d := time.Until(deadline); d > 0 {
					return d
				}
				return 0
			}
			sleep := func(d time.Duration) bool {
				select {
				case <-time.After(d):
					return true
				case <-ctx.Done():
					return false
				}
			}

			completed := 0
			onProgress := func(p gamelist.Progress) {
				if p.Total > 0 && p.Current >= p.Total && completed < len(names) {
					remaining := len(names) - completed
					if remaining > 0 {
						sleep(budget() / time.Duration(remaining))
					}
				}
			}
			for _, name := range names {
				if err := ctx.Err(); err != nil {
					break
				}
				dir := filepath.Join(m.srcRoot, name)
				res, err := gamelist.Generate(dir, onProgress)
				completed++
				if err != nil {
					res = gamelist.Result{Dir: name, Failed: []string{err.Error()}}
				} else {
					res.Dir = name
				}
				sendWorkerMsg(ch, gamelistResultMsg{r: res})
			}
			sleep(budget())
			sendWorkerMsg(ch, gamelistDoneMsg{})
		}()
		return nil
	}
}

func sendWorkerMsg(ch chan tea.Msg, msg tea.Msg) {
	if ch == nil {
		return
	}

	if _, isProgress := msg.(installProgressMsg); isProgress {
		select {
		case ch <- msg:
		default:
		}
		return
	}

	for {
		select {
		case ch <- msg:
			return
		default:
			select {
			case <-ch:
			case <-time.After(5 * time.Millisecond):
			}
		}
	}
}
