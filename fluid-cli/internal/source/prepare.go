package source

import (
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/config"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/docsprogress"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/sshconfig"
)

// SavePreparedHost updates the config with resolved host details after a
// successful prepare, saves to disk, and fires a docs-progress report.
func SavePreparedHost(cfg *config.Config, configPath, hostname string, resolved *sshconfig.ResolvedHost) error {
	found := false
	for i, h := range cfg.Hosts {
		if h.Name == hostname {
			cfg.Hosts[i].Address = resolved.Hostname
			cfg.Hosts[i].SSHUser = resolved.User
			cfg.Hosts[i].SSHPort = resolved.Port
			cfg.Hosts[i].Prepared = true
			found = true
			break
		}
	}
	if !found {
		cfg.Hosts = append(cfg.Hosts, config.HostConfig{
			Name:     hostname,
			Address:  resolved.Hostname,
			SSHUser:  resolved.User,
			SSHPort:  resolved.Port,
			Prepared: true,
		})
	}

	var saveErr error
	if configPath != "" {
		saveErr = cfg.Save(configPath)
	}

	if cfg.DocsSessionCode != "" && cfg.APIURL != "" {
		go docsprogress.ReportCompletion(cfg.APIURL, cfg.DocsSessionCode, 1)
	}

	return saveErr
}
