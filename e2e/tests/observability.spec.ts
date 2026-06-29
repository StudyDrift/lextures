/**
 * Observability — Prometheus metrics endpoint (plan 17.7).
 *
 * Proves the internal /metrics endpoint serves valid Prometheus exposition
 * format including HTTP request metrics (AC-1) and that it is NOT exposed on the
 * public API port (FR-1 / AC-6: /metrics lives on a separate internal port).
 */
import { test, expect } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'
const metricsBase = process.env.E2E_METRICS_URL ?? 'http://localhost:9090'

test.describe('Observability — metrics endpoint', () => {
  test('serves Prometheus exposition with HTTP metrics (AC-1)', async () => {
    // Generate traffic so the HTTP histograms/counters have data points.
    for (let i = 0; i < 3; i++) {
      await fetch(`${apiBase}/health`)
    }

    const res = await fetch(`${metricsBase}/metrics`)
    expect(res.ok).toBeTruthy()
    const body = await res.text()

    // AC-1: response includes http request total and duration metrics.
    expect(body).toContain('lextures_http_requests_total')
    expect(body).toContain('lextures_http_request_duration_seconds')
    // build_info is always present.
    expect(body).toContain('lextures_build_info')
    // Resource collectors expose DB pool series.
    expect(body).toContain('lextures_db_pool_max_connections')
    // Standard Prometheus exposition uses HELP/TYPE comment lines.
    expect(body).toContain('# TYPE lextures_http_requests_total counter')
  })

  test('metrics reflect requests with bounded (route-grouped) labels', async () => {
    await fetch(`${apiBase}/health`)
    const body = await (await fetch(`${metricsBase}/metrics`)).text()
    // The /health route is recorded with a 2xx status class, not a raw status code.
    expect(body).toMatch(/lextures_http_requests_total\{[^}]*route="\/health"[^}]*status="2xx"[^}]*\}/)
    // No raw numeric status codes leak into the status label (low cardinality).
    expect(body).not.toMatch(/status="200"/)
  })

  test('/metrics is NOT served on the public API port (FR-1 / AC-6)', async () => {
    const res = await fetch(`${apiBase}/metrics`)
    // The public API router has no /metrics route; it must not expose it.
    expect(res.status).toBe(404)
    expect(res.ok).toBeFalsy()
  })
})
