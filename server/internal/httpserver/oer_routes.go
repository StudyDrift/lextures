package httpserver

import "github.com/go-chi/chi/v5"

func (d Deps) registerOERRoutes(r chi.Router) {
	r.Get("/api/v1/oer/providers", d.handleGetOERProviders())
	r.Get("/api/v1/oer/search", d.handleGetOERSearch())
}
