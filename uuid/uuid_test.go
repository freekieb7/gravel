package uuid_test

import (
	"testing"

	"github.com/freekieb7/gravel/uuid"
)

func TestUUIDConversion(t *testing.T) {
	id := uuid.NewV4()
	idStr := id.String()

	idParsed, err := uuid.Parse(idStr)
	if err != nil {
		t.Fatal(err)
	}

	if id != idParsed {
		t.Error("parse failed")
	}
}

func BenchmarkUUIDToString(b *testing.B) {
	for range b.N {
		id := uuid.NewV4()
		idStr := id.String()
		uuid.Parse(idStr)
	}
}
