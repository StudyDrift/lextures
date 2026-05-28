// Package middleware provides HTTP middleware; ai_gateway.go documents the AI gateway integration surface (plan 10.17).
// Enforcement is implemented in httpserver via Deps.enforceAIGateway — this file exists so route wiring has a stable import path.
package middleware
