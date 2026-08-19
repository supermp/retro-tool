package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SystemType string

const (
	SystemAurknix  SystemType = "Aurknix / Rocknix"
	SystemAnbernic SystemType = "Anbernic (Linux)"
)

const anbernicRootDir = "Roms"

type InstallRule interface {
	DirName(src string) string
	FileName(src string) string
	PostProcess(dstDir string) error
	RootDir(base string) string
}

func AllSystems() []SystemType { return []SystemType{SystemAurknix, SystemAnbernic} }

func RuleForSystem(s SystemType) InstallRule {
	switch s {
	case SystemAurknix:
		return aurknixRule{}
	case SystemAnbernic:
		return anbernicRule{}
	default:
		return aurknixRule{}
	}
}

type aurknixRule struct{}

func (aurknixRule) DirName(src string) string {
	switch strings.ToUpper(src) {
	case "MD":
		return "megadrive"
	case "PS":
		return "psx"
	case "FC":
		return "famicom"
	default:
		return strings.ToLower(src)
	}
}

func (aurknixRule) FileName(src string) string { return src }

func (aurknixRule) PostProcess(dstDir string) error { return nil }

func (aurknixRule) RootDir(base string) string { return base }

type anbernicRule struct{}

func (anbernicRule) DirName(src string) string { return strings.ToUpper(src) }

func (anbernicRule) FileName(src string) string { return src }

func (anbernicRule) RootDir(base string) string { return filepath.Join(base, anbernicRootDir) }

func (anbernicRule) PostProcess(dstDir string) error {
	if err := os.Remove(filepath.Join(dstDir, "gamelist.xml")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除 gamelist.xml 失败: %w", err)
	}

	boxart := filepath.Join(dstDir, "boxart")
	imgs := filepath.Join(dstDir, "Imgs")
	if err := os.Rename(boxart, imgs); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("boxart 改名为 Imgs 失败: %w", err)
	}

	if strings.EqualFold(filepath.Base(dstDir), "FBNEO") {
		if err := flattenDir(dstDir, "Imgs"); err != nil {
			return fmt.Errorf("拍平 FBNEO 失败: %w", err)
		}
		if err := flattenDir(imgs, ""); err != nil {
			return fmt.Errorf("拍平 FBNEO/Imgs 失败: %w", err)
		}
	}

	return nil
}

func flattenDir(dir, skip string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	moves := make(map[string]string)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == skip {
			continue
		}
		sub := filepath.Join(dir, name)
		if err := collectFiles(sub, dir, moves); err != nil {
			return err
		}
	}

	for src, dst := range moves {
		if src == dst {
			continue
		}
		if err := os.Rename(src, dst); err != nil {
			if errors.Is(err, os.ErrExist) {
				return fmt.Errorf("文件名冲突: %s", dst)
			}
			return err
		}
	}

	return removeEmptySubdirs(dir, skip)
}

func collectFiles(subDir, rootDir string, moves map[string]string) error {
	return filepath.WalkDir(subDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || path == subDir {
			return nil
		}
		moves[path] = filepath.Join(rootDir, d.Name())
		return nil
	})
}

func removeEmptySubdirs(dir, skip string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == skip {
			continue
		}
		sub := filepath.Join(dir, e.Name())
		if err := removeEmptySubdirs(sub, ""); err != nil {
			return err
		}
		if err := os.Remove(sub); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
