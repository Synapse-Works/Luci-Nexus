package openwrt

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Subsystem string

const (
	SubsystemWireless Subsystem = "wireless"
	SubsystemNetwork  Subsystem = "network"
	SubsystemFirewall Subsystem = "firewall"
	SubsystemDHCP     Subsystem = "dhcp"
	SubsystemSystem   Subsystem = "system"
)

func ClassifyIntent(query string) []Subsystem {
	q := strings.ToLower(query)
	var sub []Subsystem

	if strings.Contains(q, "wifi") || strings.Contains(q, "wireless") || strings.Contains(q, "ssid") || strings.Contains(q, "password") {
		sub = append(sub, SubsystemWireless)
	}
	if strings.Contains(q, "ip") || strings.Contains(q, "lan") || strings.Contains(q, "wan") || strings.Contains(q, "interface") || strings.Contains(q, "route") {
		sub = append(sub, SubsystemNetwork)
	}
	if strings.Contains(q, "port") || strings.Contains(q, "firewall") || strings.Contains(q, "block") || strings.Contains(q, "forward") {
		sub = append(sub, SubsystemFirewall)
	}
	if strings.Contains(q, "dhcp") || strings.Contains(q, "lease") || strings.Contains(q, "dns") {
		sub = append(sub, SubsystemDHCP)
	}

	if len(sub) == 0 {
		sub = []Subsystem{SubsystemSystem, SubsystemNetwork}
	}
	return sub
}

func GetSparseContext(ctx context.Context, subsystems []Subsystem) (string, error) {
	c, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	var sb strings.Builder
	sb.WriteString("=== ROUTER CONTEXT ===
")

	for _, s := range subsystems {
		sb.WriteString(fmt.Sprintf("
--- SUBSYSTEM: %s ---
", s))
		out, err := exec.CommandContext(c, "uci", "show", string(s)).CombinedOutput()
		if err == nil && len(out) > 0 {
			if len(out) > 4096 {
				sb.Write(out[:4096])
				sb.WriteString("
[...truncated...]")
			} else {
				sb.Write(out)
			}
		}
	}

	return sb.String(), nil
}
