package gamelist

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	gamelistReportStep = 30
)

type Game struct {
	Path  string `xml:"path"`
	Name  string `xml:"name"`
	Image string `xml:"image"`
}

type Gamelist struct {
	XMLName xml.Name `xml:"gameList"`
	Games   []Game   `xml:"game"`
}

type Progress struct {
	Current int
	Total   int
	Success int
	Failed  int
}

type Result struct {
	Dir     string
	Total   int
	Success int
	Failed  []string
	Skipped bool
}

func WriteGamelist(dir string, games []Game) error {
	path := filepath.Join(dir, "gamelist.xml")

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	bw := bufio.NewWriter(f)
	defer bw.Flush()

	if _, err := bw.WriteString("<?xml version=\"1.0\"?>\n"); err != nil {
		return err
	}

	enc := xml.NewEncoder(bw)
	enc.Indent("", "\t")
	if err := enc.Encode(Gamelist{Games: games}); err != nil {
		return err
	}
	if err := enc.Flush(); err != nil {
		return err
	}
	if _, err := bw.WriteString("\n"); err != nil {
		return err
	}
	return bw.Flush()
}

func Generate(dir string, report func(Progress)) (Result, error) {
	if isArcadeDir(dir) {
		return Result{Skipped: true}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return Result{}, fmt.Errorf("读取目录失败: %w", err)
	}

	var files []os.DirEntry
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		name := e.Name()
		if strings.EqualFold(name, "gamelist.xml") {
			continue
		}
		if isBiosName(name) {
			continue
		}
		files = append(files, e)
	}

	boxartMap, err := loadBoxartMap(dir)
	if err != nil {
		return Result{}, fmt.Errorf("读取 boxart 目录失败: %w", err)
	}

	total := len(files)
	games := make([]Game, 0, total)
	failed := make([]string, 0)

	emit := func(current, success, failedCount int) {
		if report != nil {
			report(Progress{Current: current, Total: total, Success: success, Failed: failedCount})
		}
	}

	emit(0, 0, 0)
	lastReport := 0

	for i, f := range files {
		name := f.Name()
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)

		imgRel, ok := boxartMap[strings.ToLower(base)]
		if ok {
			games = append(games, Game{
				Path:  "./" + name,
				Name:  base,
				Image: imgRel,
			})
		} else {
			failed = append(failed, name)
		}
		current := i + 1
		if current == total || current-lastReport >= gamelistReportStep {
			emit(current, len(games), len(failed))
			lastReport = current
		}
	}

	if err := WriteGamelist(dir, games); err != nil {
		return Result{}, fmt.Errorf("写入 Gamelist 失败: %w", err)
	}

	return Result{
		Dir:     dir,
		Total:   total,
		Success: len(games),
		Failed:  failed,
	}, nil
}

func loadBoxartMap(dir string) (map[string]string, error) {
	boxart := filepath.Join(dir, "boxart")
	entries, err := os.ReadDir(boxart)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	prio := map[string]int{".png": 3, ".jpg": 2, ".jpeg": 1}
	m := make(map[string]string)
	score := make(map[string]int)

	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		p, ok := prio[ext]
		if !ok {
			continue
		}
		base := strings.TrimSuffix(name, ext)
		key := strings.ToLower(base)
		if p > score[key] {
			score[key] = p
			m[key] = "./boxart/" + name
		}
	}
	return m, nil
}

func isArcadeDir(dir string) bool {
	name := strings.ToUpper(filepath.Base(dir))
	return strings.Contains(name, "FBNEO") || strings.Contains(name, "MAME")
}

func isBiosName(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".bios")
}
