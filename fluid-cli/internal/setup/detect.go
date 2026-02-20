package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/aspectrr/fluid.sh/fluid/internal/hostexec"
)

// DistroInfo holds detected OS distribution information.
type DistroInfo struct {
	ID         string // e.g. "ubuntu", "debian", "fedora", "rocky"
	Name       string // e.g. "Ubuntu 22.04"
	PkgManager string // "apt" or "dnf"
}

// DetectOS reads /etc/os-release to determine the distribution and package manager.
func DetectOS(ctx context.Context, run hostexec.RunFunc) (DistroInfo, error) {
	stdout, _, code, err := run(ctx, "cat /etc/os-release 2>/dev/null")
	if err != nil || code != 0 {
		return DistroInfo{}, fmt.Errorf("cannot read /etc/os-release: host may not be Linux")
	}

	info := DistroInfo{}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if k, v, ok := strings.Cut(line, "="); ok {
			v = strings.Trim(v, "\"")
			switch k {
			case "ID":
				info.ID = v
			case "PRETTY_NAME":
				info.Name = v
			}
		}
	}

	switch info.ID {
	case "ubuntu", "debian", "linuxmint", "pop":
		info.PkgManager = "apt"
	case "fedora", "rocky", "almalinux", "centos", "rhel":
		info.PkgManager = "dnf"
	default:
		return info, fmt.Errorf("unsupported distribution %q - see https://fluid.sh/docs/daemon for manual setup", info.ID)
	}

	return info, nil
}
