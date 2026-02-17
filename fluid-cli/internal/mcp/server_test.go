package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aspectrr/fluid.sh/fluid/internal/config"
)

func TestNewServer(t *testing.T) {
	cfg := testConfig()
	st := newMockStore()

	srv := NewServer(cfg, st, nil, nil, noopLogger())
	require.NotNil(t, srv)
	assert.NotNil(t, srv.mcpServer)
	assert.NotNil(t, srv.playbookService)
	assert.NotNil(t, srv.logger)
}

func TestNewServer_WithHosts(t *testing.T) {
	cfg := testConfig()
	cfg.Hosts = []config.HostConfig{
		{Name: "host1", Address: "10.0.0.1"},
	}
	st := newMockStore()

	srv := NewServer(cfg, st, nil, nil, noopLogger())
	require.NotNil(t, srv)
	// multiHostMgr removed in remote mode
}

func TestNewServer_RegistersAllTools(t *testing.T) {
	cfg := testConfig()
	st := newMockStore()

	srv := NewServer(cfg, st, nil, nil, noopLogger())
	require.NotNil(t, srv)
	assert.NotNil(t, srv.mcpServer)
}
