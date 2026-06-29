package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
)

// DBPoolSnapshot is a point-in-time view of the pgx connection pool. app.go
// adapts pgxpool.Stat() into this so telemetry stays decoupled from the DB layer.
type DBPoolSnapshot struct {
	Total        int32
	Acquired     int32
	Idle         int32
	Max          int32
	Constructing int32
}

// RedisPoolSnapshot is a point-in-time view of the Redis client pool.
type RedisPoolSnapshot struct {
	Total    uint32
	Idle     uint32
	Stale    uint32
	Hits     uint64
	Misses   uint64
	Timeouts uint64
}

// JobQueueSnapshot is a point-in-time view of the durable job queue (plan 17.3).
type JobQueueSnapshot struct {
	Pending     int
	Running     int
	Failed      int
	DeadLetters int
	Depth       int
	ByType      map[string]int
}

// Sources are the live snapshot closures the collector reads at scrape time.
// Any field may be nil (e.g. Redis disabled, no DB pool) — the collector simply
// omits those series. Reading at scrape time (pull model) keeps gauges current
// without a background goroutine (plan 17.7 NFR Scalability).
type Sources struct {
	DBPool   func() DBPoolSnapshot
	Redis    func() RedisPoolSnapshot
	JobQueue func() (JobQueueSnapshot, bool)
}

// resourceCollector implements prometheus.Collector and emits DB pool, Redis
// pool, and job-queue gauges by reading Sources at collection time (plan 17.7
// FR-1: DB pool utilization, Redis connection count, job queue depth by type).
type resourceCollector struct {
	sources Sources

	dbTotal        *prometheus.Desc
	dbAcquired     *prometheus.Desc
	dbIdle         *prometheus.Desc
	dbMax          *prometheus.Desc
	dbUtilization  *prometheus.Desc
	redisTotal     *prometheus.Desc
	redisIdle      *prometheus.Desc
	redisStale     *prometheus.Desc
	redisHits      *prometheus.Desc
	redisMisses    *prometheus.Desc
	redisTimeouts  *prometheus.Desc
	jobDepth       *prometheus.Desc
	jobByStatus    *prometheus.Desc
	jobByType      *prometheus.Desc
	jobDeadLetters *prometheus.Desc
}

func newResourceCollector(s Sources) *resourceCollector {
	d := func(name, help string, labels ...string) *prometheus.Desc {
		return prometheus.NewDesc(prometheus.BuildFQName(namespace, "", name), help, labels, nil)
	}
	return &resourceCollector{
		sources:        s,
		dbTotal:        d("db_pool_total_connections", "Total connections in the pgx pool."),
		dbAcquired:     d("db_pool_acquired_connections", "Currently acquired (in-use) pgx connections."),
		dbIdle:         d("db_pool_idle_connections", "Idle pgx connections."),
		dbMax:          d("db_pool_max_connections", "Maximum configured pgx connections."),
		dbUtilization:  d("db_pool_utilization_ratio", "Acquired/max ratio of the pgx pool (0..1)."),
		redisTotal:     d("redis_pool_total_connections", "Total connections in the Redis pool."),
		redisIdle:      d("redis_pool_idle_connections", "Idle connections in the Redis pool."),
		redisStale:     d("redis_pool_stale_connections", "Stale connections removed from the Redis pool."),
		redisHits:      d("redis_pool_hits_total", "Free connection hits in the Redis pool."),
		redisMisses:    d("redis_pool_misses_total", "Free connection misses in the Redis pool."),
		redisTimeouts:  d("redis_pool_timeouts_total", "Pool timeouts waiting for a Redis connection."),
		jobDepth:       d("job_queue_depth", "Total backlog (pending+running+failed) in the durable job queue."),
		jobByStatus:    d("job_queue_jobs", "Job-queue rows by status.", "status"),
		jobByType:      d("job_queue_depth_by_type", "Job-queue backlog by job type.", "job_type"),
		jobDeadLetters: d("job_queue_dead_letters", "Un-redriven dead-letter rows in the job queue."),
	}
}

func (c *resourceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.dbTotal
	ch <- c.dbAcquired
	ch <- c.dbIdle
	ch <- c.dbMax
	ch <- c.dbUtilization
	ch <- c.redisTotal
	ch <- c.redisIdle
	ch <- c.redisStale
	ch <- c.redisHits
	ch <- c.redisMisses
	ch <- c.redisTimeouts
	ch <- c.jobDepth
	ch <- c.jobByStatus
	ch <- c.jobByType
	ch <- c.jobDeadLetters
}

func (c *resourceCollector) Collect(ch chan<- prometheus.Metric) {
	g := func(desc *prometheus.Desc, v float64, labels ...string) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, labels...)
	}
	counter := func(desc *prometheus.Desc, v float64) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v)
	}

	if c.sources.DBPool != nil {
		s := c.sources.DBPool()
		g(c.dbTotal, float64(s.Total))
		g(c.dbAcquired, float64(s.Acquired))
		g(c.dbIdle, float64(s.Idle))
		g(c.dbMax, float64(s.Max))
		util := 0.0
		if s.Max > 0 {
			util = float64(s.Acquired) / float64(s.Max)
		}
		g(c.dbUtilization, util)
	}

	if c.sources.Redis != nil {
		s := c.sources.Redis()
		g(c.redisTotal, float64(s.Total))
		g(c.redisIdle, float64(s.Idle))
		g(c.redisStale, float64(s.Stale))
		counter(c.redisHits, float64(s.Hits))
		counter(c.redisMisses, float64(s.Misses))
		counter(c.redisTimeouts, float64(s.Timeouts))
	}

	if c.sources.JobQueue != nil {
		if s, ok := c.sources.JobQueue(); ok {
			g(c.jobDepth, float64(s.Depth))
			g(c.jobByStatus, float64(s.Pending), "pending")
			g(c.jobByStatus, float64(s.Running), "running")
			g(c.jobByStatus, float64(s.Failed), "failed")
			g(c.jobDeadLetters, float64(s.DeadLetters))
			for jt, n := range s.ByType {
				g(c.jobByType, float64(n), jt)
			}
		}
	}
}

// compile-time assertion.
var _ prometheus.Collector = (*resourceCollector)(nil)
