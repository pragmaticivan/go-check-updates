package vuln

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
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
	cacheMu    sync.RWMutex
	httpClient *http.Client
}

// NewClient creates a new vulnerability client
func NewClient() Client {
	return &RealClient{
		cache: make(map[string]SeverityCounts),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
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
	c.cacheMu.RLock()
	if counts, ok := c.cache[cacheKey]; ok {
		c.cacheMu.RUnlock()
		return counts, nil
	}
	c.cacheMu.RUnlock()

	counts := SeverityCounts{}

	// Prepare OSV API query
	query := osvQuery{}
	query.Package.Name = modulePath
	query.Package.Ecosystem = "Go"
	query.Version = version

	jsonData, err := json.Marshal(query)
	if err != nil {
		return counts, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Query OSV API
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.osv.dev/v1/query", bytes.NewBuffer(jsonData))
	if err != nil {
		return counts, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return counts, fmt.Errorf("failed to query OSV API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return counts, fmt.Errorf("OSV API returned status %d", resp.StatusCode)
	}

	var osvResp osvResponse
	if err := json.NewDecoder(resp.Body).Decode(&osvResp); err != nil {
		return counts, fmt.Errorf("failed to decode OSV response: %w", err)
	}

	// Count vulnerabilities by severity
	for _, vuln := range osvResp.Vulns {
		counts.Total++

		severity := strings.ToUpper(vuln.DatabaseSpecific.Severity)
		if severity == "" && len(vuln.Severity) > 0 {
			// Try to extract severity from CVSS score
			severity = ExtractSeverityFromCVSS(vuln.Severity[0].Score)
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
	c.cacheMu.Lock()
	c.cache[cacheKey] = counts
	c.cacheMu.Unlock()

	return counts, nil
}

// ExtractSeverityFromCVSS extracts severity level from CVSS score string
// Parses CVSS vector strings like "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
// Returns severity based on impact metrics (C=Confidentiality, I=Integrity, A=Availability)
func ExtractSeverityFromCVSS(cvssScore string) string {
	if cvssScore == "" {
		return "MEDIUM"
	}

	// Parse CVSS vector into a map of metrics
	metrics := ParseCVSSVector(cvssScore)

	// Extract impact metrics
	confidentiality := metrics["C"]
	integrity := metrics["I"]
	availability := metrics["A"]
	scope := metrics["S"]

	// Count high-impact metrics (Confidentiality, Integrity, Availability)
	highImpacts := 0
	if confidentiality == "H" {
		highImpacts++
	}
	if integrity == "H" {
		highImpacts++
	}
	if availability == "H" {
		highImpacts++
	}

	// Multiple high impacts often indicate critical severity
	// Also check for scope change which can elevate severity
	if highImpacts >= 2 || (highImpacts >= 1 && scope == "C") {
		return "CRITICAL"
	}

	if highImpacts == 1 {
		return "HIGH"
	}

	// Check for medium impacts
	if confidentiality == "M" || integrity == "M" || availability == "M" {
		return "MEDIUM"
	}

	// Check for low impacts
	if confidentiality == "L" || integrity == "L" || availability == "L" {
		return "LOW"
	}

	return "MEDIUM"
}

// ParseCVSSVector parses a CVSS vector string into a map of metric:value pairs
// Example: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H" -> {"AV":"N", "AC":"L", ...}
func ParseCVSSVector(vector string) map[string]string {
	metrics := make(map[string]string)

	// Split by slash to get individual metrics
	parts := strings.Split(vector, "/")

	// Skip the first part if it starts with "CVSS:" (version indicator)
	startIdx := 0
	if len(parts) > 0 && strings.HasPrefix(parts[0], "CVSS:") {
		startIdx = 1
	}

	// Parse each metric:value pair
	for i := startIdx; i < len(parts); i++ {
		// Split by colon to separate metric from value
		pair := strings.SplitN(parts[i], ":", 2)
		if len(pair) == 2 {
			metric := strings.TrimSpace(pair[0])
			value := strings.TrimSpace(pair[1])
			metrics[metric] = value
		}
	}

	return metrics
}
