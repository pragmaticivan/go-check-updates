package vuln

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// SeverityCounts holds vulnerability counts by severity level
type SeverityCounts struct {
	Low      int
	Medium   int
	High     int
	Critical int
	Total    int
}

// Client provides vulnerability checking capabilities
type Client interface {
	CheckModule(ctx context.Context, modulePath, version string) (SeverityCounts, error)
}

// RealClient implements Client using OSV API
type RealClient struct {
	cache      map[string]SeverityCounts
	httpClient *http.Client
}

// NewClient creates a new vulnerability client
func NewClient() Client {
	return &RealClient{
		cache:      make(map[string]SeverityCounts),
		httpClient: &http.Client{},
	}
}

// osvQuery represents the request to OSV API
type osvQuery struct {
	Package struct {
		Name      string `json:"name"`
		Ecosystem string `json:"ecosystem"`
	} `json:"package"`
	Version string `json:"version"`
}

// osvResponse represents the response from OSV API
type osvResponse struct {
	Vulns []struct {
		ID               string `json:"id"`
		Summary          string `json:"summary"`
		DatabaseSpecific struct {
			Severity string `json:"severity"`
		} `json:"database_specific"`
		Severity []struct {
			Type  string `json:"type"`
			Score string `json:"score"`
		} `json:"severity"`
	} `json:"vulns"`
}

// CheckModule fetches vulnerability data for a specific module version using OSV API
func (c *RealClient) CheckModule(ctx context.Context, modulePath, version string) (SeverityCounts, error) {
	cacheKey := fmt.Sprintf("%s@%s", modulePath, version)

	// Check cache first
	if counts, ok := c.cache[cacheKey]; ok {
		return counts, nil
	}

	counts := SeverityCounts{}

	// Prepare OSV API query
	query := osvQuery{}
	query.Package.Name = modulePath
	query.Package.Ecosystem = "Go"
	query.Version = version

	jsonData, err := json.Marshal(query)
	if err != nil {
		c.cache[cacheKey] = counts
		return counts, nil
	}

	// Query OSV API
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.osv.dev/v1/query", bytes.NewBuffer(jsonData))
	if err != nil {
		c.cache[cacheKey] = counts
		return counts, nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.cache[cacheKey] = counts
		return counts, nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		c.cache[cacheKey] = counts
		return counts, nil
	}

	var osvResp osvResponse
	if err := json.NewDecoder(resp.Body).Decode(&osvResp); err != nil {
		c.cache[cacheKey] = counts
		return counts, nil
	}

	// Count vulnerabilities by severity
	for _, vuln := range osvResp.Vulns {
		counts.Total++

		severity := strings.ToUpper(vuln.DatabaseSpecific.Severity)
		if severity == "" && len(vuln.Severity) > 0 {
			// Try to extract severity from CVSS score
			severity = extractSeverityFromCVSS(vuln.Severity[0].Score)
		}

		switch severity {
		case "LOW":
			counts.Low++
		case "MODERATE", "MEDIUM":
			counts.Medium++
		case "HIGH":
			counts.High++
		case "CRITICAL":
			counts.Critical++
		default:
			counts.Medium++ // Default to medium if unknown
		}
	}

	// Cache the result
	c.cache[cacheKey] = counts
	return counts, nil
}

// extractSeverityFromCVSS extracts severity level from CVSS score string
func extractSeverityFromCVSS(cvssScore string) string {
	// CVSS format: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H"
	// Base score ranges: 0.0-3.9=LOW, 4.0-6.9=MEDIUM, 7.0-8.9=HIGH, 9.0-10.0=CRITICAL

	// Count high-impact metrics (Confidentiality, Integrity, Availability)
	highImpacts := 0
	if strings.Contains(cvssScore, "/C:H") {
		highImpacts++
	}
	if strings.Contains(cvssScore, "/I:H") {
		highImpacts++
	}
	if strings.Contains(cvssScore, "/A:H") {
		highImpacts++
	}

	// Multiple high impacts often indicate critical severity
	// Also check for scope change which can elevate severity
	if highImpacts >= 2 || (highImpacts >= 1 && strings.Contains(cvssScore, "/S:C")) {
		return "CRITICAL"
	}

	if highImpacts == 1 {
		return "HIGH"
	}

	// Check for medium impacts
	if strings.Contains(cvssScore, "/C:M") || strings.Contains(cvssScore, "/I:M") || strings.Contains(cvssScore, "/A:M") {
		return "MEDIUM"
	}

	// Check for low impacts
	if strings.Contains(cvssScore, "/C:L") || strings.Contains(cvssScore, "/I:L") || strings.Contains(cvssScore, "/A:L") {
		return "LOW"
	}

	return "MEDIUM"
}
