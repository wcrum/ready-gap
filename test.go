package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
)

const (
	testDumpFile = "test_sites_dump.json"
	testHtmlFile = "test_sites_report.html"
)

// RunTest generates test data and creates both JSON and HTML files
func RunTest() {
	// Create sample test data
	sites := createTestData()

	// Generate JSON file
	if err := generateTestJSON(sites); err != nil {
		log.Fatalf("Failed to generate JSON: %v", err)
	}

	// Generate HTML file
	if err := generateTestHTML(sites); err != nil {
		log.Fatalf("Failed to generate HTML: %v", err)
	}

	fmt.Println("✅ Test files generated successfully!")
	fmt.Printf("📄 JSON: %s\n", testDumpFile)
	fmt.Printf("🌐 HTML: %s\n", testHtmlFile)
}

// createTestData generates sample sites data for testing
func createTestData() map[string]*Set {
	sites := make(map[string]*Set)

	// Test host 1: Google services
	googleSet := NewSet()
	googleSet.Add("/search?q=test")
	googleSet.Add("/maps/api/geocode/json")
	googleSet.Add("/apis/oauth2/v1/tokeninfo")
	googleSet.Add("/apis/calendar/v3/calendars")
	googleSet.Add("/apis/drive/v3/files")
	googleSet.Add("/apis/gmail/v1/users/me/messages")
	sites["googleapis.com:443"] = googleSet

	// Test host 2: GitHub
	githubSet := NewSet()
	githubSet.Add("/api/v3/user")
	githubSet.Add("/api/v3/repos/owner/repo/commits")
	githubSet.Add("/api/v3/repos/owner/repo/issues")
	githubSet.Add("/api/v3/repos/owner/repo/pulls")
	githubSet.Add("/api/v3/repos/owner/repo/contents/README.md")
	githubSet.Add("/api/v3/repos/owner/repo/branches")
	githubSet.Add("/api/v3/repos/owner/repo/releases")
	githubSet.Add("/api/v3/repos/owner/repo/actions/runs")
	sites["api.github.com:443"] = githubSet

	// Test host 3: AWS services
	awsSet := NewSet()
	awsSet.Add("/s3/bucket-name/file.txt")
	awsSet.Add("/ec2/instances/i-1234567890abcdef0")
	awsSet.Add("/lambda/functions/my-function")
	awsSet.Add("/dynamodb/tables/my-table")
	awsSet.Add("/cloudformation/stacks/my-stack")
	awsSet.Add("/rds/db-instances/my-database")
	sites["aws.amazon.com:443"] = awsSet

	// Test host 4: Internal service
	internalSet := NewSet()
	internalSet.Add("/api/v1/users")
	internalSet.Add("/api/v1/users/123")
	internalSet.Add("/api/v1/users/123/profile")
	internalSet.Add("/api/v1/users/123/settings")
	internalSet.Add("/api/v1/auth/login")
	internalSet.Add("/api/v1/auth/logout")
	internalSet.Add("/api/v1/auth/refresh")
	internalSet.Add("/api/v1/data/analytics")
	internalSet.Add("/api/v1/data/reports")
	internalSet.Add("/api/v1/data/export")
	sites["internal-api.company.com:8080"] = internalSet

	// Test host 5: CDN
	cdnSet := NewSet()
	cdnSet.Add("/static/js/app.js")
	cdnSet.Add("/static/css/styles.css")
	cdnSet.Add("/static/images/logo.png")
	cdnSet.Add("/static/images/banner.jpg")
	cdnSet.Add("/static/fonts/roboto.woff2")
	cdnSet.Add("/static/videos/demo.mp4")
	sites["cdn.example.com:443"] = cdnSet

	// Test host 6: Empty host (no paths)
	emptySet := NewSet()
	sites["empty-host.com:80"] = emptySet

	return sites
}

// generateTestJSON creates a JSON file with the test data
func generateTestJSON(sites map[string]*Set) error {
	// Convert sites map to a serializable format
	sitesData := make(map[string][]string)
	for host, set := range sites {
		sitesData[host] = set.List()
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(sitesData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sites data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(testDumpFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write sites data to file: %w", err)
	}

	fmt.Printf("📄 Generated JSON file: %s\n", testDumpFile)
	return nil
}

// generateTestHTML creates an HTML file from the test data
func generateTestHTML(sites map[string]*Set) error {
	// Convert sites data to template format
	var hosts []HostInfo
	for host, set := range sites {
		hosts = append(hosts, HostInfo{
			Host:  host,
			Paths: set.List(),
		})
	}

	siteData := SiteData{Hosts: hosts}

	// Read the HTML template
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	// Create the output file
	file, err := os.Create(testHtmlFile)
	if err != nil {
		return fmt.Errorf("failed to create HTML file: %w", err)
	}
	defer file.Close()

	// Execute the template
	if err := tmpl.Execute(file, siteData); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}

	fmt.Printf("🌐 Generated HTML file: %s\n", testHtmlFile)
	return nil
}
