package install

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	fileCopyBufferSize     = 512 * 1024
	progressReportInterval = 80 * time.Millisecond
	maxReportedErrors      = 20
	fileSyncInterval       = 10 * 1024 * 1024
)

type DirInfo struct {
	Name  string
	Files int64
	Bytes int64
}

func ScanDirs(srcRoot string) ([]DirInfo, error) {
	entries, err := os.ReadDir(srcRoot)
	if err != nil {
		return nil, err
	}
	var out []DirInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := DirInfo{Name: e.Name()}
		walkErr := filepath.WalkDir(filepath.Join(srcRoot, e.Name()), func(_ string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			info, ierr := d.Info()
			if ierr != nil {
				return ierr
			}
			p.Files++
			p.Bytes += info.Size()
			return nil
		})
		if walkErr != nil {
			return nil, fmt.Errorf("扫描目录 %s 失败: %w", e.Name(), walkErr)
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

type planFile struct {
	Src       string
	Dst       string
	Size      int64
	BytesDone int64
}

type planDir struct {
	SrcName string
	DstName string
	SrcDir  string
	DstDir  string
	Files   []planFile
	Bytes   int64
}

func BuildPlan(srcRoot, dstRoot string, rule InstallRule, names []string) ([]planDir, error) {
	var plans []planDir
	seen := map[string]string{}
	for _, name := range names {
		dstName := rule.DirName(name)
		if prev, dup := seen[dstName]; dup {
			return nil, fmt.Errorf("规则映射后目录重名：%s 与 %s 都会变成 %s，先处理源目录冲突", prev, name, dstName)
		}
		seen[dstName] = name

		srcDir := filepath.Join(srcRoot, name)
		dstRootForRule := rule.RootDir(dstRoot)
		dstDir := filepath.Join(dstRootForRule, dstName)
		plan := planDir{SrcName: name, DstName: dstName, SrcDir: srcDir, DstDir: dstDir}

		err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, rerr := filepath.Rel(srcDir, path)
			if rerr != nil {
				return rerr
			}
			info, ierr := d.Info()
			if ierr != nil {
				return ierr
			}
			plan.Files = append(plan.Files, planFile{
				Src:  path,
				Dst:  filepath.Join(plan.DstDir, transformRel(rule, rel)),
				Size: info.Size(),
			})
			plan.Bytes += info.Size()
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("扫描 %s 失败: %w", name, err)
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

func transformRel(rule InstallRule, rel string) string {
	parts := strings.Split(rel, string(filepath.Separator))
	for i, p := range parts {
		if i < len(parts)-1 {
			parts[i] = rule.DirName(p)
		} else {
			parts[i] = rule.FileName(p)
		}
	}
	return filepath.Join(parts...)
}

type Throttle struct {
	Interval time.Duration
	last     time.Time
}

func (t *Throttle) Allow(now time.Time) bool {
	if now.Sub(t.last) >= t.Interval {
		t.last = now
		return true
	}
	return false
}

type InstallProgress struct {
	TotalBytes  int64
	DoneBytes   int64
	TotalFiles  int64
	DoneFiles   int64
	DirIdx      int
	DirTotal    int
	CurFile     string
	CurDstDir   string
	DirFiles    int64
	DirBytes    int64
	SpeedBps    float64
	FilesPerSec float64
	Elapsed     time.Duration
}

type DirResult struct {
	SrcName string
	DstName string
	Files   int64
	Bytes   int64
}

type InstallResult struct {
	PerDir  []DirResult
	Errors  []string
	Files   int64
	Bytes   int64
	Aborted bool
}

func NewInstaller(ctx context.Context, rule InstallRule, plans []planDir, emit func(InstallProgress)) *installer {
	th := &Throttle{Interval: progressReportInterval}
	throttled := func(p InstallProgress) {
		if th.Allow(time.Now()) {
			emit(p)
		}
	}
	c := &installer{ctx: ctx, rule: rule, plans: plans, emit: throttled, buf: make([]byte, fileCopyBufferSize)}
	for i := range plans {
		c.totalBytes += plans[i].Bytes
		c.totalFiles += int64(len(plans[i].Files))
	}
	return c
}

type installer struct {
	ctx        context.Context
	plans      []planDir
	rule       InstallRule
	emit       func(InstallProgress)
	buf        []byte
	totalBytes int64
	totalFiles int64
}

func (c *installer) Run() InstallResult {
	prog := InstallProgress{DirTotal: len(c.plans), TotalFiles: c.totalFiles, TotalBytes: c.totalBytes}
	var res InstallResult
	c.emit(prog)

	for pi := range c.plans {
		p := &c.plans[pi]
		if cerr := c.ctx.Err(); cerr != nil {
			res.Aborted = true
			break
		}
		prog.DirIdx = pi
		prog.DirFiles = int64(len(p.Files))
		prog.DirBytes = 0
		for i := range p.Files {
			prog.DirBytes += p.Files[i].Size
		}
		c.emit(prog)

		dirReady := true
		if p.DstName != "" && p.DstName != "." && p.DstName != ".." && p.DstDir != p.SrcDir {
			if err := os.RemoveAll(p.DstDir); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("%s: 清空旧目录失败: %v", p.DstDir, err))
				dirReady = false
			}
		}
		if dirReady {
			if err := os.MkdirAll(p.DstDir, 0o755); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("%s: 创建目录失败: %v", p.DstDir, err))
				dirReady = false
			}
		}

		var dirFiles int64
		if dirReady {
			for fi := range p.Files {
				if cerr := c.ctx.Err(); cerr != nil {
					res.Aborted = true
					break
				}
				f := &p.Files[fi]
				installErr := c.installOne(f, &prog)
				if installErr != nil {
					_ = os.Remove(f.Dst)
					f.BytesDone = 0
					if cerr := c.ctx.Err(); cerr != nil {
						_ = cerr
						res.Aborted = true
						break
					}
					res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", f.Src, installErr))
				}
				dirFiles++
				if installErr == nil {
					prog.DoneFiles++
				}
				c.emit(prog)
			}
			if err := c.rule.PostProcess(p.DstDir); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("%s: 后处理失败: %v", p.DstDir, err))
			}
		}

		res.PerDir = append(res.PerDir, DirResult{
			SrcName: p.SrcName, DstName: p.DstName, Files: dirFiles, Bytes: dirBytes(p),
		})
		if res.Aborted {
			break
		}
	}

	for _, r := range res.PerDir {
		res.Files += r.Files
		res.Bytes += r.Bytes
	}
	if len(res.Errors) > maxReportedErrors {
		truncated := len(res.Errors) - maxReportedErrors
		res.Errors = append(res.Errors[:maxReportedErrors], fmt.Sprintf("…… 其余 %d 个错误未列出", truncated))
	}
	c.emit(prog)
	return res
}

func dirBytes(p *planDir) int64 {
	var n int64
	for i := range p.Files {
		n += p.Files[i].BytesDone
	}
	return n
}

func (c *installer) installOne(f *planFile, prog *InstallProgress) error {
	src, err := os.Open(f.Src)
	if err != nil {
		return err
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(f.Dst), 0o755); err != nil {
		return err
	}
	dst, err := os.Create(f.Dst)
	if err != nil {
		return err
	}
	defer dst.Close()

	prog.CurFile = f.Src
	prog.CurDstDir = filepath.Dir(f.Dst)
	c.emit(*prog)

	syncedBytes := int64(0)
	for {
		if cerr := c.ctx.Err(); cerr != nil {
			return cerr
		}
		n, rerr := src.Read(c.buf)
		if n > 0 {
			if _, werr := dst.Write(c.buf[:n]); werr != nil {
				return werr
			}
			f.BytesDone += int64(n)
			prog.DoneBytes += int64(n)
			syncedBytes += int64(n)
			if syncedBytes >= fileSyncInterval {
				if werr := dst.Sync(); werr != nil {
					return werr
				}
				syncedBytes = 0
			}
			c.emit(*prog)
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				break
			}
			return rerr
		}
	}
	if werr := dst.Sync(); werr != nil {
		return werr
	}
	if written := f.BytesDone; written != f.Size {
		return fmt.Errorf("实际写入 %d 字节，与源文件大小 %d 不一致（源文件可能在安装中被修改）", written, f.Size)
	}
	return nil
}
