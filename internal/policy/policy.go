package policy

import (
	"fmt"
	"strings"

	"github.com/lucinexus/lucinexus/internal/tools"
)

type PolicyEngine struct {
	ReadOnly bool
}

func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{ReadOnly: false}
}

func (p *PolicyEngine) AuthorizeToolCall(tc tools.ToolCall) error {
	if p.ReadOnly && (tc.Name == "uci_set" || tc.Name == "uci_commit" || tc.Name == "service_control") {
		return fmt.Errorf("policy error: mutation tool %s blocked in read-only mode", tc.Name)
	}

	switch tc.Name {
	case "uci_get", "service_control", "network_diagnose", "system_status":
		return nil
	case "uci_set", "uci_commit":
		argsStr := string(tc.Arguments)
		if strings.Contains(argsStr, ";") || strings.Contains(argsStr, "|") || strings.Contains(argsStr, "&") || strings.Contains(argsStr, "`") || strings.Contains(argsStr, "$") {
			return fmt.Errorf("policy error: shell metacharacter injection detected in tool argument")
		}
		return nil
	default:
		return fmt.Errorf("policy error: unknown tool %s", tc.Name)
	}
}
