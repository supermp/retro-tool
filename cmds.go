package main

import (
	"context"
	"fmt"
	"path/filepath"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"retro-tool/internal/drives"
	"retro-tool/internal/gamelist"
	"retro-tool/internal/install"
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

func startInstallCmd(ctx context.Context, cancel context.CancelFunc, srcRoot, dstRoot string, rule install.InstallRule, names []string, ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					cancel()
					sendWorkerMsg(ch, installPlanErrMsg{err: fmt.Errorf("安装过程异常: %v", r)})
				}
			}()
			plan, err := install.BuildPlan(srcRoot, dstRoot, rule, names)
			if err != nil {
				cancel()
				sendWorkerMsg(ch, installPlanErrMsg{err})
				return
			}
			installer := install.NewInstaller(ctx, rule, plan, func(p install.InstallProgress) {
				sendWorkerMsg(ch, installProgressMsg(p))
			})
			res := installer.Run()
			sendWorkerMsg(ch, installDoneMsg(res))
		}()
		return nil
	}
}

func startGamelistCmd(ctx context.Context, srcRoot string, names []string, ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					sendWorkerMsg(ch, gamelistResultMsg{r: gamelist.Result{Dir: "[错误]", Failed: []string{fmt.Sprintf("Gamelist 生成异常: %v", r)}}})
				}
			}()
			for _, name := range names {
				if err := ctx.Err(); err != nil {
					break
				}
				dir := filepath.Join(srcRoot, name)
				res, err := gamelist.Generate(dir)
				if err != nil {
					res = gamelist.Result{Dir: name, Failed: []string{err.Error()}}
				} else {
					res.Dir = name
				}
				sendWorkerMsg(ch, gamelistResultMsg{r: res})
			}
			sendWorkerMsg(ch, gamelistDoneMsg{})
		}()
		return nil
	}
}

func sendWorkerMsg(ch chan tea.Msg, msg tea.Msg) {
	if ch == nil {
		return
	}
	select {
	case ch <- msg:
		return
	default:
	}
	if _, isProgress := msg.(installProgressMsg); isProgress {
		return
	}
	select {
	case <-ch:
	default:
	}
	ch <- msg
}
