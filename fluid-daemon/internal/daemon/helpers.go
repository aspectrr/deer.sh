package daemon

import (
	"context"
	"fmt"
	"net/url"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"

	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/sourcevm"
)

// adhocSourceVMManager creates a temporary sourcevm.Manager from a SourceHostConnection.
// This allows the daemon to operate on source VMs on remote hosts even when the
// local srcVMMgr is nil or points to a different host.
func (s *Server) adhocSourceVMManager(conn *fluidv1.SourceHostConnection) (*sourcevm.Manager, error) {
	host := conn.GetSshHost()
	if host == "" {
		return nil, fmt.Errorf("ssh_host is required in source_host_connection")
	}

	// Two SSH users are involved:
	// - conn.GetSshUser() (default "fluid-daemon") connects to the remote libvirt
	//   host for virsh/qemu operations (VM listing, snapshots, disk access).
	// - "fluid-readonly" (passed to NewManager) connects to source VMs running on
	//   that host for read-only file and command access.
	user := conn.GetSshUser()
	if user == "" {
		user = "fluid-daemon"
	}

	port := conn.GetSshPort()
	if port == 0 {
		port = 22
	}

	uriHost := host
	if port != 22 {
		uriHost = fmt.Sprintf("%s:%d", host, port)
	}
	uri := fmt.Sprintf("qemu+ssh://%s@%s/system", user, uriHost)
	if s.sshIdentityFile != "" {
		uri += fmt.Sprintf("?keyfile=%s", url.QueryEscape(s.sshIdentityFile))
	}
	proxyJump := fmt.Sprintf("%s@%s", user, host)
	if port != 22 {
		proxyJump = fmt.Sprintf("%s@%s:%d", user, host, port)
	}

	return sourcevm.NewManager(uri, "default", s.keyMgr, "fluid-readonly", proxyJump, s.sshIdentityFile, s.caPubKey, s.logger), nil
}

// sourceHostConns builds SourceHostConnections from the daemon's configured source hosts.
func (s *Server) sourceHostConns() []*fluidv1.SourceHostConnection {
	conns := make([]*fluidv1.SourceHostConnection, 0, len(s.cfg.SourceHosts))
	for _, h := range s.cfg.SourceHosts {
		user := h.SSHUser
		if user == "" {
			user = "fluid-daemon"
		}
		port := h.SSHPort
		if port == 0 {
			port = 22
		}
		typ := h.Type
		if typ == "" {
			typ = "libvirt"
		}
		conns = append(conns, &fluidv1.SourceHostConnection{
			Type:    typ,
			SshHost: h.Address,
			SshPort: int32(port),
			SshUser: user,
		})
	}
	return conns
}

// resolveSourceHost looks up which configured source host owns vmName.
// It checks the cache first, then discovers VMs across all configured hosts.
func (s *Server) resolveSourceHost(ctx context.Context, vmName string) (*fluidv1.SourceHostConnection, error) {
	// Check cache
	s.vmHostMu.RLock()
	if conn, ok := s.vmHostCache[vmName]; ok {
		s.vmHostMu.RUnlock()
		return conn, nil
	}
	s.vmHostMu.RUnlock()

	// Discover across all configured source hosts
	for _, conn := range s.sourceHostConns() {
		mgr, err := s.adhocSourceVMManager(conn)
		if err != nil {
			s.logger.Warn("failed to create manager for source host", "host", conn.SshHost, "error", err)
			continue
		}
		vms, err := mgr.ListVMs(ctx)
		if err != nil {
			s.logger.Warn("failed to list VMs on source host", "host", conn.SshHost, "error", err)
			continue
		}
		s.vmHostMu.Lock()
		for _, vm := range vms {
			s.vmHostCache[vm.Name] = conn
		}
		s.vmHostMu.Unlock()

		s.vmHostMu.RLock()
		if c, ok := s.vmHostCache[vmName]; ok {
			s.vmHostMu.RUnlock()
			return c, nil
		}
		s.vmHostMu.RUnlock()
	}
	return nil, fmt.Errorf("VM %q not found on any configured source host", vmName)
}
