package parameter_explorer

import (
	"fmt"
	"math"
)

func boolNum(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func callFn(name string, args []float64) (float64, error) {
	need := func(n int) error {
		if len(args) != n {
			return &EvalError{Code: "arity", Message: fmt.Sprintf("%s expects %d args", name, n)}
		}
		return nil
	}
	switch name {
	case "abs":
		if err := need(1); err != nil {
			return 0, err
		}
		return math.Abs(args[0]), nil
	case "sqrt":
		if err := need(1); err != nil {
			return 0, err
		}
		if args[0] < 0 {
			return math.NaN(), nil
		}
		return math.Sqrt(args[0]), nil
	case "sin":
		if err := need(1); err != nil {
			return 0, err
		}
		return math.Sin(args[0]), nil
	case "cos":
		if err := need(1); err != nil {
			return 0, err
		}
		return math.Cos(args[0]), nil
	case "tan":
		if err := need(1); err != nil {
			return 0, err
		}
		return math.Tan(args[0]), nil
	case "asin":
		if err := need(1); err != nil {
			return 0, err
		}
		return math.Asin(args[0]), nil
	case "acos":
		if err := need(1); err != nil {
			return 0, err
		}
		return math.Acos(args[0]), nil
	case "atan":
		if err := need(1); err != nil {
			return 0, err
		}
		return math.Atan(args[0]), nil
	case "ln", "log":
		if err := need(1); err != nil {
			return 0, err
		}
		if args[0] <= 0 {
			return math.NaN(), nil
		}
		return math.Log(args[0]), nil
	case "log10":
		if err := need(1); err != nil {
			return 0, err
		}
		if args[0] <= 0 {
			return math.NaN(), nil
		}
		return math.Log10(args[0]), nil
	case "exp":
		if err := need(1); err != nil {
			return 0, err
		}
		if args[0] > 700 {
			return 0, &EvalError{Code: "overflow", Message: "exp overflow"}
		}
		return math.Exp(args[0]), nil
	case "floor":
		if err := need(1); err != nil {
			return 0, err
		}
		return math.Floor(args[0]), nil
	case "ceil":
		if err := need(1); err != nil {
			return 0, err
		}
		return math.Ceil(args[0]), nil
	case "round":
		if err := need(1); err != nil {
			return 0, err
		}
		return math.Round(args[0]), nil
	case "sign":
		if err := need(1); err != nil {
			return 0, err
		}
		if args[0] > 0 {
			return 1, nil
		}
		if args[0] < 0 {
			return -1, nil
		}
		return 0, nil
	case "min":
		if len(args) < 1 {
			return 0, &EvalError{Code: "arity", Message: "min expects at least 1 arg"}
		}
		m := args[0]
		for _, a := range args[1:] {
			m = math.Min(m, a)
		}
		return m, nil
	case "max":
		if len(args) < 1 {
			return 0, &EvalError{Code: "arity", Message: "max expects at least 1 arg"}
		}
		m := args[0]
		for _, a := range args[1:] {
			m = math.Max(m, a)
		}
		return m, nil
	case "pow":
		if err := need(2); err != nil {
			return 0, err
		}
		if math.Abs(args[1]) > 1000 {
			return 0, &EvalError{Code: "exponent", Message: "exponent too large"}
		}
		return math.Pow(args[0], args[1]), nil
	case "hypot":
		if err := need(2); err != nil {
			return 0, err
		}
		return math.Hypot(args[0], args[1]), nil
	case "clamp":
		if err := need(3); err != nil {
			return 0, err
		}
		return math.Min(args[2], math.Max(args[0], args[1])), nil
	default:
		return 0, &EvalError{Code: "unknown_fn", Message: fmt.Sprintf("unknown function %q", name)}
	}
}
