package httpserver

import "github.com/go-chi/chi/v5"

func (d Deps) registerCloudProviderRoutes(r chi.Router) {
	r.Get("/api/v1/cloud-providers", d.handleGetCloudProviders())
}
