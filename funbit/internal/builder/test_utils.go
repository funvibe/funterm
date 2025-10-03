package builder

// CustomInt interface for testing reflection paths
type CustomInt interface {
	ToInt() int
}

// MyInt implements CustomInt interface for testing
type MyInt struct {
	val int
}

func (m MyInt) ToInt() int {
	return m.val
}
