package fastpath

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Intent struct {
	Action      string
	Description string
	Execute     func(ctx context.Context) (string, error)
}

var localIntents = map[string]Intent{
	"restart wifi": {
		Action:      "wifi_restart",
		Description: "Reload and restart wireless radio interfaces",
		Execute: func(ctx context.Context) (string, error) {
			out, err := exec.CommandContext(ctx, "wifi", "reload").CombinedOutput()
			return fmt.Sprintf("Wireless subsystem reloaded: %s", string(out)), err
		},
	},
	"turn on wifi": {
		Action:      "wifi_up",
		Description: "Enable wireless interfaces",
		Execute: func(ctx context.Context) (string, error) {
			_ = exec.CommandContext(ctx, "uci", "set", "wireless.@wifi-device[0].disabled=0").Run()
			_ = exec.CommandContext(ctx, "uci", "commit", "wireless").Run()
			out, err := exec.CommandContext(ctx, "wifi", "reload").CombinedOutput()
			return fmt.Sprintf("WiFi enabled: %s", string(out)), err
		},
	},
	"turn off wifi": {
		Action:      "wifi_down",
		Description: "Disable wireless interfaces",
		Execute: func(ctx context.Context) (string, error) {
			_ = exec.CommandContext(ctx, "uci", "set", "wireless.@wifi-device[0].disabled=1").Run()
			_ = exec.CommandContext(ctx, "uci", "commit", "wireless").Run()
			out, err := exec.CommandContext(ctx, "wifi", "reload").CombinedOutput()
			return fmt.Sprintf("WiFi disabled: %s", string(out)), err
		},
	},
	"renew wan": {
		Action:      "wan_renew",
		Description: "Renew WAN DHCP lease",
		Execute: func(ctx context.Context) (string, error) {
			out, err := exec.CommandContext(ctx, "ifup", "wan").CombinedOutput()
			return fmt.Sprintf("WAN interface re-triggered: %s", string(out)), err
		},
	},
	"ping test": {
		Action:      "ping_test",
		Description: "Quick Internet ICMP check",
		Execute: func(ctx context.Context) (string, error) {
			out, err := exec.CommandContext(ctx, "ping", "-c", "2", "-W", "2", "1.1.1.1").CombinedOutput()
			return string(out), err
		},
	},
	"status": {
		Action:      "status",
		Description: "Quick system uptime and memory status",
		Execute: func(ctx context.Context) (string, error) {
			uptime, _ := exec.CommandContext(ctx, "uptime").Output()
			mem, _ := exec.CommandContext(ctx, "free").Output()
			return fmt.Sprintf("Uptime: %s
Memory:
%s", string(uptime), string(mem)), nil
		},
	},
}

func MatchLocalIntent(query string) (*Intent, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	for pattern, intent := range localIntents {
		if strings.Contains(q, pattern) {
			return &intent, true
		}
	}
	return nil, false
}
