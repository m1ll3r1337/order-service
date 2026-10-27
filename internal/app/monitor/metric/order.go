package mmetric

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/m1ll3r1337/order-service/internal/app/monitor"
)

const (
	labelResult = "result"
	labelStatus = "status"
)

type createResult string

const (
	createResultSuccess createResult = "success"
	createResultError   createResult = "error"
)

type Order struct {
	createTotal           *prometheus.CounterVec
	createDurationSeconds *prometheus.HistogramVec
	createdAmountTotal    prometheus.Counter
	inStatus              *prometheus.GaugeVec
	eventPublishFailed    prometheus.Counter
}

var _ monitor.OrderMetrics = (*Order)(nil)

func NewPrometheusOrder(registerer prometheus.Registerer) (*Order, error) {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	createTotalName := "create_total"
	createTotalCounterVec, err := counterVec(registerer, createTotalName, "", labelResult)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", createTotalName, err)
	}

	createDurationSecondsName := "create_duration_seconds"
	createDurationSecondsHistogramVec, err := histogramVec(registerer, createDurationSecondsName, "", prometheus.DefBuckets, labelResult)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", createDurationSecondsName, err)
	}

	createdAmountTotalName := "created_amount_cents_total"
	createdAmountTotalCounter, err := counter(registerer, createdAmountTotalName, "")
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", createdAmountTotalName, err)
	}

	inStatusName := "in_status"
	inStatusGauge, err := gaugeVec(registerer, inStatusName, "", labelStatus)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", inStatusName, err)
	}

	metricEventPublishFailedName := "event_publish_failed_total"
	metricEventPublishFailedHelp := "Total number of failed Kafka event publish attempts."
	eventPublishFailedCounter, err := counter(registerer, metricEventPublishFailedName, metricEventPublishFailedHelp)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", metricEventPublishFailedName, err)
	}

	return &Order{
		createTotal:           createTotalCounterVec,
		createDurationSeconds: createDurationSecondsHistogramVec,
		createdAmountTotal:    createdAmountTotalCounter,
		inStatus:              inStatusGauge,
		eventPublishFailed:    eventPublishFailedCounter,
	}, nil
}

type orderCreateMetric struct {
	parent  *Order
	started time.Time
}

func (m *Order) Create() monitor.OrderCreateMetric {
	return &orderCreateMetric{
		parent:  m,
		started: time.Now(),
	}
}

func (m *Order) Status() monitor.OrderStatusMetric {
	return &orderStatusMetric{parent: m}
}

func (m *orderCreateMetric) Success(totalPrice int64) {
	duration := time.Since(m.started).Seconds()
	m.parent.createTotal.WithLabelValues(string(createResultSuccess)).Inc()
	m.parent.createDurationSeconds.WithLabelValues(string(createResultSuccess)).Observe(duration)
	if totalPrice > 0 {
		m.parent.createdAmountTotal.Add(float64(totalPrice))
	}
}

func (m *orderCreateMetric) Failed(err error) {
	duration := time.Since(m.started).Seconds()
	m.parent.createTotal.WithLabelValues(string(createResultError)).Inc()
	m.parent.createDurationSeconds.WithLabelValues(string(createResultError)).Observe(duration)
}

func (m *orderCreateMetric) PublishFailed() {
	m.parent.eventPublishFailed.Inc()
}

type orderStatusMetric struct {
	parent *Order
}

func (m *orderStatusMetric) Inc(status string) {
	m.parent.inStatus.WithLabelValues(string(normalizeStatus(status))).Inc()
}

func (m *orderStatusMetric) Dec(status string) {
	m.parent.inStatus.WithLabelValues(string(normalizeStatus(status))).Dec()
}

func (m *orderStatusMetric) Transition(oldStatus, newStatus string) {
	oldS := normalizeStatus(oldStatus)
	newS := normalizeStatus(newStatus)
	if oldS == newS {
		return
	}

	m.parent.inStatus.WithLabelValues(string(oldS)).Dec()
	m.parent.inStatus.WithLabelValues(string(newS)).Inc()
}

func (m *orderStatusMetric) Set(status string, count float64) {
	m.parent.inStatus.WithLabelValues(string(normalizeStatus(status))).Set(count)
}
