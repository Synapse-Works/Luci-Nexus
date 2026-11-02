package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type ToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type ToolCall struct {
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id,omitempty"`
	Name       string `json:"name"`
	Success    bool   `json:"success"`
	Output     string `json:"output"`
	Error      string `json:"error,omitempty"`
}

var safeNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

func GetAvailableToolDefinitions() []ToolDef {
	return []ToolDef{
		{
			Name:        "uci_get",
			Description: "Query a specific UCI configuration value or section",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"package": map[string]string{"type": "string", "description": "UCI package, e.g. wireless, network, firewall"},
					"section": map[string]string{"type": "string", "description": "UCI section name or type (optional)"},
					"option":  map[string]string{"type": "string", "description": "UCI option name (optional)"},
				},
				"required": []string{"package"},
			},
		},
		{
			Name:        "uci_set",
			Description: "Stage a UCI configuration change (requires uci_commit to persist)",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path":  map[string]string{"type": "string", "description": "Full UCI path, e.g. wireless.default_radio0.disabled"},
					"value": map[string]string{"type": "string", "description": "Value to assign"},
				},
				"required": []string{"path", "value"},
			},
		},
		{
			Name:        "uci_commit",
			Description: "Persist staged UCI configuration changes to flash storage",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"package": map[string]string{"type": "string", "description": "Package name to commit"},
				},
				"required": []string{"package"},
			},
		},
		{
			Name:        "service_control",
			Description: "Safely reload, restart, start, or stop an OpenWrt init service",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"service": map[string]string{"type": "string", "description": "Service name, e.g. network, firewall, dnsmasq"},
					"action":  map[string]string{"type": "string", "description": "Action: reload, restart, start, stop"},
				},
				"required": []string{"service", "action"},
			},
		},
		{
			Name:        "network_diagnose",
			Description: "Run safe ICMP ping diagnostics or DNS verification",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target": map[string]string{"type": "string", "description": "Target IP or hostname"},
					"count":  map[string]string{"type": "integer", "description": "Packet count (1-5)"},
				},
				"required": []string{"target"},
			},
		},
		{
			Name:        "system_status",
			Description: "Retrieve router uptime, load average, and free memory",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

func ExecuteToolCall(ctx context.Context, tc ToolCall) ToolResult {
	res := ToolResult{ToolCallID: tc.ID, Name: tc.Name}

	switch tc.Name {
	case "uci_get":
		var args struct {
			Package string `json:"package"`
			Section string `json:"section"`
			Option  string `json:"option"`
		}
		if err := json.Unmarshal(tc.Arguments, &args); err != nil {
			res.Error = "invalid arguments: " + err.Error()
			return res
		}
		query := args.Package
		if args.Section != "" {
			query += "." + args.Section
			if args.Option != "" {
				query += "." + args.Option
			}
		}
		out, err := runCommand(ctx, "uci", "get", query)
		if err != nil {
			res.Error = err.Error()
			res.Output = out
			return res
		}
		res.Success = true
		res.Output = strings.TrimSpace(out)

	case "uci_set":
		var args struct {
			Path  string `json:"path"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal(tc.Arguments, &args); err != nil {
			res.Error = "invalid arguments: " + err.Error()
			return res
		}
		assign := fmt.Sprintf("%s=%s", args.Path, args.Value)
		out, err := runCommand(ctx, "uci", "set", assign)
		if err != nil {
			res.Error = err.Error()
			res.Output = out
			return res
		}
		res.Success = true
		res.Output = fmt.Sprintf("staged %s", assign)

	case "uci_commit":
		var args struct {
			Package string `json:"package"`
		}
		if err := json.Unmarshal(tc.Arguments, &args); err != nil {
			res.Error = "invalid arguments: " + err.Error()
			return res
		}
		out, err := runCommand(ctx, "uci", "commit", args.Package)
		if err != nil {
			res.Error = err.Error()
			return res
		}
		res.Success = true
		res.Output = fmt.Sprintf("committed %s: %s", args.Package, out)

	case "service_control":
		var args struct {
			Service string `json:"service"`
			Action  string `json:"action"`
		}
		if err := json.Unmarshal(tc.Arguments, &args); err != nil {
			res.Error = "invalid arguments: " + err.Error()
			return res
		}
		if !safeNameRegex.MatchString(args.Service) {
			res.Error = "invalid service identifier"
			return res
		}
		servicePath := fmt.Sprintf("/etc/init.d/%s", args.Service)
		out, err := runCommand(ctx, servicePath, args.Action)
		if err != nil {
			res.Error = err.Error()
			return res
		}
		res.Success = true
		res.Output = strings.TrimSpace(out)

	case "network_diagnose":
		var args struct {
			Target string `json:"target"`
			Count  int    `json:"count"`
		}
		if err := json.Unmarshal(tc.Arguments, &args); err != nil {
			res.Error = "invalid arguments: " + err.Error()
			return res
		}
		if args.Count <= 0 || args.Count > 5 {
			args.Count = 2
		}
		out, err := runCommand(ctx, "ping", "-c", fmt.Sprintf("%d", args.Count), "-W", "2", args.Target)
		if err != nil {
			res.Error = err.Error()
			return res
		}
		res.Success = true
		res.Output = strings.TrimSpace(out)

	case "system_status":
		uptimeOut, _ := runCommand(ctx, "uptime")
		freeOut, _ := runCommand(ctx, "free")
		res.Success = true
		res.Output = fmt.Sprintf("UPTIME: %s | MEMORY: %s", strings.TrimSpace(uptimeOut), strings.TrimSpace(freeOut))

	default:
		res.Error = fmt.Sprintf("unsupported tool: %s", tc.Name)
	}

	return res
}

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	c, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	cmd := exec.CommandContext(c, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return stdout.String() + " " + stderr.String(), fmt.Errorf("%s %v failed: %w", name, args, err)
	}
	return stdout.String(), nil
}
