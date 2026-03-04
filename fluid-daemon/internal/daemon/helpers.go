package daemon

import (
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
