package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/lucinexus/lucinexus/internal/config"
	"github.com/lucinexus/lucinexus/internal/engine"
	"github.com/lucinexus/lucinexus/internal/server"
)

func main() {
	cfgPath := flag.String("config", "", "Path to custom JSON config")
	serverMode := flag.Bool("server", false, "Run in background daemon mode")
	dryRun := flag.Bool("dry-run", true, "Preview plan without mutating state")
	approve := flag.Bool("approve", false, "Apply changes without preview")
	curlCmd := flag.String("curl", "", "Import and configure custom AI provider from curl command")
	version := flag.Bool("version", false, "Print version")
	flag.Parse()

	if *version {
		fmt.Printf("LuciNexus v%s (%s)
Autonomous, Resilient OpenWrt Router Agent
", config.Version, config.BuildTime)
		os.Exit(0)
	}

	if *curlCmd != "" {
		handleCURLImport(*curlCmd)
		return
	}

	cfg, err := config.LoadConfig(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v
", err)
		os.Exit(1)
	}

	orch, err := engine.NewOrchestrator(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Init error: %v
", err)
		os.Exit(1)
	}

	if *serverMode {
		srv := server.NewServer(cfg, orch)
		fmt.Printf("Starting LuciNexus daemon on 127.0.0.1:%d...
", cfg.ServerPort)
		if err := srv.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Server exited: %v
", err)
		}
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: lucinexus [flags] "<command prompt>"")
		fmt.Println("Example: lucinexus "restart wifi"")
		fmt.Println("Example: lucinexus -approve "open port 8080 for web server"")
		fmt.Println("Example: lucinexus --curl "curl https://api.deepseek.com/..."")
		os.Exit(0)
	}

	isDryRun := *dryRun
	if *approve {
		isDryRun = false
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()

	plan, err := orch.Execute(ctx, args[0], isDryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Execution Failed: %v
", err)
		os.Exit(1)
	}

	fmt.Println("==================================================")
	fmt.Printf("LuciNexus Plan Summary: %s
", plan.Summary)
	if plan.FastpathUsed {
		fmt.Println("[Mode: Local Fastpath - Zero API latency, Offline safe]")
	}
	if isDryRun {
		fmt.Println("[Status: Dry-Run Preview - Run with -approve to execute]")
	} else {
		fmt.Println("[Status: Executed Successfully]")
	}
	fmt.Println("==================================================")
	if len(plan.ToolCalls) > 0 {
		fmt.Println("Planned Tool Operations:")
		for i, tc := range plan.ToolCalls {
			fmt.Printf(" %d. %s (args: %s)
", i+1, tc.Name, string(tc.Arguments))
		}
	}
	if plan.RollbackTaken {
		fmt.Println("WARNING: Execution error occurred. System automatically rolled back to clean state!")
	}
}

func handleCURLImport(cmd string) {
	fmt.Println("Parsing cURL snippet...")
	parsed, err := config.ParseCURL(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to parse cURL snippet: %v
", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Detected Endpoint: %s
", parsed.Endpoint)
	fmt.Printf("✓ Detected Model:    %s
", parsed.Model)
	if parsed.APIKey != "" {
		fmt.Printf("✓ Detected Token:    %s****
", parsed.APIKey[:min(4, len(parsed.APIKey))])
	}

	fmt.Print("Testing connection to endpoint... ")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	lat, err := parsed.VerifyEndpoint(ctx)
	if err != nil {
		fmt.Printf("❌ Failed: %v
", err)
	} else {
		fmt.Printf("✓ Success (%v latency)
", lat)
	}

	_ = exec.Command("uci", "set", "lucinexus.@api[0].provider=custom").Run()
	_ = exec.Command("uci", "set", fmt.Sprintf("lucinexus.@api[0].endpoint=%s", parsed.Endpoint)).Run()
	_ = exec.Command("uci", "set", fmt.Sprintf("lucinexus.@api[0].model=%s", parsed.Model)).Run()
	if parsed.APIKey != "" {
		_ = exec.Command("uci", "set", fmt.Sprintf("lucinexus.@api[0].key=%s", parsed.APIKey)).Run()
	}
	_ = exec.Command("uci", "commit", "lucinexus").Run()
	fmt.Println("✓ Settings successfully applied to /etc/config/lucinexus!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
