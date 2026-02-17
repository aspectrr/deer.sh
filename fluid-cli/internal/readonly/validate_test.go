package readonly

import (
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
		{"curl http://example.com", "curl can modify things"},
		{"python3 -c 'import os; os.system(\"rm -rf /\")'", "python is arbitrary code"},
		{"bash -c 'rm -rf /'", "bash allows arbitrary code"},
		{"sh -c 'rm -rf /'", "sh allows arbitrary code"},
		{"vi /etc/hosts", "vi is an editor"},
		{"nano /etc/hosts", "nano is an editor"},
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
