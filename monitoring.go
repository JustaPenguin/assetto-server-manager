package servermanager

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/getsentry/raven-go"
	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

const (
	sentryDSN   = "https://d8325490c2fa49b58cd9b557aed9ce8a@sentry.justapengu.in/4"
	sentryJSDSN = "https://9153aa91818949c7a70708d0f0566faf@sentry.justapengu.in/5"
)

var (
	panicHandler = middleware.Recoverer

	defaultPanicCapture = func(fn func()) {
		defer func() {
			if r := recover(); r != nil {
				_, _ = fmt.Fprintf(logMultiWriter, "\n\nrecovered from panic: %v\n\n", r)
				_, _ = fmt.Fprint(logMultiWriter, string(debug.Stack()))
			}
		}()

		fn()
	}

	panicCapture = defaultPanicCapture

	prometheusMonitoringHandler = http.NotFoundHandler

	prometheusMonitoringWrapper = func(next http.Handler) http.Handler {
		return next
	}
)

func InitMonitoring() {
	logrus.Infof("initialising Raven monitoring")
	err := raven.SetDSN(sentryDSN)

	if err != nil {
		logrus.WithError(err).Error("could not initialise raven monitoring")
	}

	raven.SetRelease(BuildVersion)

	panicHandler = raven.Recoverer
	panicCapture = func(fn func()) {
		raven.CapturePanic(fn, nil)
	}

	http.DefaultTransport = RoundTripper(http.DefaultTransport)
	http.DefaultClient.Transport = http.DefaultTransport

	logrus.Infof("initialising Prometheus Monitoring")
	prometheus.MustRegister(HTTPInFlightGauge, HTTPCounter, HTTPDuration, HTTPResponseSize, httpInFlightRequests, httpRequestCounter, dnsLatencyVec, tlsLatencyVec, histVec)
	prometheusMonitoringHandler = promhttp.Handler
	prometheusMonitoringWrapper = func(next http.Handler) http.Handler {
		return promhttp.InstrumentHandlerInFlight(HTTPInFlightGauge,
			promhttp.InstrumentHandlerDuration(HTTPDuration.MustCurryWith(prometheus.Labels{"handler": "push"}),
				promhttp.InstrumentHandlerCounter(HTTPCounter,
					promhttp.InstrumentHandlerResponseSize(HTTPResponseSize, next),
				),
			),
		)
	}
}

var httpInFlightRequests = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "client_in_flight_requests",
	Help: "A gauge of in-flight requests for the wrapped crawler client.",
})

var httpRequestCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "client_api_requests_total",
		Help: "A counter for requests from the wrapped crawler client.",
	},
	[]string{"code", "method"},
)

// dnsLatencyVec uses custom buckets based on expected dns durations.
// It has an instance label "event", which is set in the
// DNSStart and DNSDonehook functions defined in the
// InstrumentTrace struct below.
var dnsLatencyVec = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "dns_duration_seconds",
		Help:    "Trace dns latency histogram.",
		Buckets: []float64{.005, .01, .025, .05},
	},
	[]string{"event"},
)

// tlsLatencyVec uses custom buckets based on expected tls durations.
// It has an instance label "event", which is set in the
// TLSHandshakeStart and TLSHandshakeDone hook functions defined in the
// InstrumentTrace struct below.
var tlsLatencyVec = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "tls_duration_seconds",
		Help:    "Trace tls latency histogram.",
		Buckets: []float64{.05, .1, .25, .5},
	},
	[]string{"event"},
)

// histVec has no labels, making it a zero-dimensional ObserverVec.
var histVec = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "client_request_duration_seconds",
		Help:    "A histogram of request latencies.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{},
)

var trace = &promhttp.InstrumentTrace{
	DNSStart: func(t float64) {
		dnsLatencyVec.WithLabelValues("dns_start")
	},
	DNSDone: func(t float64) {
		dnsLatencyVec.WithLabelValues("dns_done")
	},
	TLSHandshakeStart: func(t float64) {
		tlsLatencyVec.WithLabelValues("tls_handshake_start")
	},
	TLSHandshakeDone: func(t float64) {
		tlsLatencyVec.WithLabelValues("tls_handshake_done")
	},
}

func RoundTripper(t http.RoundTripper) http.RoundTripper {
	return promhttp.InstrumentRoundTripperInFlight(httpInFlightRequests,
		promhttp.InstrumentRoundTripperCounter(httpRequestCounter,
			promhttp.InstrumentRoundTripperTrace(trace,
				promhttp.InstrumentRoundTripperDuration(histVec, t),
			),
		),
	)
}

var HTTPInFlightGauge = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "in_flight_requests",
	Help: "A gauge of requests currently being served by the wrapped handler.",
})

var HTTPCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "web_requests_total",
		Help: "A counter for requests to the wrapped handler.",
	},
	[]string{"code", "method"},
)

// HTTPDuration is partitioned by the HTTP method and handler. It uses custom
// buckets based on the expected request duration.
var HTTPDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "request_duration_seconds",
		Help:    "A histogram of latencies for requests.",
		Buckets: []float64{.25, .5, 1, 2.5, 5, 10},
	},
	[]string{"handler", "method"},
)

// HTTPResponseSize has no labels, making it a zero-dimensional
// ObserverVec.
var HTTPResponseSize = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "response_size_bytes",
		Help:    "A histogram of response sizes for requests.",
		Buckets: []float64{200, 500, 900, 1500},
	},
	[]string{},
)
