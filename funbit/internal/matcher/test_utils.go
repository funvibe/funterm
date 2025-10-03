package matcher

import (
	"bytes"
)

func bytesEqual(a, b []byte) bool {
	return bytes.Equal(a, b)
}
