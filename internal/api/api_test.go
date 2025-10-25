package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlwaysTrue(t *testing.T) {
	assert.True(t, true, "Never failes")
}
