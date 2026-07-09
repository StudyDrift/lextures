package background

import (
	introcourseservice "github.com/lextures/lextures/server/internal/service/introcourse"
	"github.com/lextures/lextures/server/internal/config"
)

// RegisterIntroCourseJobs adds intro course enrollment retry and backfill handlers.
func RegisterIntroCourseJobs(r *Registry, svc *introcourseservice.Service, cfgSrc ConfigSource) {
	if r == nil || svc == nil {
		return
	}
	introcourseservice.RegisterJobHandlers(func(jobType string, h introcourseservice.JobHandler) {
		r.Register(jobType, HandlerFunc(h.Execute))
	}, svc, func() config.Config { return cfgSrc.Config() })
}