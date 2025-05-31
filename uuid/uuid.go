package uuid

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
)

var (
	ErrInvalidFormat = errors.New("uuid: invalid format")
)

type UUID [16]byte

func NewV4() UUID {
	var uuid UUID

	_, err := rand.Read(uuid[:])
	if err != nil {
		return uuid
	}

	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10

	return uuid
}

func Parse(s string) (UUID, error) {
	var uuid UUID

	switch len(s) {
	case 36:
	case 36 + 9:
		{
			if !strings.EqualFold(s[:9], "urn:uuid:") {
				return uuid, ErrInvalidFormat
			}

			s = s[9:]
		}
	default:
		{
			return uuid, ErrInvalidFormat
		}
	}

	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return uuid, ErrInvalidFormat
	}

	hex.Decode(uuid[:4], []byte(s[:8]))
	hex.Decode(uuid[4:6], []byte(s[9:13]))
	hex.Decode(uuid[6:8], []byte(s[14:18]))
	hex.Decode(uuid[8:10], []byte(s[19:23]))
	hex.Decode(uuid[10:], []byte(s[24:]))

	return uuid, nil
}

func (uuid UUID) String() string {
	var buf [36]byte

	hex.Encode(buf[:], uuid[:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], uuid[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], uuid[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], uuid[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], uuid[10:])

	return string(buf[:])
}

func (uuid UUID) Version() byte {
	return uuid[6] >> 4
}
