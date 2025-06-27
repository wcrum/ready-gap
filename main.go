package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/elazarl/goproxy"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	defaultAddr = ":8080"
	dumpFile    = "sites_dump.json"
	htmlFile    = "sites_report.html"
	configFile  = "configuration.yaml"
)

// Configuration represents the YAML configuration structure
type Configuration struct {
	AllowedHosts []string `yaml:"allowed_hosts"`
}

// SiteData represents the data structure for HTML template
type SiteData struct {
	Hosts []HostInfo
}

// HostInfo represents a single host and its paths
type HostInfo struct {
	Host      string
	Paths     []string
	Known     bool
	PathCount int
}

// RequestInfo represents a single request path with known status
type RequestInfo struct {
	Path  string
	Known bool
}

// SitesData represents the JSON output structure
type SitesData struct {
	Hosts map[string]HostData `json:"hosts"`
}

// HostData represents host data in JSON output
type HostData struct {
	Paths      []string `json:"paths"`
	TotalPaths int      `json:"total_paths"`
	HostKnown  bool     `json:"host_known"`
}

// Global configuration
var config Configuration

func main() {
	addr := flag.String("addr", defaultAddr, "proxy listen address")
	testMode := flag.Bool("test", false, "run in test mode to generate sample files")
	flag.Parse()

	setupLogging()

	// Load configuration (needed for both normal and test mode)
	if err := loadConfiguration(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if *testMode {
		RunTest()
		return
	}

	sites := make(map[string]*Set)
	knownPaths := make(map[string]*Set) // Track known paths per host

	proxy := setupProxy(sites, knownPaths)

	setupGracefulShutdown(sites, knownPaths)

	log.Infof("Starting proxy server on %s", *addr)
	if err := http.ListenAndServe(*addr, proxy); err != nil {
		log.Fatalf("Failed to start proxy server: %v", err)
	}
}

// loadConfiguration loads the YAML configuration file
func loadConfiguration() error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse configuration file: %w", err)
	}

	log.Infof("Loaded configuration with %d allowed hosts", len(config.AllowedHosts))
	return nil
}

// isKnownHost checks if a host is in the allowed hosts list
func isKnownHost(host string) bool {
	for _, allowedHost := range config.AllowedHosts {
		fmt.Println("Allowed host:", allowedHost)
		if allowedHost == host {
			return true
		}
	}
	return false
}

func setupLogging() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

// setupProxy creates and configures the proxy server
func setupProxy(sites map[string]*Set, knownPaths map[string]*Set) *goproxy.ProxyHttpServer {
	proxy := goproxy.NewProxyHttpServer()
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		trackRequest(sites, knownPaths, req)
		return req, nil
	})

	proxy.Logger = log.StandardLogger()
	return proxy
}

// trackRequest adds the request path to the appropriate host's set
func trackRequest(sites map[string]*Set, knownPaths map[string]*Set, req *http.Request) {
	host := req.URL.Host
	if _, exists := sites[host]; !exists {
		sites[host] = NewSet()
		knownPaths[host] = NewSet()
	}

	sites[host].Add(req.URL.Path)

	// If this host is known, add ALL paths to known paths
	if isKnownHost(host) {
		knownPaths[host].Add(req.URL.Path)
	}
	// Note: If host is unknown, we don't add to knownPaths, so it remains empty
}

// setupGracefulShutdown handles graceful shutdown and data dumping
func setupGracefulShutdown(sites map[string]*Set, knownPaths map[string]*Set) {
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Info("Received shutdown signal, dumping sites data...")
		if err := dumpSitesData(sites, knownPaths); err != nil {
			log.Errorf("Failed to dump sites data: %v", err)
			os.Exit(1)
		}

		log.Info("Shutdown complete")
		os.Exit(0)
	}()
}

// dumpSitesData serializes and writes the sites data to a JSON file
func dumpSitesData(sites map[string]*Set, knownPaths map[string]*Set) error {
	// Create enhanced JSON structure
	sitesData := SitesData{
		Hosts: make(map[string]HostData),
	}

	for host, set := range sites {
		allPaths := set.List()

		sitesData.Hosts[host] = HostData{
			Paths:      allPaths,
			TotalPaths: len(allPaths),
			HostKnown:  isKnownHost(host),
		}
	}

	jsonData, err := json.MarshalIndent(sitesData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sites data: %w", err)
	}

	if err := os.WriteFile(dumpFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write sites data to file: %w", err)
	}

	log.Infof("Sites data dumped to %s", dumpFile)
	fmt.Printf("Sites data dumped to %s\n", dumpFile)

	// Generate HTML report
	if err := generateHTMLReport(sites, knownPaths); err != nil {
		return fmt.Errorf("failed to generate HTML report: %w", err)
	}

	return nil
}

// generateHTMLReport creates an HTML file from the sites data
func generateHTMLReport(sites map[string]*Set, knownPaths map[string]*Set) error {
	// Convert sites data to template format
	var hosts []HostInfo
	for host, set := range sites {
		allPaths := set.List()

		hosts = append(hosts, HostInfo{
			Host:      host,
			Paths:     allPaths,
			Known:     isKnownHost(host),
			PathCount: len(allPaths),
		})
	}

	siteData := SiteData{Hosts: hosts}

	// Read the HTML template
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	// Create the output file
	file, err := os.Create(htmlFile)
	if err != nil {
		return fmt.Errorf("failed to create HTML file: %w", err)
	}
	defer file.Close()

	// Execute the template
	if err := tmpl.Execute(file, siteData); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}

	log.Infof("HTML report generated: %s", htmlFile)
	fmt.Printf("HTML report generated: %s\n", htmlFile)
	return nil
}
