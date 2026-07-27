package contenttools

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
)

const (
	// EnvAIKillSwitch disables all AI-capable Content Tools (CT.8 FR-16).
	EnvAIKillSwitch = "CONTENT_TOOLS_AI_KILL_SWITCH"

	PolicyDenialToolID     = "tool_id"
	PolicyDenialCapability = "capability"
	PolicyDenialAllowlist  = "allowlist"
	PolicyDenialKillTool   = "kill_tool"
	PolicyDenialKillCap    = "kill_capability"
	PolicyDenialKillAI     = "kill_all_ai"
	PolicyDenialKillInst   = "kill_instance"
	PolicyDenialAIOptOut   = "ai_opt_out"
	PolicyDenialCOPPA      = "coppa"
)

var (
	ErrPolicyDenied = errors.New("content tool denied by org policy")
	ErrKillPath     = errors.New("content tool temporarily unavailable (incident kill path)")
)

// PolicyDecision is the result of evaluating org + kill-path policy for a tool.
type PolicyDecision struct {
	Allowed bool
	Reason  string // machine code when denied
	Detail  string // human-readable
}

type killCache struct {
	mu      sync.RWMutex
	loaded  time.Time
	allAI   bool
	tools   map[string]struct{}
	caps    map[string]struct{}
	inst    map[string]struct{}
}

var durableKills = &killCache{
	tools: map[string]struct{}{},
	caps:  map[string]struct{}{},
	inst:  map[string]struct{}{},
}

// SyncDurableKillsFromDB refreshes the process-local kill cache (throttled to 5s).
func SyncDurableKillsFromDB(ctx context.Context, pool *pgxpool.Pool) {
	syncDurableKills(ctx, pool, false)
}

// ForceSyncDurableKillsFromDB refreshes the kill cache immediately (after admin kill writes).
func ForceSyncDurableKillsFromDB(ctx context.Context, pool *pgxpool.Pool) {
	syncDurableKills(ctx, pool, true)
}

func syncDurableKills(ctx context.Context, pool *pgxpool.Pool, force bool) {
	if pool == nil {
		return
	}
	if !force {
		durableKills.mu.RLock()
		fresh := time.Since(durableKills.loaded) < 5*time.Second && !durableKills.loaded.IsZero()
		durableKills.mu.RUnlock()
		if fresh {
			return
		}
	}
	rows, err := ctrepo.ListActiveKills(ctx, pool)
	if err != nil {
		return
	}
	next := &killCache{
		loaded: time.Now().UTC(),
		tools:  map[string]struct{}{},
		caps:   map[string]struct{}{},
		inst:   map[string]struct{}{},
	}
	for _, r := range rows {
		switch r.Scope {
		case "all_ai":
			next.allAI = true
		case "tool":
			next.tools[r.Target] = struct{}{}
		case "capability":
			next.caps[r.Target] = struct{}{}
		case "instance":
			next.inst[r.Target] = struct{}{}
		}
	}
	durableKills.mu.Lock()
	*durableKills = *next
	durableKills.mu.Unlock()
}

// AIKillSwitchEngaged reports env or durable all-AI kill.
func AIKillSwitchEngaged() bool {
	v := strings.TrimSpace(os.Getenv(EnvAIKillSwitch))
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	}
	durableKills.mu.RLock()
	defer durableKills.mu.RUnlock()
	return durableKills.allAI
}

// EvaluateToolPolicy checks org policy + kill path for a tool (server-side; FR-3 / NFR Security).
func EvaluateToolPolicy(policy *ctrepo.PolicyRow, m *CompiledManifest, instanceID *uuid.UUID) PolicyDecision {
	if m == nil {
		return PolicyDecision{Allowed: false, Reason: PolicyDenialToolID, Detail: "tool not found"}
	}
	if AIKillSwitchEngaged() {
		classes := CapabilityClassesForManifest(m)
		for _, c := range classes {
			if c == "ai" {
				IncPolicyDenial(PolicyDenialKillAI)
				return PolicyDecision{Allowed: false, Reason: PolicyDenialKillAI, Detail: "AI tools are temporarily unavailable."}
			}
		}
	}
	durableKills.mu.RLock()
	_, toolKilled := durableKills.tools[m.ID]
	caps := durableKills.caps
	inst := durableKills.inst
	durableKills.mu.RUnlock()
	if toolKilled {
		IncPolicyDenial(PolicyDenialKillTool)
		return PolicyDecision{Allowed: false, Reason: PolicyDenialKillTool, Detail: "This tool is temporarily unavailable."}
	}
	if instanceID != nil {
		if _, ok := inst[instanceID.String()]; ok {
			IncPolicyDenial(PolicyDenialKillInst)
			return PolicyDecision{Allowed: false, Reason: PolicyDenialKillInst, Detail: "This tool instance is temporarily unavailable."}
		}
	}
	classes := CapabilityClassesForManifest(m)
	for _, c := range classes {
		if _, ok := caps[c]; ok {
			IncPolicyDenial(PolicyDenialKillCap)
			return PolicyDecision{Allowed: false, Reason: PolicyDenialKillCap, Detail: "This tool capability is temporarily unavailable."}
		}
	}

	if policy == nil {
		return PolicyDecision{Allowed: true}
	}
	if len(policy.AllowedToolIDs) > 0 {
		ok := false
		for _, id := range policy.AllowedToolIDs {
			if id == m.ID {
				ok = true
				break
			}
		}
		if !ok {
			IncPolicyDenial(PolicyDenialAllowlist)
			return PolicyDecision{Allowed: false, Reason: PolicyDenialAllowlist, Detail: "Tool is not on the organization allowlist."}
		}
	}
	for _, id := range policy.DeniedToolIDs {
		if id == m.ID {
			IncPolicyDenial(PolicyDenialToolID)
			return PolicyDecision{Allowed: false, Reason: PolicyDenialToolID, Detail: "Tool is denied by organization policy."}
		}
	}
	denied := map[string]struct{}{}
	for _, c := range policy.DeniedCapabilities {
		denied[c] = struct{}{}
	}
	for _, c := range classes {
		if _, ok := denied[c]; ok {
			IncPolicyDenial(PolicyDenialCapability)
			return PolicyDecision{Allowed: false, Reason: PolicyDenialCapability, Detail: "Tool capability is denied by organization policy."}
		}
	}
	return PolicyDecision{Allowed: true}
}

// OrgPolicyDeniedSet builds the denied-tool map for FilterCatalog.
func OrgPolicyDeniedSet(policy *ctrepo.PolicyRow, reg *Registry) map[string]struct{} {
	out := map[string]struct{}{}
	if reg == nil {
		return out
	}
	for _, m := range reg.List() {
		dec := EvaluateToolPolicy(policy, m, nil)
		if !dec.Allowed {
			out[m.ID] = struct{}{}
		}
	}
	return out
}

// LoadOrgPolicy loads policy for a course's org (defaults when unset).
func LoadOrgPolicy(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (*ctrepo.PolicyRow, error) {
	row, err := ctrepo.GetPolicy(ctx, pool, orgID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		def := ctrepo.DefaultPolicy(orgID)
		return &def, nil
	}
	return row, nil
}

// NormalizePolicyCapability validates a capability class token.
func NormalizePolicyCapability(raw string) (string, bool) {
	c := strings.TrimSpace(strings.ToLower(raw))
	_, ok := PolicyCapabilityClasses[c]
	return c, ok
}
