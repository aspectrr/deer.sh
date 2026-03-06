package sshconfig

import (
	"strings"
	"testing"
)

func TestListHostsFromReader_BasicHosts(t *testing.T) {
	input := `Host web-prod
  HostName 10.0.0.1

Host db-staging
  HostName 10.0.0.2
  User deploy
`
	hosts := ListHostsFromReader(strings.NewReader(input))
	want := []string{"web-prod", "db-staging"}
	if len(hosts) != len(want) {
		t.Fatalf("got %d hosts, want %d: %v", len(hosts), len(want), hosts)
	}
	for i, h := range want {
		if hosts[i] != h {
			t.Errorf("hosts[%d] = %q, want %q", i, hosts[i], h)
		}
	}
}

func TestListHostsFromReader_WildcardFiltering(t *testing.T) {
	input := `Host *
  ServerAliveInterval 60

Host *.example.com
  User admin

Host bastion
  HostName bastion.example.com
`
	hosts := ListHostsFromReader(strings.NewReader(input))
	if len(hosts) != 1 || hosts[0] != "bastion" {
		t.Fatalf("got %v, want [bastion]", hosts)
	}
}

func TestListHostsFromReader_MultipleAliasesPerLine(t *testing.T) {
	input := `Host alpha bravo charlie
  HostName 10.0.0.1
`
	hosts := ListHostsFromReader(strings.NewReader(input))
	want := []string{"alpha", "bravo", "charlie"}
	if len(hosts) != len(want) {
		t.Fatalf("got %d hosts, want %d: %v", len(hosts), len(want), hosts)
	}
	for i, h := range want {
		if hosts[i] != h {
			t.Errorf("hosts[%d] = %q, want %q", i, hosts[i], h)
		}
	}
}

func TestListHostsFromReader_MultipleAliasesWithWildcard(t *testing.T) {
	input := `Host jump-* real-host
  HostName 10.0.0.1
`
	hosts := ListHostsFromReader(strings.NewReader(input))
	if len(hosts) != 1 || hosts[0] != "real-host" {
		t.Fatalf("got %v, want [real-host]", hosts)
	}
}

func TestListHostsFromReader_EmptyInput(t *testing.T) {
	hosts := ListHostsFromReader(strings.NewReader(""))
	if len(hosts) != 0 {
		t.Fatalf("got %v, want empty", hosts)
	}
}

func TestListHostsFromReader_CommentsOnly(t *testing.T) {
	input := `# This is a comment
# Another comment

# Host not-a-host
`
	hosts := ListHostsFromReader(strings.NewReader(input))
	if len(hosts) != 0 {
		t.Fatalf("got %v, want empty", hosts)
	}
}

func TestListHostsFromReader_QuestionMarkWildcard(t *testing.T) {
	input := `Host web?
  HostName 10.0.0.1

Host db1
  HostName 10.0.0.2
`
	hosts := ListHostsFromReader(strings.NewReader(input))
	if len(hosts) != 1 || hosts[0] != "db1" {
		t.Fatalf("got %v, want [db1]", hosts)
	}
}
