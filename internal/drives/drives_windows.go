//go:build windows

package drives

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	driveUnknown   = 0
	driveNoRoot    = 1
	driveRemovable = 2
	driveFixed     = 3
	driveRemote    = 4
	driveCDROM     = 5
	driveRAMDisk   = 6
)

func (d Drive) IsRemovable() bool { return d.Type == driveRemovable }

func (d Drive) TypeName() string {
	switch d.Type {
	case driveRemovable:
		return "可移动磁盘"
	case driveFixed:
		return "本地磁盘"
	case driveRemote:
		return "网络驱动器"
	case driveCDROM:
		return "光驱"
	default:
		return "未知类型"
	}
}

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	pGetDriveType         = kernel32.NewProc("GetDriveTypeW")
	pGetDiskFreeSpaceEx   = kernel32.NewProc("GetDiskFreeSpaceExW")
	pGetVolumeInformation = kernel32.NewProc("GetVolumeInformationW")
)

func ListDrives(includeFixed bool) []Drive {
	var out []Drive
	for c := 'A'; c <= 'Z'; c++ {
		root := fmt.Sprintf("%c:\\", c)
		rp, err := syscall.UTF16PtrFromString(root)
		if err != nil {
			continue
		}
		t, _, _ := pGetDriveType.Call(uintptr(unsafe.Pointer(rp)))
		typ := uint32(t)
		if typ != driveRemovable && typ != driveFixed {
			continue
		}
		if typ == driveFixed && !includeFixed {
			continue
		}
		d := Drive{Root: root, Type: typ}

		nameBuf := make([]uint16, 256)
		fsBuf := make([]uint16, 64)
		var serial, maxLen, flags uint32
		_, _, _ = pGetVolumeInformation.Call(
			uintptr(unsafe.Pointer(rp)),
			uintptr(unsafe.Pointer(&nameBuf[0])), uintptr(len(nameBuf)),
			uintptr(unsafe.Pointer(&serial)),
			uintptr(unsafe.Pointer(&maxLen)),
			uintptr(unsafe.Pointer(&flags)),
			uintptr(unsafe.Pointer(&fsBuf[0])), uintptr(len(fsBuf)),
		)
		if nameBuf[0] != 0 {
			d.Label = syscall.UTF16ToString(nameBuf)
		}
		if serial != 0 {
			d.Serial = fmt.Sprintf("%08X", serial)
		}

		var freeAvail, total, totalFree uint64
		_, _, _ = pGetDiskFreeSpaceEx.Call(
			uintptr(unsafe.Pointer(rp)),
			uintptr(unsafe.Pointer(&freeAvail)),
			uintptr(unsafe.Pointer(&total)),
			uintptr(unsafe.Pointer(&totalFree)),
		)
		d.Free, d.Total = int64(freeAvail), int64(total)

		out = append(out, d)
	}
	return out
}
