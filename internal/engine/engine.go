package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/lucinexus/lucinexus/internal/config"
	"github.com/lucinexus/lucinexus/internal/fastpath"
	"github.com/lucinexus/lucinexus/internal/llm"
	"github.com/lucinexus/lucinexus/internal/openwrt"
	"github.com/lucinexus/lucinexus/internal/policy"
	"github.com/lucinexus/lucinexus/internal/tools"
	"github.com/lucinexus/lucinexus/internal/transaction"
)

type ExecutionPlan struct {
	Query         string             `json:"query"`
	FastpathUsed  bool               `json:"fastpath_used"`
	Summary       string             `json:"summary"`
	ToolCalls     []tools.ToolCall   `json:"tool_calls"`
	Results       []tools.ToolResult `json:"results,omitempty"`
	RollbackTaken bool               `json:"rollback_taken"`
}

type Orchestrator struct {
	cfg    *config.Config
	policy *policy.PolicyEngine
	client llm.LLMClient
}

func NewOrchestrator(cfg *config.Config) (*Orchestrator, error) {
	var client llm.LLMClient
	var err error
	if cfg.APIKey != "" || cfg.Provider == config.ProviderLocal || cfg.Provider == config.ProviderCustom {
		client, err = llm.NewClient(cfg)
		if err != nil {
			return nil, err
		}
	}
	return &Orchestrator{
		cfg:    cfg,
		policy: policy.NewPolicyEngine(),
		client: client,
	}, nil
}

func (o *Orchestrator) Execute(ctx context.Context, query string, dryRun bool) (*ExecutionPlan, error) {
	plan := &ExecutionPlan{Query: query}

	if o.cfg.EnableFastpath {
		if intent, ok := fastpath.MatchLocalIntent(query); ok {
			plan.FastpathUsed = true
			plan.Summary = intent.Description
			if dryRun {
				plan.Summary += " [DRY RUN - ACTION NOT APPLIED]"
				return plan, nil
			}
			out, err := intent.Execute(ctx)
			if err != nil {
				return plan, fmt.Errorf("fastpath %s failed: %w", intent.Action, err)
			}
			plan.Summary = fmt.Sprintf("Fastpath: %s", out)
			return plan, nil
		}
	}

	if o.client == nil {
		return nil, fmt.Errorf("LLM API key not configured and command did not match local offline fastpath")
	}

	subsystems := openwrt.ClassifyIntent(query)
	contextData, _ := openwrt.GetSparseContext(ctx, subsystems)

	systemPrompt := fmt.Sprintf(`You are LuciNexus, the autonomous OpenWrt router agent.
Use structured tool calls to fulfill the request. NEVER output raw shell code.
Router context:
%s`, contextData)

	availableTools := tools.GetAvailableToolDefinitions()
	calls, summary, err := o.client.Plan(ctx, systemPrompt, query, availableTools)
	if err != nil {
		return nil, fmt.Errorf("planning failed: %w", err)
	}
	plan.ToolCalls = calls
	plan.Summary = summary

	if dryRun || len(calls) == 0 {
		return plan, nil
	}

	var affectedPackages []string
	for _, sub := range subsystems {
		affectedPackages = append(affectedPackages, string(sub))
	}
	snap, _ := transaction.CreateSnapshot(affectedPackages)
	if snap != nil {
		defer snap.Discard()
	}

	for _, tc := range calls {
		if err := o.policy.AuthorizeToolCall(tc); err != nil {
			return plan, fmt.Errorf("policy rejection: %w", err)
		}
		res := tools.ExecuteToolCall(ctx, tc)
		plan.Results = append(plan.Results, res)
		if !res.Success {
			if o.cfg.AutoRollback && snap != nil {
				_ = snap.Rollback(ctx)
				plan.RollbackTaken = true
			}
			return plan, fmt.Errorf("step %s failed: %s (auto-rollback triggered)", tc.Name, res.Error)
		}
	}

	if snap != nil && o.cfg.AutoRollback {
		hctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if err := snap.HealthCheck(hctx); err != nil {
			_ = snap.Rollback(ctx)
			plan.RollbackTaken = true
			return plan, fmt.Errorf("health check failed: %w (changes reverted)", err)
		}
	}

	return plan, nil
}
