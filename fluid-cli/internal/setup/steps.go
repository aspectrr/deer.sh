package setup

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/aspectrr/fluid.sh/fluid/internal/hostexec"
)

func allSteps(distro DistroInfo) []StepDef {
	return []StepDef{
		stepInstallPrereqs(distro),
		stepDownloadDaemon(),
		stepCreateUserDirs(),
		stepGenerateConfig(),
		stepCreateSystemdUnit(),
		stepEnableAndStart(),
	}
}

func installPrereqCommands(distro DistroInfo) []string {
	switch distro.PkgManager {
	case "apt":
		return []string{
			"apt-get update -qq",
			"apt-get install -y qemu-system-x86 libvirt-daemon-system libvirt-clients",
		}
	case "dnf":
		return []string{
			"dnf install -y qemu-kvm libvirt libvirt-client",
		}
	default:
		return []string{"(unsupported package manager)"}
	}
}

func stepInstallPrereqs(distro DistroInfo) StepDef {
	return StepDef{
		Name:        "Install prerequisites",
		Description: "Install libvirt, QEMU, and other required packages",
		Commands:    installPrereqCommands(distro),
		Check: func(ctx context.Context, run hostexec.RunFunc) (bool, error) {
			_, _, code, _ := run(ctx, "which qemu-system-x86_64 >/dev/null 2>&1 && which virsh >/dev/null 2>&1")
			return code == 0, nil
		},
		Execute: func(ctx context.Context, sudoRun hostexec.RunFunc) error {
			var cmd string
			switch distro.PkgManager {
			case "apt":
				cmd = "apt-get update -qq && apt-get install -y -qq qemu-system-x86 libvirt-daemon-system libvirt-clients"
			case "dnf":
				cmd = "dnf install -y qemu-kvm libvirt libvirt-client"
			default:
				return fmt.Errorf("unsupported package manager: %s", distro.PkgManager)
			}
			_, stderr, code, err := sudoRun(ctx, cmd)
			if err != nil {
				return fmt.Errorf("install prerequisites: %w", err)
			}
			if code != 0 {
				return fmt.Errorf("install prerequisites failed (exit %d): %s", code, stderr)
			}
			return nil
		},
	}
}

func stepDownloadDaemon() StepDef {
	arch := runtime.GOARCH
	return StepDef{
		Name:        "Download fluid-daemon",
		Description: "Download the fluid-daemon binary",
		Commands: []string{
			fmt.Sprintf("curl -fsSL -o /usr/local/bin/fluid-daemon https://github.com/aspectrr/fluid.sh/releases/latest/download/fluid-daemon-linux-%s", arch),
			"chmod +x /usr/local/bin/fluid-daemon",
		},
		Check: func(ctx context.Context, run hostexec.RunFunc) (bool, error) {
			_, _, code, _ := run(ctx, "which fluid-daemon >/dev/null 2>&1")
			return code == 0, nil
		},
		Execute: func(ctx context.Context, sudoRun hostexec.RunFunc) error {
			arch := runtime.GOARCH
			cmd := fmt.Sprintf(
				"curl -fsSL -o /usr/local/bin/fluid-daemon https://github.com/aspectrr/fluid.sh/releases/latest/download/fluid-daemon-linux-%s && chmod +x /usr/local/bin/fluid-daemon",
				arch,
			)
			_, stderr, code, err := sudoRun(ctx, cmd)
			if err != nil {
				return fmt.Errorf("download fluid-daemon: %w", err)
			}
			if code != 0 {
				return fmt.Errorf("download failed (exit %d): %s", code, stderr)
			}
			return nil
		},
	}
}

func stepCreateUserDirs() StepDef {
	return StepDef{
		Name:        "Create system user and directories",
		Description: "Create the fluid-daemon system user and storage directories",
		Commands: []string{
			"useradd -r -s /usr/sbin/nologin -d /var/lib/fluid -M fluid-daemon",
			"mkdir -p /var/lib/fluid/images /var/lib/fluid/sandboxes /etc/fluid-daemon",
			"chown -R fluid-daemon:fluid-daemon /var/lib/fluid",
		},
		Check: func(ctx context.Context, run hostexec.RunFunc) (bool, error) {
			_, _, code, _ := run(ctx, "id fluid-daemon >/dev/null 2>&1")
			return code == 0, nil
		},
		Execute: func(ctx context.Context, sudoRun hostexec.RunFunc) error {
			cmds := []string{
				"id fluid-daemon >/dev/null 2>&1 || useradd -r -s /usr/sbin/nologin -d /var/lib/fluid -M fluid-daemon",
				"mkdir -p /var/lib/fluid/images /var/lib/fluid/sandboxes /etc/fluid-daemon",
				"chown -R fluid-daemon:fluid-daemon /var/lib/fluid",
			}
			cmd := strings.Join(cmds, " && ")
			_, stderr, code, err := sudoRun(ctx, cmd)
			if err != nil {
				return fmt.Errorf("create user/dirs: %w", err)
			}
			if code != 0 {
				return fmt.Errorf("create user/dirs failed (exit %d): %s", code, stderr)
			}
			return nil
		},
	}
}

func stepGenerateConfig() StepDef {
	return StepDef{
		Name:        "Generate daemon config",
		Description: "Write default daemon configuration file",
		Commands: []string{
			"Write /etc/fluid-daemon/daemon.yaml (default config)",
		},
		Check: func(ctx context.Context, run hostexec.RunFunc) (bool, error) {
			_, _, code, _ := run(ctx, "test -f /etc/fluid-daemon/daemon.yaml")
			return code == 0, nil
		},
		Execute: func(ctx context.Context, sudoRun hostexec.RunFunc) error {
			configYAML := `listen:
  grpc: ":9091"

backend: qemu

storage:
  images: /var/lib/fluid/images
  overlays: /var/lib/fluid/sandboxes

network:
  bridge: virbr0
  subnet: 10.0.0.0/24

ssh:
  ca_key_path: /etc/fluid-daemon/ssh_ca
  ca_pub_key_path: /etc/fluid-daemon/ssh_ca.pub
  key_dir: /etc/fluid-daemon/keys
  cert_ttl: 30m
  default_user: sandbox
`
			cmd := fmt.Sprintf("cat > /etc/fluid-daemon/daemon.yaml << 'FLUID_CONFIG_EOF'\n%sFLUID_CONFIG_EOF", configYAML)
			_, stderr, code, err := sudoRun(ctx, cmd)
			if err != nil {
				return fmt.Errorf("write config: %w", err)
			}
			if code != 0 {
				return fmt.Errorf("write config failed (exit %d): %s", code, stderr)
			}
			return nil
		},
	}
}

func stepCreateSystemdUnit() StepDef {
	return StepDef{
		Name:        "Create systemd unit",
		Description: "Install the fluid-daemon systemd service file",
		Commands: []string{
			"Write /etc/systemd/system/fluid-daemon.service",
		},
		Check: func(ctx context.Context, run hostexec.RunFunc) (bool, error) {
			_, _, code, _ := run(ctx, "test -f /etc/systemd/system/fluid-daemon.service")
			return code == 0, nil
		},
		Execute: func(ctx context.Context, sudoRun hostexec.RunFunc) error {
			unit := `[Unit]
Description=Fluid Daemon - Sandbox Host Agent
After=network.target libvirtd.service
Wants=libvirtd.service

[Service]
Type=simple
User=fluid-daemon
AmbientCapabilities=CAP_NET_ADMIN
ExecStart=/usr/local/bin/fluid-daemon serve
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
`
			cmd := fmt.Sprintf("cat > /etc/systemd/system/fluid-daemon.service << 'FLUID_UNIT_EOF'\n%sFLUID_UNIT_EOF", unit)
			_, stderr, code, err := sudoRun(ctx, cmd)
			if err != nil {
				return fmt.Errorf("write systemd unit: %w", err)
			}
			if code != 0 {
				return fmt.Errorf("write systemd unit failed (exit %d): %s", code, stderr)
			}
			return nil
		},
	}
}

func stepEnableAndStart() StepDef {
	return StepDef{
		Name:        "Enable and start daemon",
		Description: "Reload systemd, enable and start the fluid-daemon service",
		Commands: []string{
			"systemctl daemon-reload",
			"systemctl enable fluid-daemon",
			"systemctl start fluid-daemon",
		},
		Check: func(ctx context.Context, run hostexec.RunFunc) (bool, error) {
			stdout, _, _, _ := run(ctx, "systemctl is-active fluid-daemon 2>/dev/null")
			return strings.TrimSpace(stdout) == "active", nil
		},
		Execute: func(ctx context.Context, sudoRun hostexec.RunFunc) error {
			cmd := "systemctl daemon-reload && systemctl enable fluid-daemon && systemctl start fluid-daemon"
			_, stderr, code, err := sudoRun(ctx, cmd)
			if err != nil {
				return fmt.Errorf("enable/start daemon: %w", err)
			}
			if code != 0 {
				return fmt.Errorf("enable/start daemon failed (exit %d): %s", code, stderr)
			}
			return nil
		},
	}
}
