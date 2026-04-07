package network

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

func TestTAPName(t *testing.T) {
	tests := []struct {
		sandboxID string
		want      string
	}{
		{"SBX-abc123def", "fl-abc123def"},
		{"SBX-xyz", "fl-xyz"},
		{"abc123def456", "fl-abc123def456"},
		{"short", "fl-short"},
		{"sbx-e2e-1774615605670673006", "fl-605670673006"},
		{"SBX-e2e-1774615605670673006", "fl-605670673006"},
	}

	for _, tt := range tests {
		got := TAPName(tt.sandboxID)
		if got != tt.want {
			t.Errorf("TAPName(%q) = %q, want %q", tt.sandboxID, got, tt.want)
		}
	}
}

func TestCreateTAPLinuxCommands(t *testing.T) {
	prevGOOS := runtimeGOOS
	prevRun := runCmdFunc
	t.Cleanup(func() {
		runtimeGOOS = prevGOOS
		runCmdFunc = prevRun
	})

	runtimeGOOS = "linux"
	var calls [][]string
	runCmdFunc = func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	}

	tapName, err := CreateTAP(context.Background(), "fl-testtap", "br0", nil)
	if err != nil {
		t.Fatalf("CreateTAP returned error: %v", err)
	}
	if tapName != "fl-testtap" {
		t.Fatalf("CreateTAP returned tap %q, want %q", tapName, "fl-testtap")
	}

	want := [][]string{
		{"ip", "tuntap", "add", "dev", "fl-testtap", "mode", "tap"},
		{"ip", "link", "set", "fl-testtap", "master", "br0"},
		{"ip", "link", "set", "fl-testtap", "promisc", "on"},
		{"ip", "link", "set", "fl-testtap", "up"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("CreateTAP command sequence mismatch:\n got: %#v\nwant: %#v", calls, want)
	}
}

func TestCreateTAPDarwinCommands(t *testing.T) {
	prevGOOS := runtimeGOOS
	prevRun := runCmdFunc
	prevOutput := runOutputCmdFunc
	t.Cleanup(func() {
		runtimeGOOS = prevGOOS
		runCmdFunc = prevRun
		runOutputCmdFunc = prevOutput
	})

	runtimeGOOS = "darwin"
	runOutputCmdFunc = func(_ context.Context, name string, args ...string) (string, error) {
		if name != "ifconfig" || !reflect.DeepEqual(args, []string{"tap", "create"}) {
			t.Fatalf("unexpected output command: %s %v", name, args)
		}
		return "tap42\n", nil
	}

	var calls [][]string
	runCmdFunc = func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	}

	tapName, err := CreateTAP(context.Background(), "ignored", "bridge0", nil)
	if err != nil {
		t.Fatalf("CreateTAP returned error: %v", err)
	}
	if tapName != "tap42" {
		t.Fatalf("CreateTAP returned tap %q, want tap42", tapName)
	}

	want := [][]string{
		{"ifconfig", "bridge0", "addm", "tap42"},
		{"ifconfig", "tap42", "up"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("CreateTAP command sequence mismatch:\n got: %#v\nwant: %#v", calls, want)
	}
}

func TestDestroyTAPDarwin(t *testing.T) {
	prevGOOS := runtimeGOOS
	prevRun := runCmdFunc
	t.Cleanup(func() {
		runtimeGOOS = prevGOOS
		runCmdFunc = prevRun
	})

	runtimeGOOS = "darwin"
	runCmdFunc = func(_ context.Context, name string, args ...string) error {
		if name != "ifconfig" || !reflect.DeepEqual(args, []string{"tap7", "destroy"}) {
			return fmt.Errorf("unexpected command: %s %v", name, args)
		}
		return nil
	}

	if err := DestroyTAP(context.Background(), "tap7"); err != nil {
		t.Fatalf("DestroyTAP returned error: %v", err)
	}
}

func TestParseTapCreateOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "plain", output: "tap0\n", want: "tap0"},
		{name: "with extra text", output: "Created tap7\n", want: "tap7"},
		{name: "empty", output: "\n", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseTapCreateOutput(tt.output); got != tt.want {
				t.Fatalf("parseTapCreateOutput(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}
