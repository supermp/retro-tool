package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	os.Exit(runTUI(wd))
}

func runTUI(src string) int {
	m := initialModel(src)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "程序运行出错:", err)
		return 1
	}
	return 0
}
