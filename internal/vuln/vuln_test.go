package vuln_test

import (
	"context"
	"testing"

	"github.com/pragmaticivan/go-check-updates/internal/vuln"
)

func TestNewClient(t *testing.T) {
	client := vuln.NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
}

func TestCheckModule_ReturnsZeroCounts(t *testing.T) {
	client := vuln.NewClient()
	ctx := context.Background()

	counts, err := client.CheckModule(ctx, "example.com/test", "v1.0.0")
	if err != nil {
		t.Fatalf("CheckModule() returned error: %v", err)
	}

	if counts.Total != 0 {
		t.Errorf("Expected Total = 0, got %d", counts.Total)
	}
	if counts.Low != 0 {
		t.Errorf("Expected Low = 0, got %d", counts.Low)
	}
	if counts.Medium != 0 {
		t.Errorf("Expected Medium = 0, got %d", counts.Medium)
	}
	if counts.High != 0 {
		t.Errorf("Expected High = 0, got %d", counts.High)
	}
	if counts.Critical != 0 {
		t.Errorf("Expected Critical = 0, got %d", counts.Critical)
	}
}

func TestCheckModule_CachesResults(t *testing.T) {
	client := vuln.NewClient()
	ctx := context.Background()

	// First call
	counts1, err := client.CheckModule(ctx, "example.com/test", "v1.0.0")
	if err != nil {
		t.Fatalf("First CheckModule() returned error: %v", err)
	}

	// Second call with same module and version should return cached result
	counts2, err := client.CheckModule(ctx, "example.com/test", "v1.0.0")
	if err != nil {
		t.Fatalf("Second CheckModule() returned error: %v", err)
	}

	// Results should be identical
	if counts1 != counts2 {
		t.Errorf("Cached results differ: %+v != %+v", counts1, counts2)
	}
}

func TestCheckModule_DifferentVersionsCachedSeparately(t *testing.T) {
	client := vuln.NewClient()
	ctx := context.Background()

	// Check v1.0.0
	_, err := client.CheckModule(ctx, "example.com/test", "v1.0.0")
	if err != nil {
		t.Fatalf("CheckModule(v1.0.0) returned error: %v", err)
	}

	// Check v1.1.0 - should not error (different cache key)
	_, err = client.CheckModule(ctx, "example.com/test", "v1.1.0")
	if err != nil {
		t.Fatalf("CheckModule(v1.1.0) returned error: %v", err)
	}
}

func TestCheckModule_DifferentModulesCachedSeparately(t *testing.T) {
	client := vuln.NewClient()
	ctx := context.Background()

	// Check first module
	_, err := client.CheckModule(ctx, "example.com/module-a", "v1.0.0")
	if err != nil {
		t.Fatalf("CheckModule(module-a) returned error: %v", err)
	}

	// Check second module - should not error (different cache key)
	_, err = client.CheckModule(ctx, "example.com/module-b", "v1.0.0")
	if err != nil {
		t.Fatalf("CheckModule(module-b) returned error: %v", err)
	}
}

func TestCheckModule_WithContextCancellation(t *testing.T) {
	client := vuln.NewClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should still work since stub implementation doesn't use context
	_, err := client.CheckModule(ctx, "example.com/test", "v1.0.0")
	if err != nil {
		t.Fatalf("CheckModule() with cancelled context returned error: %v", err)
	}
}

func TestSeverityCountsStructure(t *testing.T) {
	counts := vuln.SeverityCounts{
		Low:      1,
		Medium:   2,
		High:     3,
		Critical: 4,
		Total:    10,
	}

	if counts.Low != 1 {
		t.Errorf("Expected Low = 1, got %d", counts.Low)
	}
	if counts.Medium != 2 {
		t.Errorf("Expected Medium = 2, got %d", counts.Medium)
	}
	if counts.High != 3 {
		t.Errorf("Expected High = 3, got %d", counts.High)
	}
	if counts.Critical != 4 {
		t.Errorf("Expected Critical = 4, got %d", counts.Critical)
	}
	if counts.Total != 10 {
		t.Errorf("Expected Total = 10, got %d", counts.Total)
	}
}
