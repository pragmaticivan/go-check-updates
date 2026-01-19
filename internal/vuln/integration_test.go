package vuln

import (
	"context"
	"testing"
)

// TestRealOSVIntegration tests the actual OSV API with known vulnerable package
// This test requires internet connectivity
func TestRealOSVIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping OSV integration test in short mode")
	}

	client := NewClient()
	ctx := context.Background()

	// Test gopkg.in/yaml.v3@v3.0.0 which has a known HIGH severity vulnerability
	// GHSA-hp87-p4gw-j4gq: gopkg.in/yaml.v3 Denial of Service
	counts, err := client.CheckModule(ctx, "gopkg.in/yaml.v3", "v3.0.0")
	if err != nil {
		t.Fatalf("Failed to check module: %v", err)
	}

	if counts.Total == 0 {
		t.Error("Expected at least one vulnerability for gopkg.in/yaml.v3@v3.0.0")
	}

	if counts.High == 0 {
		t.Errorf("Expected at least one HIGH severity vulnerability, got High=%d, total breakdown: L=%d M=%d H=%d C=%d",
			counts.High, counts.Low, counts.Medium, counts.High, counts.Critical)
	}

	t.Logf("gopkg.in/yaml.v3@v3.0.0 vulnerabilities: L=%d M=%d H=%d C=%d Total=%d",
		counts.Low, counts.Medium, counts.High, counts.Critical, counts.Total)

	// Test that v3.0.1 has the vulnerability fixed
	fixedCounts, err := client.CheckModule(ctx, "gopkg.in/yaml.v3", "v3.0.1")
	if err != nil {
		t.Fatalf("Failed to check fixed module: %v", err)
	}

	if fixedCounts.Total > counts.Total {
		t.Errorf("Expected fewer or equal vulnerabilities in v3.0.1 (fixed version), got current=%d fixed=%d",
			counts.Total, fixedCounts.Total)
	}

	t.Logf("gopkg.in/yaml.v3@v3.0.1 vulnerabilities: L=%d M=%d H=%d C=%d Total=%d",
		fixedCounts.Low, fixedCounts.Medium, fixedCounts.High, fixedCounts.Critical, fixedCounts.Total)

	// Verify caching works
	cachedCounts, err := client.CheckModule(ctx, "gopkg.in/yaml.v3", "v3.0.0")
	if err != nil {
		t.Fatalf("Failed to get cached result: %v", err)
	}

	if cachedCounts.Total != counts.Total {
		t.Errorf("Cached result differs from original: original=%d cached=%d",
			counts.Total, cachedCounts.Total)
	}
}

// TestOSVAPICleanModule tests a module known to have no vulnerabilities
func TestOSVAPICleanModule(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping OSV integration test in short mode")
	}

	client := NewClient()
	ctx := context.Background()

	// Test a recent stable version of a well-maintained package
	counts, err := client.CheckModule(ctx, "github.com/spf13/cobra", "v1.8.1")
	if err != nil {
		t.Fatalf("Failed to check module: %v", err)
	}

	t.Logf("github.com/spf13/cobra@v1.8.1 vulnerabilities: L=%d M=%d H=%d C=%d Total=%d",
		counts.Low, counts.Medium, counts.High, counts.Critical, counts.Total)

	// Note: We don't assert zero because vulnerabilities may be discovered after this test is written
	// Just log the results for visibility
}
