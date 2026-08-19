package drives

type Drive struct {
	Root   string
	Label  string
	Serial string
	Type   uint32
	Total  int64
	Free   int64
}
