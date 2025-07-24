package json

import (
	"errors"
	"slices"
)

var (
	ErrNotFound = errors.New("search: not found")
)

// Can only handle field in object
func Search(data []byte, path string) ([]byte, error) {
	scanner := NewScanner(data)
	target := []byte(path)

	for {
		token, err := scanner.Next()
		if err != nil {
			return nil, err
		}

		if token.Type == TOKEN_TYPE_END_OF_DOCUMENT {
			return nil, ErrNotFound
		}

		if slices.Compare(token.Value, target) == 0 {
			return scanner.ExtractValue()
		}
	}
}
