package parameter_explorer

// PresetID identifies a built-in model.
type PresetID string

const (
	PresetLinear           PresetID = "linear"
	PresetQuadratic        PresetID = "quadratic"
	PresetExponential      PresetID = "exponential"
	PresetLogistic         PresetID = "logistic"
	PresetProjectile       PresetID = "projectile"
	PresetSupplyDemand     PresetID = "supply_demand"
	PresetNormal           PresetID = "normal"
	PresetCompoundInterest PresetID = "compound_interest"
)

// PresetDef is a data-driven model entry (expression + sweep defaults + bind slots).
type PresetDef struct {
	ID          PresetID
	Expression  string // in terms of sweep var `x` and slot names
	SweepFrom   float64
	SweepTo     float64
	SweepPoints int
	Slots       []string // bind keys authors map to parameter ids
	YLabel      string
	XLabel      string
}

// PresetLibrary is the built-in model catalog (data, not code branches for new entries).
var PresetLibrary = []PresetDef{
	{
		ID: PresetLinear, Expression: "m * x + b",
		SweepFrom: -10, SweepTo: 10, SweepPoints: 101,
		Slots: []string{"m", "b"}, XLabel: "x", YLabel: "y",
	},
	{
		ID: PresetQuadratic, Expression: "a * x^2 + b * x + c",
		SweepFrom: -10, SweepTo: 10, SweepPoints: 101,
		Slots: []string{"a", "b", "c"}, XLabel: "x", YLabel: "y",
	},
	{
		ID: PresetExponential, Expression: "a * exp(k * x)",
		SweepFrom: -2, SweepTo: 4, SweepPoints: 101,
		Slots: []string{"a", "k"}, XLabel: "x", YLabel: "y",
	},
	{
		ID: PresetLogistic, Expression: "K / (1 + ((K - P0) / P0) * exp(-r * x))",
		SweepFrom: 0, SweepTo: 40, SweepPoints: 121,
		Slots: []string{"K", "P0", "r"}, XLabel: "t", YLabel: "P",
	},
	{
		ID: PresetProjectile, Expression: "x * tan(theta) - (g * x^2) / (2 * v0^2 * cos(theta)^2)",
		SweepFrom: 0, SweepTo: 80, SweepPoints: 101,
		Slots: []string{"v0", "theta", "g"}, XLabel: "x", YLabel: "y",
	},
	{
		ID: PresetSupplyDemand, Expression: "(a - b * x) - (c + d * x)", // surplus = demand - supply
		SweepFrom: 0, SweepTo: 20, SweepPoints: 101,
		Slots: []string{"a", "b", "c", "d"}, XLabel: "Q", YLabel: "surplus",
	},
	{
		ID: PresetNormal, Expression: "(1 / (sigma * sqrt(2 * pi))) * exp(-0.5 * ((x - mu) / sigma)^2)",
		SweepFrom: -5, SweepTo: 5, SweepPoints: 121,
		Slots: []string{"mu", "sigma"}, XLabel: "x", YLabel: "pdf",
	},
	{
		ID: PresetCompoundInterest, Expression: "P * (1 + r / n)^(n * x)",
		SweepFrom: 0, SweepTo: 20, SweepPoints: 101,
		Slots: []string{"P", "r", "n"}, XLabel: "t", YLabel: "A",
	},
}

// LookupPreset returns a preset definition by id.
func LookupPreset(id string) (PresetDef, bool) {
	for _, p := range PresetLibrary {
		if string(p.ID) == id {
			return p, true
		}
	}
	return PresetDef{}, false
}

// ResolveBoundVars maps bind slots → parameter values into expression variables.
// Always includes sweep variable `x`.
func ResolveBoundVars(bind map[string]string, params map[string]float64, x float64) map[string]float64 {
	out := map[string]float64{"x": x}
	for slot, paramID := range bind {
		if v, ok := params[paramID]; ok {
			out[slot] = v
		}
	}
	// also expose all params by their own ids
	for k, v := range params {
		out[k] = v
	}
	return out
}
