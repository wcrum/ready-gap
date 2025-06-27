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
)

const (
	defaultAddr = ":8080"
	dumpFile    = "sites_dump.json"
	htmlFile    = "sites_report.html"
)

type SiteData struct {
	Hosts []HostInfo
}

type HostInfo struct {
	Host  string
	Paths []string
}

func main() {
	addr := flag.String("addr", defaultAddr, "proxy listen address")
	testMode := flag.Bool("test", false, "run in test mode to generate sample files")
	flag.Parse()

	if *testMode {
		RunTest()
		return
	}

	setupLogging()

	sites := make(map[string]*Set)

	proxy := setupProxy(sites)

	setupGracefulShutdown(sites)

	log.Infof("Starting proxy server on %s", *addr)
	if err := http.ListenAndServe(*addr, proxy); err != nil {
		log.Fatalf("Failed to start proxy server: %v", err)
	}
}

func setupLogging() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func setupProxy(sites map[string]*Set) *goproxy.ProxyHttpServer {
	proxy := goproxy.NewProxyHttpServer()
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		trackRequest(sites, req)
		return req, nil
	})

	proxy.Logger = log.StandardLogger()
	return proxy
}

func trackRequest(sites map[string]*Set, req *http.Request) {
	host := req.URL.Host
	if _, exists := sites[host]; !exists {
		sites[host] = NewSet()
	}
	sites[host].Add(req.URL.Path)
}

func setupGracefulShutdown(sites map[string]*Set) {
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Info("Received shutdown signal, dumping sites data...")
		if err := dumpSitesData(sites); err != nil {
			log.Errorf("Failed to dump sites data: %v", err)
			os.Exit(1)
		}

		log.Info("Shutdown complete")
		os.Exit(0)
	}()
}

func dumpSitesData(sites map[string]*Set) error {
	sitesData := make(map[string][]string)
	for host, set := range sites {
		sitesData[host] = set.List()
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

	if err := generateHTMLReport(sites); err != nil {
		return fmt.Errorf("failed to generate HTML report: %w", err)
	}

	return nil
}

func generateHTMLReport(sites map[string]*Set) error {
	var hosts []HostInfo
	for host, set := range sites {
		hosts = append(hosts, HostInfo{
			Host:  host,
			Paths: set.List(),
		})
	}

	siteData := SiteData{Hosts: hosts}

	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	file, err := os.Create(htmlFile)
	if err != nil {
		return fmt.Errorf("failed to create HTML file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, siteData); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}

	log.Infof("HTML report generated: %s", htmlFile)
	fmt.Printf("HTML report generated: %s\n", htmlFile)
	return nil
}
