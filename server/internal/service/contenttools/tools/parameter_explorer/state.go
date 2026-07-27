package parameter_explorer

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// ParseConfig unmarshals instructor config with defaults applied.
func ParseConfig(raw json.RawMessage) Config {
	cfg := DefaultConfig()
	if len(raw) == 0 {
		return cfg
	}
	var overlay Config
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return cfg
	}
	if strings.TrimSpace(overlay.Prompt) != "" {
		cfg.Prompt = overlay.Prompt
	}
	if overlay.Hint != "" {
		cfg.Hint = overlay.Hint
	}
	if len(overlay.Parameters) > 0 {
		cfg.Parameters = overlay.Parameters
		if len(cfg.Parameters) > 6 {
			cfg.Parameters = cfg.Parameters[:6]
		}
	}
	if overlay.Model.Kind != "" {
		cfg.Model = overlay.Model
	}
	if len(overlay.Outputs) > 0 {
		cfg.Outputs = overlay.Outputs
	}
	if overlay.NoticingPrompts != nil {
		cfg.NoticingPrompts = overlay.NoticingPrompts
		if len(cfg.NoticingPrompts) > 20 {
			cfg.NoticingPrompts = cfg.NoticingPrompts[:20]
		}
	}
	cfg.RequireAllCheckpoints = overlay.RequireAllCheckpoints
	return cfg
}

// ParseState unmarshals learner state with defaults and caps.
func ParseState(raw json.RawMessage) State {
	st := EmptyState()
	if len(raw) == 0 {
		return st
	}
	_ = json.Unmarshal(raw, &st)
	if st.V == 0 {
		st.V = 1
	}
	if st.Params == nil {
		st.Params = map[string]any{}
	}
	if st.Trace == nil {
		st.Trace = []TraceEntry{}
	}
	if len(st.Trace) > MaxTraceEntries {
		st.Trace = st.Trace[len(st.Trace)-MaxTraceEntries:]
	}
	if st.Checkpoints == nil {
		st.Checkpoints = map[string]string{}
	}
	if st.Answers == nil {
		st.Answers = map[string]string{}
	}
	return st
}

// DefaultParams builds the initial parameter map from config.
func DefaultParams(cfg Config) map[string]any {
	out := map[string]any{}
	for _, p := range cfg.Parameters {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			continue
		}
		switch p.Kind {
		case ParamBoolean:
			if b, ok := p.Default.(bool); ok {
				out[id] = b
			} else {
				out[id] = false
			}
		case ParamChoice:
			if s, ok := p.Default.(string); ok {
				out[id] = s
			} else {
				out[id] = ""
			}
		default:
			out[id] = asFloat(p.Default, 0)
		}
	}
	return out
}

// ParamsAsFloats converts numeric/boolean params for expression evaluation.
// Booleans become 0/1; choice params are omitted unless numeric.
func ParamsAsFloats(params map[string]any) map[string]float64 {
	out := map[string]float64{}
	for k, v := range params {
		switch t := v.(type) {
		case float64:
			out[k] = t
		case float32:
			out[k] = float64(t)
		case int:
			out[k] = float64(t)
		case int64:
			out[k] = float64(t)
		case json.Number:
			if f, err := t.Float64(); err == nil {
				out[k] = f
			}
		case bool:
			if t {
				out[k] = 1
			} else {
				out[k] = 0
			}
		case string:
			// skip non-numeric strings
		}
	}
	return out
}

func asFloat(v any, fallback float64) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return fallback
		}
		return f
	default:
		return fallback
	}
}

// ClampParams ensures number params stay within [min,max] and known ids.
func ClampParams(cfg Config, params map[string]any) map[string]any {
	known := map[string]Parameter{}
	for _, p := range cfg.Parameters {
		known[p.ID] = p
	}
	out := map[string]any{}
	for id, p := range known {
		v, ok := params[id]
		if !ok {
			out[id] = DefaultParams(cfg)[id]
			continue
		}
		switch p.Kind {
		case ParamBoolean:
			b, ok := v.(bool)
			if !ok {
				out[id] = false
			} else {
				out[id] = b
			}
		case ParamChoice:
			s, _ := v.(string)
			valid := false
			for _, opt := range p.Options {
				if opt.Value == s {
					valid = true
					break
				}
			}
			if !valid {
				if def, ok := p.Default.(string); ok {
					s = def
				} else if len(p.Options) > 0 {
					s = p.Options[0].Value
				}
			}
			out[id] = s
		default:
			f := asFloat(v, asFloat(p.Default, 0))
			if p.Max >= p.Min {
				f = math.Min(p.Max, math.Max(p.Min, f))
			}
			out[id] = f
		}
	}
	return out
}

// AppendTrace adds a distinct configuration snapshot (downsampled / capped).
func AppendTrace(st State, params map[string]any, at string) State {
	if at == "" {
		at = NowRFC3339()
	}
	// Skip if identical to last entry (downsample rapid moves).
	if len(st.Trace) > 0 {
		last := st.Trace[len(st.Trace)-1]
		if paramsEqual(last.Params, params) {
			return st
		}
	}
	st.Trace = append(st.Trace, TraceEntry{At: at, Params: cloneParams(params)})
	if len(st.Trace) > MaxTraceEntries {
		// Keep first + evenly spaced remainder + last.
		st.Trace = downsampleTrace(st.Trace, MaxTraceEntries)
	}
	return st
}

func paramsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if fmt.Sprint(va) != fmt.Sprint(vb) {
			return false
		}
	}
	return true
}

func cloneParams(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func downsampleTrace(entries []TraceEntry, max int) []TraceEntry {
	if len(entries) <= max {
		return entries
	}
	out := make([]TraceEntry, 0, max)
	out = append(out, entries[0])
	inner := max - 2
	if inner < 1 {
		return []TraceEntry{entries[0], entries[len(entries)-1]}
	}
	step := float64(len(entries)-2) / float64(inner)
	for i := 0; i < inner; i++ {
		idx := 1 + int(math.Floor(float64(i)*step))
		if idx >= len(entries)-1 {
			idx = len(entries) - 2
		}
		out = append(out, entries[idx])
	}
	out = append(out, entries[len(entries)-1])
	return out
}

// NowRFC3339 returns the current UTC time as RFC3339.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// ParamBins returns coarse coverage bins for analytics (CT.7).
// Format: "{paramId}:{bucket}" with 10 buckets over [min,max].
func ParamBins(cfg Config, params map[string]any) []string {
	out := []string{}
	for _, p := range cfg.Parameters {
		if p.Kind != ParamNumber && p.Kind != "" {
			continue
		}
		f, ok := params[p.ID]
		if !ok {
			continue
		}
		val := asFloat(f, 0)
		bucket := 0
		span := p.Max - p.Min
		if span > 0 {
			bucket = int(math.Floor((val - p.Min) / span * 10))
			if bucket < 0 {
				bucket = 0
			}
			if bucket > 9 {
				bucket = 9
			}
		}
		out = append(out, fmt.Sprintf("%s:%d", p.ID, bucket))
	}
	return out
}

// CheckpointPrompts returns prompts that have unlockWhen predicates.
func CheckpointPrompts(cfg Config) []NoticingPrompt {
	out := make([]NoticingPrompt, 0)
	for _, p := range cfg.NoticingPrompts {
		if strings.TrimSpace(p.UnlockWhen) != "" {
			out = append(out, p)
		}
	}
	return out
}

// EvalUnlock evaluates unlockWhen against current params.
func EvalUnlock(predicate string, params map[string]any) (bool, error) {
	predicate = strings.TrimSpace(predicate)
	if predicate == "" {
		return true, nil
	}
	return EvalPredicate(predicate, ParamsAsFloats(params))
}

// IsComplete reports whether all required prompts (and checkpoints) are satisfied.
func IsComplete(cfg Config, st State) bool {
	for _, p := range cfg.NoticingPrompts {
		if !p.Required {
			continue
		}
		if cfg.RequireAllCheckpoints || strings.TrimSpace(p.UnlockWhen) != "" {
			if _, hit := st.Checkpoints[p.ID]; !hit && strings.TrimSpace(p.UnlockWhen) != "" {
				return false
			}
		}
		ans := strings.TrimSpace(st.Answers[p.ID])
		if ans == "" {
			return false
		}
	}
	// If no required prompts, completion is answering all prompts when any exist;
	// otherwise exploring alone does not auto-complete.
	hasRequired := false
	for _, p := range cfg.NoticingPrompts {
		if p.Required {
			hasRequired = true
			break
		}
	}
	if !hasRequired && len(cfg.NoticingPrompts) > 0 {
		for _, p := range cfg.NoticingPrompts {
			if strings.TrimSpace(st.Answers[p.ID]) == "" {
				return false
			}
		}
	}
	if len(cfg.NoticingPrompts) == 0 {
		return false
	}
	return true
}

// ValidateConfigForAuthoring rejects bad expressions / unknown presets / empty params.
func ValidateConfigForAuthoring(cfg Config) error {
	if strings.TrimSpace(cfg.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	if len(cfg.Parameters) < 1 || len(cfg.Parameters) > 6 {
		return fmt.Errorf("parameters: need 1–6")
	}
	ids := map[string]struct{}{}
	for _, p := range cfg.Parameters {
		if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.Label) == "" {
			return fmt.Errorf("each parameter needs id and label")
		}
		if _, dup := ids[p.ID]; dup {
			return fmt.Errorf("duplicate parameter id %q", p.ID)
		}
		ids[p.ID] = struct{}{}
		switch p.Kind {
		case ParamNumber, "":
			if p.Step <= 0 {
				return fmt.Errorf("parameter %q: step must be > 0", p.ID)
			}
			if p.Max < p.Min {
				return fmt.Errorf("parameter %q: max < min", p.ID)
			}
		case ParamBoolean:
		case ParamChoice:
			if len(p.Options) < 2 {
				return fmt.Errorf("parameter %q: choice needs ≥2 options", p.ID)
			}
		default:
			return fmt.Errorf("parameter %q: unknown kind %q", p.ID, p.Kind)
		}
	}
	switch cfg.Model.Kind {
	case "preset":
		def, ok := LookupPreset(cfg.Model.Preset)
		if !ok {
			return fmt.Errorf("unknown preset %q", cfg.Model.Preset)
		}
		for _, slot := range def.Slots {
			pid, ok := cfg.Model.Bind[slot]
			if !ok || strings.TrimSpace(pid) == "" {
				return fmt.Errorf("preset bind missing slot %q", slot)
			}
			if _, known := ids[pid]; !known {
				return fmt.Errorf("bind slot %q references unknown parameter %q", slot, pid)
			}
		}
		// Smoke-eval at sweep midpoint with defaults to catch broken preset expressions.
		defaults := ParamsAsFloats(DefaultParams(cfg))
		mid := (def.SweepFrom + def.SweepTo) / 2
		vars := ResolveBoundVars(cfg.Model.Bind, defaults, mid)
		if _, err := EvalExpression(def.Expression, vars); err != nil {
			return fmt.Errorf("preset %q failed smoke eval: %w", cfg.Model.Preset, err)
		}
	case "expression":
		if err := ValidateExpression(cfg.Model.Expression); err != nil {
			return fmt.Errorf("expression: %w", err)
		}
		if cfg.Model.Sweep == nil {
			return fmt.Errorf("expression model requires sweep")
		}
		if cfg.Model.Sweep.Points < 2 || cfg.Model.Sweep.Points > MaxSweepPoints {
			return fmt.Errorf("sweep points must be 2–%d", MaxSweepPoints)
		}
	default:
		return fmt.Errorf("model.kind must be preset or expression")
	}
	for _, np := range cfg.NoticingPrompts {
		if strings.TrimSpace(np.ID) == "" || strings.TrimSpace(np.Text) == "" {
			return fmt.Errorf("noticing prompt needs id and text")
		}
		if np.UnlockWhen != "" {
			if err := ValidateExpression(np.UnlockWhen); err != nil {
				return fmt.Errorf("prompt %q unlockWhen: %w", np.ID, err)
			}
		}
		if np.Kind == "choice" && len(np.Options) < 2 {
			return fmt.Errorf("prompt %q: choice needs ≥2 options", np.ID)
		}
	}
	return nil
}
