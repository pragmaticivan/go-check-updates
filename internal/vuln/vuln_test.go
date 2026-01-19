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

	// Should return an error when context is cancelled
	_, err := client.CheckModule(ctx, "example.com/test", "v1.0.0")
	if err == nil {
		t.Fatal("CheckModule() with cancelled context should return error")
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

func TestCheckModule_ConcurrentAccess(t *testing.T) {
	client := vuln.NewClient()
	ctx := context.Background()

	// Test concurrent access to the same module
	const numGoroutines = 100
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			modulePath := "example.com/test"
			version := "v1.0.0"

			_, err := client.CheckModule(ctx, modulePath, version)
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	close(errors)
	for err := range errors {
		t.Errorf("Concurrent CheckModule() returned error: %v", err)
	}
}

func TestParseCVSSVector(t *testing.T) {
	tests := []struct {
		name     string
		vector   string
		expected map[string]string
	}{
		{
			name:   "Standard CVSS 3.1 vector",
			vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			expected: map[string]string{
				"AV": "N",
				"AC": "L",
				"PR": "N",
				"UI": "N",
				"S":  "U",
				"C":  "H",
				"I":  "H",
				"A":  "H",
			},
		},
		{
			name:   "CVSS vector with different ordering",
			vector: "CVSS:3.0/C:H/I:L/A:N/S:U/AV:N/AC:L/PR:N/UI:N",
			expected: map[string]string{
				"C":  "H",
				"I":  "L",
				"A":  "N",
				"S":  "U",
				"AV": "N",
				"AC": "L",
				"PR": "N",
				"UI": "N",
			},
		},
		{
			name:   "CVSS vector with scope changed",
			vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:L/I:L/A:L",
			expected: map[string]string{
				"AV": "N",
				"AC": "L",
				"PR": "N",
				"UI": "R",
				"S":  "C",
				"C":  "L",
				"I":  "L",
				"A":  "L",
			},
		},
		{
			name:     "Empty vector",
			vector:   "",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vuln.ParseCVSSVector(tt.vector)
			
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d metrics, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := result[key]; !ok {
					t.Errorf("Missing metric %s", key)
				} else if actualValue != expectedValue {
					t.Errorf("For metric %s: expected %s, got %s", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestExtractSeverityFromCVSS(t *testing.T) {
	tests := []struct {
		name     string
		cvss     string
		expected string
	}{
		{
			name:     "Critical: multiple high impacts",
			cvss:     "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			expected: "CRITICAL",
		},
		{
			name:     "Critical: high impact with scope change",
			cvss:     "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:L/A:L",
			expected: "CRITICAL",
		},
		{
			name:     "High: single high impact",
			cvss:     "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N",
			expected: "HIGH",
		},
		{
			name:     "High: availability high",
			cvss:     "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
			expected: "HIGH",
		},
		{
			name:     "Medium: medium impacts",
			cvss:     "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:M/I:N/A:N",
			expected: "MEDIUM",
		},
		{
			name:     "Low: low impacts",
			cvss:     "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N",
			expected: "LOW",
		},
		{
			name:     "Medium: no impacts (default)",
			cvss:     "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:N",
			expected: "MEDIUM",
		},
		{
			name:     "Medium: empty string",
			cvss:     "",
			expected: "MEDIUM",
		},
		{
			name:     "Different ordering still works",
			cvss:     "CVSS:3.0/C:H/I:H/A:L/S:U/AV:N/AC:L/PR:N/UI:N",
			expected: "CRITICAL",
		},
		{
			name:     "Medium: mixed low and medium",
			cvss:     "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:L/I:M/A:N",
			expected: "MEDIUM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vuln.ExtractSeverityFromCVSS(tt.cvss)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s for CVSS: %s", tt.expected, result, tt.cvss)
			}
		})
	}
}
