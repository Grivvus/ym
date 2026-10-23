package metrics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Grivvus/ym/internal/repository"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v4/disk"
)

var ErrCantRegisterCollector = errors.New("can't register collector")

type metrics struct {
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	DiskUsage           prometheus.Gauge

	TotalUsersRegistered        prometheus.Gauge
	TracksUploaded              prometheus.Gauge
	CurrentTranscodingQueueSize prometheus.Gauge
	DailyActiveUsers            prometheus.Counter
}

type MetricsManager struct {
	repo   repository.MetricsRepository
	logger *slog.Logger
	values metrics
}

func New(logger *slog.Logger, repo repository.MetricsRepository) MetricsManager {
	return MetricsManager{
		repo:   repo,
		logger: logger,
		values: metrics{
			HTTPRequestsTotal: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "ym",
					Subsystem: "http",
					Name:      "requests_total",
				},
				[]string{"method", "route", "status"},
			),
			HTTPRequestDuration: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "ym",
					Subsystem: "http",
					Name:      "request_duration_s",
					Buckets:   prometheus.DefBuckets,
				},
				[]string{"method", "route", "status"},
			),
			DiskUsage: prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: "ym",
				Name:      "disk_usage_percentage",
			}),
			TotalUsersRegistered: prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: "ym",
				Name:      "total_users_registered",
			}),
			TracksUploaded: prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: "ym",
				Name:      "total_tracks_uploaded",
			}),
			CurrentTranscodingQueueSize: prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: "ym",
				Name:      "current_transcoding_queue_size",
			}),
			DailyActiveUsers: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: "ym",
				Name:      "daily_active_users",
			}),
		},
	}
}

func (m *MetricsManager) Register(registry *prometheus.Registry) error {
	errs := make([]error, 0, 10)
	errs = append(errs, registry.Register(m.values.HTTPRequestDuration))
	errs = append(errs, registry.Register(m.values.HTTPRequestsTotal))
	errs = append(errs, registry.Register(m.values.TotalUsersRegistered))
	errs = append(errs, registry.Register(m.values.TracksUploaded))
	errs = append(errs, registry.Register(m.values.CurrentTranscodingQueueSize))
	errs = append(errs, registry.Register(m.values.DiskUsage))
	var err error
	for _, errn := range errs {
		if errn != nil {
			errors.Join(err, errn)
		}
	}
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCantRegisterCollector, err)
	}
	return nil
}

func (m *MetricsManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		sr := &statusRecorder{
			w, http.StatusOK,
		}

		next.ServeHTTP(sr, r)

		route := r.Pattern
		if route == "" {
			route = "unknown"
		}

		status := strconv.Itoa(sr.statusCode)

		duration := time.Since(start).Seconds()

		m.values.HTTPRequestDuration.WithLabelValues(r.Method, route, status).Observe(duration)
		m.values.HTTPRequestsTotal.WithLabelValues(r.Method, route, status).Inc()
	})
}

// Will block goroutine until ctx is Done
func (m *MetricsManager) Job(ctx context.Context, period time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			func() {
				nctx, cancel := context.WithTimeout(ctx, period)
				defer cancel()
				var wg sync.WaitGroup
				wg.Add(3)
				go m.collectRegisteredUsersCnt(nctx, &wg)
				go m.collectTranscodingQueueSize(nctx, &wg)
				go m.collectUploadedTracksCnt(nctx, &wg)
				wg.Wait()
			}()
		}
		time.Sleep(period)
	}
}

func (m *MetricsManager) DiskUsageJob(ctx context.Context, period time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			du, err := disk.Usage("/")
			if err != nil {
				m.logger.Warn("can't collect disk usage statistic", "err", err)
				break
			}
			m.values.DiskUsage.Set(du.UsedPercent)

		}
		time.Sleep(period)
	}
}

// we could try to make one *generic* function
// that can collect all metrics
func (m *MetricsManager) collectTranscodingQueueSize(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	size, err := m.repo.CurrentTranscodingQueueSize(ctx)
	if err != nil {
		m.logger.Warn(
			"can't collect metric",
			"metric_name", "CurrentTranscodingQueueSize",
			"err", err,
		)
		return
	}
	m.values.CurrentTranscodingQueueSize.Set(float64(size))
}

func (m *MetricsManager) collectUploadedTracksCnt(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	tracksCnt, err := m.repo.TotalUniqueTracks(ctx)
	if err != nil {
		m.logger.Warn(
			"can't collect metric",
			"metric_name", "TracksUploaded",
			"err", err,
		)
		return
	}
	m.values.TracksUploaded.Set(float64(tracksCnt))
}

func (m *MetricsManager) collectRegisteredUsersCnt(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	usersCnt, err := m.repo.TotalUsersRegistered(ctx)
	if err != nil {
		m.logger.Warn(
			"can't collect metric",
			"metric_name", "UsersRegistered",
			"err", err,
		)
		return
	}

	m.values.TotalUsersRegistered.Set(float64(usersCnt))
}

type statusRecorder struct {
	// embed standard response writer
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}
