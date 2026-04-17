package readonly

import (
	"sort"
	"testing"
)

func TestValidateCommand_Allowed(t *testing.T) {
	allowed := []string{
		"ls -la /etc",
		"cat /etc/hostname",
		"ps aux",
		"journalctl -u nginx --no-pager",
		"systemctl status nginx",
		"systemctl show sshd",
		"systemctl list-units",
		"systemctl is-active nginx",
		"systemctl is-enabled sshd",
		"df -h",
		"free -m",
		"uname -a",
		"whoami",
		"uptime",
		"ss -tlnp",
		"ip addr",
		"dpkg -l",
		"dpkg --list",
		"rpm -qa",
		"rpm -q nginx",
		"apt list --installed",
		"pip list",
		"dmesg | tail -20",
		"ps aux | grep nginx",
		"cat /etc/hosts | sort | uniq",
		"find /etc -name '*.conf' | head -10",
		"ls -la /var/log | grep syslog",
		"echo hello",
		"base64 /etc/hostname",
		"du -sh /var/log",
		"stat /etc/passwd",
		"head -5 /etc/passwd",
		"tail -f /var/log/syslog",
		"md5sum /etc/hostname",
		"sha256sum /etc/hostname",
		"env",
		"printenv PATH",
		"date",
		"which nginx",
		"hostname",
		"lscpu",
		"nproc",
		"arch",
		"id",
		"groups",
		"who",
		"w",
		"last -5",
		"netstat -tlnp",
		"dig example.com",
		"nslookup example.com",
		"lsblk",
		"tree /etc/nginx",
		"file /usr/bin/ls",
		"wc -l /etc/passwd",
		"readlink /proc/self/exe",
		"realpath /etc/../etc/hosts",
		"basename /etc/hosts",
		"dirname /etc/hosts",
		"pgrep nginx",
		"lsmod",
		"lspci",
		"lsusb",
		"test -f /etc/hosts",
		"strings /usr/bin/ls | head -10",
		"ps aux | grep nginx | awk '{print $2}'",
		// Chained commands
		"ls /etc && cat /etc/hostname",
		"uname -a ; hostname",
		// Env var prefix
		"FOO=bar env",
		// xargs with allowed commands
		"find /etc | xargs grep pattern",
		"find /etc | xargs cat",
		"find /etc | xargs -0 grep pattern",
		// xargs alone (defaults to /bin/echo, safe)
		"echo foo | xargs",
		// sed without -i is fine
		"sed -n 's/foo/bar/p' file",
		// openssl read-only subcommands
		"openssl x509 -in /etc/ssl/cert.pem -text -noout",
		"openssl s_client -connect localhost:443",
		"openssl s_client -connect 127.0.0.1:443",
		"openssl s_client -connect [::1]:443",
		"openssl s_client -connect [::1]",
		"openssl verify -CAfile /etc/ssl/ca.pem /etc/ssl/cert.pem",
		"openssl version",
		"openssl ciphers",
		"openssl crl -in /etc/ssl/crl.pem -text -noout",
		"openssl req -text -noout -in /tmp/csr.pem",
		// curl read-only operations
		"curl localhost:9200/_cluster/health",
		"curl -s localhost:9200/_cluster/health?pretty",
		"curl -s localhost:9200/_cat/nodes?v",
		"curl -s localhost:9200/_cat/shards?v",
		"curl -s localhost:9200/_cluster/allocation/explain?pretty",
		"curl -s localhost:9200/_cat/indices?v",
		"curl -s localhost:9200/_nodes/stats?pretty",
		"curl -s localhost:9200/_cluster/settings?include_defaults=true&flat_settings=true",
		"curl -s localhost:9200/_cat/allocation?v",
		"curl -v localhost:9200/",
		"curl -k https://localhost:9200/_cluster/health",
		"curl --cacert /etc/ssl/ca.pem https://localhost:9200/",
		"curl -s localhost:9200/_cluster/health | grep status",
		"curl -s localhost:9200/_cat/indices?v | grep red",
		"curl -s http://localhost:9200/_cluster/health?pretty",
		"curl -s -u elastic:changeme localhost:9200/_cluster/health",
		"curl -H 'Content-Type: application/json' localhost:9200/_search",
	}

	for _, cmd := range allowed {
		if err := ValidateCommand(cmd); err != nil {
			t.Errorf("expected command %q to be allowed, got error: %v", cmd, err)
		}
	}
}

func TestValidateCommand_Blocked(t *testing.T) {
	blocked := []struct {
		cmd    string
		reason string
	}{
		{"rm -rf /", "rm is destructive"},
		{"sudo ls", "sudo escalates privileges"},
		{"mv /etc/hosts /tmp/", "mv is destructive"},
		{"cp /etc/hosts /tmp/", "cp can modify files"},
		{"dd if=/dev/zero of=/dev/sda", "dd is destructive"},
		{"kill -9 1", "kill is destructive"},
		{"shutdown -h now", "shutdown is destructive"},
		{"reboot", "reboot is destructive"},
		{"systemctl start nginx", "start is not allowed"},
		{"systemctl stop nginx", "stop is not allowed"},
		{"systemctl restart nginx", "restart is not allowed"},
		{"systemctl enable nginx", "enable is not allowed"},
		{"systemctl disable nginx", "disable is not allowed"},
		{"dpkg -i package.deb", "install is not allowed"},
		{"rpm -i package.rpm", "install is not allowed"},
		{"apt install nginx", "install is not allowed"},
		{"apt remove nginx", "remove is not allowed"},
		{"pip install requests", "install is not allowed"},
		{"chmod 777 /etc/hosts", "chmod modifies permissions"},
		{"chown root:root /etc/hosts", "chown modifies ownership"},
		{"useradd testuser", "useradd is destructive"},
		{"userdel testuser", "userdel is destructive"},
		{"passwd root", "passwd modifies credentials"},
		{"mkfs.ext4 /dev/sda1", "mkfs is destructive"},
		{"mount /dev/sda1 /mnt", "mount modifies system"},
		{"wget http://example.com", "wget downloads files"},
		// curl write operations
		{"curl -X POST http://localhost:9200/_cluster/settings", "curl POST is not read-only"},
		{"curl -X PUT http://localhost:9200/myindex", "curl PUT is not read-only"},
		{"curl -X DELETE http://localhost:9200/myindex", "curl DELETE is not read-only"},
		{"curl -d '{\"settings\": {}}' http://localhost:9200", "curl with data is not read-only"},
		{"curl --data 'payload' http://localhost:9200", "curl with data is not read-only"},
		{"curl --data-raw 'payload' http://localhost:9200", "curl with data-raw is not read-only"},
		{"curl --data-binary @file http://localhost:9200", "curl with data-binary is not read-only"},
		{"curl --data-urlencode 'key=val' http://localhost:9200", "curl with data-urlencode is not read-only"},
		{"curl -F 'file=@/etc/passwd' http://localhost:9200", "curl with form is not read-only"},
		{"curl -T /etc/passwd http://localhost:9200", "curl upload is not read-only"},
		{"curl --upload-file /etc/passwd http://localhost:9200", "curl upload is not read-only"},
		{"curl -o /tmp/out http://localhost:9200", "curl output to file is not read-only"},
		{"curl --output /tmp/out http://localhost:9200", "curl output to file is not read-only"},
		{"curl -O http://localhost:9200/data", "curl remote-name is not read-only"},
		{"curl --remote-name http://localhost:9200/data", "curl remote-name is not read-only"},
		{"curl --proxy evil.com:80 http://localhost:9200", "curl proxy is not read-only"},
		{"curl -x evil.com:80 http://localhost:9200", "curl proxy is not read-only"},
		{"curl --config /tmp/evil http://localhost:9200", "curl config file is not read-only"},
		{"curl -K /tmp/evil http://localhost:9200", "curl config file is not read-only"},

		{"python3 -c 'import os; os.system(\"rm -rf /\")'", "python is arbitrary code"},
		{"bash -c 'rm -rf /'", "bash allows arbitrary code"},
		{"sh -c 'rm -rf /'", "sh allows arbitrary code"},
		{"vi /etc/hosts", "vi is an editor"},
		{"nano /etc/hosts", "nano is an editor"},
		// xargs with disallowed commands
		{"find /etc | xargs rm -rf /", "xargs can invoke arbitrary commands"},
		{"find /etc | xargs /usr/bin/rm", "xargs with path-qualified disallowed command"},
		// sed -i (in-place editing)
		{"sed -i 's/foo/bar/' file", "sed -i modifies files"},
		{"sed --in-place 's/foo/bar/' file", "sed --in-place modifies files"},
		// openssl dangerous subcommands
		{"openssl genrsa 2048", "genrsa generates keys"},
		{"openssl genpkey -algorithm RSA", "genpkey generates keys"},
		{"openssl req -new -key server.key", "req creates CSRs"},
		{"openssl enc -aes-256-cbc -in file", "enc encrypts/decrypts"},
		{"openssl ca -in req.pem", "ca is a CA operation"},
		{"openssl pkcs12 -export -in cert.pem", "pkcs12 can export"},
		{"openssl rand 32", "rand generates random data"},
		{"openssl s_client -connect remote.host:443", "s_client to remote host"},
		{"openssl s_client -connect localhost:443 -proxy evil.com:80", "s_client proxy bypasses localhost restriction"},
		{"openssl s_client -proxy evil.com:80 -connect localhost:443", "s_client proxy before connect"},
		{"openssl s_server -port 4433", "s_server starts a server"},
	}

	for _, tc := range blocked {
		err := ValidateCommand(tc.cmd)
		if err == nil {
			t.Errorf("expected command %q to be blocked (%s), but it was allowed", tc.cmd, tc.reason)
		}
	}
}

func TestValidateCommand_Redirection(t *testing.T) {
	tests := []string{
		"echo hello > /tmp/out",
		"cat /etc/hosts >> /tmp/out",
		"ls > /dev/null",
	}

	for _, cmd := range tests {
		err := ValidateCommand(cmd)
		if err == nil {
			t.Errorf("expected command %q to be blocked (redirection), but it was allowed", cmd)
		}
	}
}

func TestValidateCommand_Pipes(t *testing.T) {
	// All segments must be allowed
	tests := []struct {
		cmd     string
		allowed bool
	}{
		{"ps aux | grep nginx", true},
		{"ps aux | rm -rf /", false},
		{"rm -rf / | grep error", false},
		{"cat /etc/hosts | sort | uniq | wc -l", true},
	}

	for _, tc := range tests {
		err := ValidateCommand(tc.cmd)
		if tc.allowed && err != nil {
			t.Errorf("expected pipe command %q to be allowed, got error: %v", tc.cmd, err)
		}
		if !tc.allowed && err == nil {
			t.Errorf("expected pipe command %q to be blocked, but it was allowed", tc.cmd)
		}
	}
}

func TestValidateCommand_Empty(t *testing.T) {
	if err := ValidateCommand(""); err == nil {
		t.Error("expected empty command to return error")
	}
	if err := ValidateCommand("   "); err == nil {
		t.Error("expected whitespace-only command to return error")
	}
}

func TestValidateCommand_SubcommandRestrictions(t *testing.T) {
	tests := []struct {
		cmd     string
		allowed bool
	}{
		{"systemctl status nginx", true},
		{"systemctl show sshd", true},
		{"systemctl list-units", true},
		{"systemctl is-active nginx", true},
		{"systemctl is-enabled nginx", true},
		{"systemctl start nginx", false},
		{"systemctl stop nginx", false},
		{"systemctl restart nginx", false},
		{"systemctl reload nginx", false},
		{"systemctl daemon-reload", false},
		{"dpkg -l", true},
		{"dpkg --list", true},
		{"dpkg -i foo.deb", false},
		{"dpkg --purge foo", false},
		{"rpm -qa", true},
		{"rpm -q nginx", true},
		{"rpm -i foo.rpm", false},
		{"apt list", true},
		{"apt install nginx", false},
		{"apt remove nginx", false},
		{"pip list", true},
		{"pip install requests", false},
	}

	for _, tc := range tests {
		err := ValidateCommand(tc.cmd)
		if tc.allowed && err != nil {
			t.Errorf("expected %q to be allowed, got: %v", tc.cmd, err)
		}
		if !tc.allowed && err == nil {
			t.Errorf("expected %q to be blocked", tc.cmd)
		}
	}
}

func TestValidateCommand_PathQualified(t *testing.T) {
	// Path-qualified allowed commands should work
	if err := ValidateCommand("/usr/bin/cat /etc/hosts"); err != nil {
		t.Errorf("expected /usr/bin/cat to be allowed: %v", err)
	}
	// Path-qualified blocked commands should still be blocked
	if err := ValidateCommand("/usr/bin/rm -rf /"); err == nil {
		t.Error("expected /usr/bin/rm to be blocked")
	}
}

func TestValidateCommand_CommandSubstitution(t *testing.T) {
	tests := []string{
		"echo $(rm -rf /)",
		"cat /etc/hosts && echo $(whoami)",
		"ls $(pwd)",
		"echo `rm -rf /`",
		"cat /etc/hosts && echo `whoami`",
		"ls `pwd`",
		"ps aux | grep `whoami`",
	}

	for _, cmd := range tests {
		err := ValidateCommand(cmd)
		if err == nil {
			t.Errorf("expected command %q to be blocked (command substitution), but it was allowed", cmd)
		}
	}
}

func TestValidateCommand_ProcessSubstitution(t *testing.T) {
	tests := []string{
		"diff <(ls /etc) <(ls /var)",
		"cat <(echo hello)",
		"tee >(cat)",
		"echo hello > >(cat)",
	}

	for _, cmd := range tests {
		err := ValidateCommand(cmd)
		if err == nil {
			t.Errorf("expected command %q to be blocked (process substitution), but it was allowed", cmd)
		}
	}
}

func TestValidateCommand_Newlines(t *testing.T) {
	tests := []string{
		"ls\nrm -rf /",
		"cat /etc/hosts\nwhoami",
		"echo hello\r\nrm -rf /",
		"ps aux\nkill -9 1",
	}

	for _, cmd := range tests {
		err := ValidateCommand(cmd)
		if err == nil {
			t.Errorf("expected command %q to be blocked (newlines), but it was allowed", cmd)
		}
	}
}

func TestValidateCommand_QuotedMetacharacters(t *testing.T) {
	// Metacharacters inside quotes should be allowed
	allowed := []string{
		"echo '$(rm -rf /)'",
		"echo \"`whoami`\"",
		"echo 'hello\nworld'",
		"cat /etc/hosts | grep 'test > output'",
	}

	for _, cmd := range allowed {
		if err := ValidateCommand(cmd); err != nil {
			t.Errorf("expected command %q to be allowed (metacharacters in quotes), got error: %v", cmd, err)
		}
	}
}

func TestValidateCommandWithExtra(t *testing.T) {
	// "docker" is not in the default allowlist
	if err := ValidateCommand("docker ps"); err == nil {
		t.Fatal("expected docker to be blocked by default")
	}

	// With extra allowed commands, docker should pass
	if err := ValidateCommandWithExtra("docker ps", []string{"docker"}); err != nil {
		t.Errorf("expected docker to be allowed with extra, got: %v", err)
	}

	// Without extra, still blocked
	if err := ValidateCommandWithExtra("docker ps", nil); err == nil {
		t.Error("expected docker to be blocked without extra")
	}

	// Default commands still work with extra
	if err := ValidateCommandWithExtra("ls -la", []string{"docker"}); err != nil {
		t.Errorf("expected ls to still be allowed: %v", err)
	}

	// Empty and whitespace entries in extra are ignored
	if err := ValidateCommandWithExtra("docker ps", []string{"", " ", "docker"}); err != nil {
		t.Errorf("expected docker to be allowed: %v", err)
	}
}

func TestAllowedCommandsList(t *testing.T) {
	cmds := AllowedCommandsList()
	if len(cmds) == 0 {
		t.Fatal("expected non-empty command list")
	}

	// Verify sorted
	if !sort.StringsAreSorted(cmds) {
		t.Error("expected command list to be sorted")
	}

	// Spot-check some known defaults
	found := make(map[string]bool)
	for _, c := range cmds {
		found[c] = true
	}
	for _, expected := range []string{"cat", "ls", "grep", "ps"} {
		if !found[expected] {
			t.Errorf("expected %q in default commands", expected)
		}
	}
}

func TestSubcommandRestrictions(t *testing.T) {
	restrs := SubcommandRestrictions()
	if len(restrs) == 0 {
		t.Error("expected non-empty result")
	}

	// Check systemctl has restrictions
	subs, ok := restrs["systemctl"]
	if !ok {
		t.Error("expected systemctl to have subcommand restrictions")
	}
	if len(subs) == 0 {
		t.Error("expected systemctl to have at least one subcommand")
	}

	// Verify values are sorted
	for cmd, subs := range restrs {
		if !sort.StringsAreSorted(subs) {
			t.Errorf("expected %q subcommands to be sorted, got %v", cmd, subs)
		}
	}

	// Spot check systemctl
	found := false
	for _, s := range subs {
		if s == "status" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'status' in systemctl subcommands")
	}
}
