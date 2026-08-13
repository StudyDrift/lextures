package kernel

import (
	"net/http"
	"reflect"
)

// None is the input type for handlers that do not read a JSON body.
type None struct{}

// Option customises a typed handler wrapper.
type Option func(*handlerConfig)

type handlerConfig struct {
	decode        DecodeOptions
	successStatus int
	opName        string
}

// WithDecodeOptions sets JSON decode policy. Converted handlers should match
// the previous endpoint (typically no Content-Type check, unknown fields ignored).
func WithDecodeOptions(opts DecodeOptions) Option {
	return func(c *handlerConfig) { c.decode = opts }
}

// WithStatus sets the success status (default 200). Use 201 for creates.
func WithStatus(code int) Option {
	return func(c *handlerConfig) { c.successStatus = code }
}

// WithName labels the operation for the unguarded-route registry.
func WithName(name string) Option {
	return func(c *handlerConfig) { c.opName = name }
}

func applyOptions(opts []Option) handlerConfig {
	cfg := handlerConfig{successStatus: http.StatusOK}
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	return cfg
}

// POST decodes In, runs guard, then h. Default success status is 200.
func POST[In, Out any](access Access, guard Guard, h func(*Ctx, In) (Out, error), opts ...Option) http.HandlerFunc {
	return wrap(http.MethodPost, access, guard, h, opts)
}

// PUT decodes In, runs guard, then h.
func PUT[In, Out any](access Access, guard Guard, h func(*Ctx, In) (Out, error), opts ...Option) http.HandlerFunc {
	return wrap(http.MethodPut, access, guard, h, opts)
}

// PATCH decodes In, runs guard, then h.
func PATCH[In, Out any](access Access, guard Guard, h func(*Ctx, In) (Out, error), opts ...Option) http.HandlerFunc {
	return wrap(http.MethodPatch, access, guard, h, opts)
}

// GET runs guard then h. In is unused; pass a func(*Ctx) (Out, error) via GETOut.
func GET[Out any](access Access, guard Guard, h func(*Ctx) (Out, error), opts ...Option) http.HandlerFunc {
	return wrap(http.MethodGet, access, guard, func(c *Ctx, _ None) (Out, error) {
		return h(c)
	}, opts)
}

// DELETE runs guard then h.
func DELETE[Out any](access Access, guard Guard, h func(*Ctx) (Out, error), opts ...Option) http.HandlerFunc {
	return wrap(http.MethodDelete, access, guard, func(c *Ctx, _ None) (Out, error) {
		return h(c)
	}, opts)
}

func wrap[In, Out any](method string, access Access, guard Guard, h func(*Ctx, In) (Out, error), opts []Option) http.HandlerFunc {
	cfg := applyOptions(opts)
	resolved := guard
	unguarded := false
	if resolved.run == nil {
		resolved = Authenticated()
		unguarded = true
	}
	if resolved.Public() {
		unguarded = true
	}
	name := cfg.opName
	if name == "" {
		name = method + " " + resolved.Name()
	}
	registerRoute(RouteInfo{Name: name, Method: method, Guard: resolved.Name(), Unguarded: unguarded})

	skipBody := isNone[In]()
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := &Ctx{Context: r.Context(), W: w, R: r, Access: access}
		if err := resolved.Check(ctx); err != nil {
			WriteError(w, r, err)
			return
		}
		var in In
		if !skipBody {
			if err := DecodeJSON(w, r, &in, cfg.decode); err != nil {
				WriteError(w, r, err)
				DrainBody(r)
				return
			}
		}
		out, err := h(ctx, in)
		if err != nil {
			WriteError(w, r, err)
			return
		}
		status := cfg.successStatus
		if ctx.status != 0 {
			status = ctx.status
		}
		if status == http.StatusNoContent {
			w.WriteHeader(status)
			return
		}
		EncodeJSON(w, status, out)
	}
}

func isNone[T any]() bool {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		return false
	}
	return t == reflect.TypeOf(None{})
}

// InputType returns the reflect type of In for OpenAPI generation (FR-12).
func InputType[In, Out any]() reflect.Type {
	return reflect.TypeOf((*In)(nil)).Elem()
}

// OutputType returns the reflect type of Out for OpenAPI generation (FR-12).
func OutputType[In, Out any]() reflect.Type {
	return reflect.TypeOf((*Out)(nil)).Elem()
}
