package drives

const (
	driveUnknown   = 0
	driveNoRoot    = 1
	driveRemovable = 2
	driveFixed     = 3
	driveRemote    = 4
	driveCDROM     = 5
	driveRAMDisk   = 6
)

type Drive struct {
	Root  string
	Label string
	Type  uint32
	Total int64
	Free  int64
}

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
