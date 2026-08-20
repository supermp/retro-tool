//go:build !windows

package drives

func ListDrives(includeFixed bool) []Drive {
	if includeFixed {
		return []Drive{{Root: "/tmp/retro", Label: "tmp", Type: driveFixed, Total: 1 << 30, Free: 1 << 30}}
	}
	return nil
}
