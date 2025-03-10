package test_test

import (
	"testing"

	"github.com/freekieb7/gravel/test"
)

func TestAssertTrue(t *testing.T) {
	test.AssertTrue(t, 1, 1)
}
