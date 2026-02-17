package sshconfig

import (
	"strings"
	"testing"
)

func TestParse_BasicConfig(t *testing.T) {
	config := `
Host webserver
    HostName 10.0.0.1
    User admin
    Port 2222
    IdentityFile /home/user/.ssh/id_rsa

Host dbserver
    HostName db.example.com
    User postgres
`
	hosts, err := Parse(strings.NewReader(config))
	if err != nil {
		t.Fatal(err)
	}

	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}

	web := hosts[0]
	if web.Name != "webserver" {
		t.Errorf("expected name webserver, got %s", web.Name)
	}
	if web.HostName != "10.0.0.1" {
		t.Errorf("expected hostname 10.0.0.1, got %s", web.HostName)
	}
	if web.User != "admin" {
		t.Errorf("expected user admin, got %s", web.User)
	}
	if web.Port != 2222 {
		t.Errorf("expected port 2222, got %d", web.Port)
	}
	if web.IdentityFile != "/home/user/.ssh/id_rsa" {
		t.Errorf("expected identity file, got %s", web.IdentityFile)
	}

	db := hosts[1]
	if db.Name != "dbserver" {
		t.Errorf("expected name dbserver, got %s", db.Name)
	}
	if db.Port != 22 {
		t.Errorf("expected default port 22, got %d", db.Port)
	}
}

func TestParse_SkipsWildcard(t *testing.T) {
	config := `
Host *
    ServerAliveInterval 60

Host myhost
    HostName 192.168.1.1
`
	hosts, err := Parse(strings.NewReader(config))
	if err != nil {
		t.Fatal(err)
	}

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host (wildcard skipped), got %d", len(hosts))
	}
	if hosts[0].Name != "myhost" {
		t.Errorf("expected myhost, got %s", hosts[0].Name)
	}
}

func TestParse_HostNameDefaultsToName(t *testing.T) {
	config := `
Host myserver
    User root
    Port 22
`
	hosts, err := Parse(strings.NewReader(config))
	if err != nil {
		t.Fatal(err)
	}

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].HostName != "myserver" {
		t.Errorf("expected HostName to default to Name, got %s", hosts[0].HostName)
	}
}

func TestParse_CommentsAndEmptyLines(t *testing.T) {
	config := `
# This is a comment
Host server1
    # Another comment
    HostName 10.0.0.1

    User admin
`
	hosts, err := Parse(strings.NewReader(config))
	if err != nil {
		t.Fatal(err)
	}

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].HostName != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1, got %s", hosts[0].HostName)
	}
	if hosts[0].User != "admin" {
		t.Errorf("expected admin, got %s", hosts[0].User)
	}
}

func TestParse_EqualsFormat(t *testing.T) {
	config := `
Host myhost
    HostName=10.0.0.1
    User=root
    Port=2222
`
	hosts, err := Parse(strings.NewReader(config))
	if err != nil {
		t.Fatal(err)
	}

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].HostName != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1, got %s", hosts[0].HostName)
	}
	if hosts[0].User != "root" {
		t.Errorf("expected root, got %s", hosts[0].User)
	}
	if hosts[0].Port != 2222 {
		t.Errorf("expected 2222, got %d", hosts[0].Port)
	}
}

func TestParse_EmptyInput(t *testing.T) {
	hosts, err := Parse(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(hosts))
	}
}

func TestParse_MultipleHosts(t *testing.T) {
	config := `
Host prod1
    HostName 10.0.1.1
    User deploy

Host prod2
    HostName 10.0.1.2
    User deploy

Host staging
    HostName 10.0.2.1
    User staging
    IdentityFile ~/.ssh/staging_key
`
	hosts, err := Parse(strings.NewReader(config))
	if err != nil {
		t.Fatal(err)
	}

	if len(hosts) != 3 {
		t.Fatalf("expected 3 hosts, got %d", len(hosts))
	}
}

func TestParse_SkipsQuestionMarkWildcard(t *testing.T) {
	config := `
Host web?
    HostName 10.0.0.1

Host realhost
    HostName 10.0.0.2
`
	hosts, err := Parse(strings.NewReader(config))
	if err != nil {
		t.Fatal(err)
	}

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].Name != "realhost" {
		t.Errorf("expected realhost, got %s", hosts[0].Name)
	}
}

func TestSplitDirective(t *testing.T) {
	tests := []struct {
		line string
		key  string
		val  string
	}{
		{"Host myhost", "Host", "myhost"},
		{"HostName=10.0.0.1", "HostName", "10.0.0.1"},
		{"  Port 22", "Port", "22"},
		{"User\troot", "User", "root"},
	}

	for _, tt := range tests {
		key, val := splitDirective(strings.TrimSpace(tt.line))
		if key != tt.key || val != tt.val {
			t.Errorf("splitDirective(%q) = (%q, %q), want (%q, %q)", tt.line, key, val, tt.key, tt.val)
		}
	}
}
