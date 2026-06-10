package ccr

import "expvar"

var (
	ccrGeneratedTotal     expvar.Int
	ccrVerificationsTotal expvar.Int
	ccrValidVerifications expvar.Int
)

func init() {
	expvar.Publish("ccr_generated_total", &ccrGeneratedTotal)
	expvar.Publish("ccr_verifications_total", &ccrVerificationsTotal)
	expvar.Publish("ccr_valid_verifications_total", &ccrValidVerifications)
}

func recordGenerated() {
	ccrGeneratedTotal.Add(1)
}

func recordVerification(valid bool) {
	ccrVerificationsTotal.Add(1)
	if valid {
		ccrValidVerifications.Add(1)
	}
}
