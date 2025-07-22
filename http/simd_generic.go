//go:build !amd64

package http

import "bytes"

func toLowerSIMD(data []byte) {
	toLowerScalar(data)
}

func equalsFast(a, b []byte) bool {
	return bytes.Equal(a, b)
}

func toLowerScalar(data []byte) {
	for i := range data {
		if data[i] >= 'A' && data[i] <= 'Z' {
			data[i] += 'a' - 'A'
		}
	}
}
