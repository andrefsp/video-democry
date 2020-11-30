package netutils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNetUtils_getRelayAddr(t *testing.T) {
	addr, err := GetRelayAddr()
	assert.Nil(t, err)
	assert.NotEqual(t, addr, 0)
}

func TestNetUtils_getIFaces(t *testing.T) {
	ifaces, err := GetIFaces()
	assert.Nil(t, err)
	assert.GreaterOrEqual(t, len(ifaces), 1)
}
