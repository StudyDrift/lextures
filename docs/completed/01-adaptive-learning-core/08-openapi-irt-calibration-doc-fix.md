# 08 — OpenAPI says IRT calibration is a "no-op" but it is actually implemented

- **Category:** Documentation bug (stale)
- **Severity:** P4
- **Area:** Adaptive learning / IRT difficulty calibration (plan 1.6)

## Summary

The OpenAPI summary for `POST /api/v1/admin/jobs/irt-calibrate` claims the 2PL calibration
body is "not yet ported in Go — background is a no-op log." That is **stale**: the job is
fully implemented (grid search + hill-climb on the marginal 2PL log-likelihood, persisting
fitted `a`/`b` and `irt_status = 'calibrated'`). The misleading doc could cause an
integrator to assume calibration does nothing.

## Evidence

Stale doc — `server/internal/openapi/openapi.go:404`:

```json
"summary": "Start IRT calibration (202 + jobId; 2PL body not yet ported in Go — background is a no-op log)"
```

Actual implementation — `server/internal/service/irtcalibration/background.go`:

```go
// runIRTCalibration: queries uncalibrated/pilot active questions, loads binary responses,
// requires >= 200 responses, then:
a, b, ok := irt.Calibrate2plMarginalGrid(bits)
...
updated, err := questionbank.UpdateQuestionIRTFitted(ctx, pool, row.courseID, row.questionID, a, b, int32(len(bits)))
```

`irt.Calibrate2plMarginalGrid` (`server/internal/service/irt/irtmath.go:221`) is a real
grid + hill-climb estimator, and the handler wires the job
(`server/internal/httpserver/admin.go:86` → `irtcalibration.RunInBackground(...)`).

> Note: the unrelated `server/internal/service/irtcalibrationjob/` package **is** a
> `Health()`-only stub, but it is not the package the endpoint uses — the live path is
> `service/irtcalibration`. The naming collision likely seeded the stale summary.

## Impact

- Misleading API documentation only; runtime behaviour is correct.

## Suggested fix

- Update the OpenAPI summary to describe the real behaviour (e.g. "Start IRT 2PL
  calibration job; fits a/b for active items with ≥200 responses; returns 202 + jobId").
- Consider deleting the dead `service/irtcalibrationjob` stub package to remove the
  naming confusion.

## Acceptance criteria

- OpenAPI no longer describes the endpoint as a no-op.
