package provider

import "testing"

func TestNormalizeCreateRequestResources_ClampsKafkaBackedRequests(t *testing.T) {
	req, clamped := NormalizeCreateRequestResources(CreateRequest{
		VCPUs:    1,
		MemoryMB: 512,
		KafkaBroker: &KafkaBrokerConfig{
			Port: 9092,
		},
	}, DefaultSandboxVCPUs, DefaultSandboxMemMB)

	if !clamped {
		t.Fatal("expected kafka-backed request to be clamped")
	}
	if req.VCPUs != KafkaBrokerMinVCPUs {
		t.Fatalf("VCPUs = %d, want %d", req.VCPUs, KafkaBrokerMinVCPUs)
	}
	if req.MemoryMB != KafkaBrokerMinMemoryMB {
		t.Fatalf("MemoryMB = %d, want %d", req.MemoryMB, KafkaBrokerMinMemoryMB)
	}
}

func TestNormalizeCreateRequestResources_LeavesNonKafkaRequestsAlone(t *testing.T) {
	req, clamped := NormalizeCreateRequestResources(CreateRequest{
		VCPUs:    1,
		MemoryMB: 512,
	}, DefaultSandboxVCPUs, DefaultSandboxMemMB)

	if clamped {
		t.Fatal("did not expect non-kafka request to be clamped")
	}
	if req.VCPUs != 1 {
		t.Fatalf("VCPUs = %d, want 1", req.VCPUs)
	}
	if req.MemoryMB != 512 {
		t.Fatalf("MemoryMB = %d, want 512", req.MemoryMB)
	}
}
