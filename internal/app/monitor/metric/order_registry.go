package mmetric

import (
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "order"
	subsystem = "order"
)

func registerCollector(
	registerer prometheus.Registerer,
	collector prometheus.Collector,
) (prometheus.Collector, error) {
	err := registerer.Register(collector)
	var alreadyRegisteredErr *prometheus.AlreadyRegisteredError
	if err != nil {
		if errors.As(err, &alreadyRegisteredErr) {
			return alreadyRegisteredErr.ExistingCollector, nil
		}

		return nil, err
	}

	return nil, nil
}

func counter(reg prometheus.Registerer, name, help string) (prometheus.Counter, error) {
	fullName := prometheus.BuildFQName(namespace, subsystem, name)

	c := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
	)
	existing, err := registerCollector(reg, c)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		counter, ok := existing.(prometheus.Counter)
		if !ok {
			return nil, fmt.Errorf("metric %q already registered with incompatible collector type %T", fullName, existing)
		}

		return counter, nil
	}

	return c, nil
}

func counterVec(reg prometheus.Registerer, name, help string, labels ...string) (*prometheus.CounterVec, error) {
	fullName := prometheus.BuildFQName(namespace, subsystem, name)

	c := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)
	existing, err := registerCollector(reg, c)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		counterVec, ok := existing.(*prometheus.CounterVec)
		if !ok {
			return nil, fmt.Errorf("metric %q already registered with incompatible collector type %T", fullName, existing)
		}

		return counterVec, nil
	}

	return c, nil
}

func histogramVec(reg prometheus.Registerer, name, help string, buckets []float64, labels ...string) (*prometheus.HistogramVec, error) {
	fullName := prometheus.BuildFQName(namespace, subsystem, name)

	v := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		},
		labels,
	)
	existing, err := registerCollector(reg, v)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		histogramVec, ok := existing.(*prometheus.HistogramVec)
		if !ok {
			return nil, fmt.Errorf("metric %q already registered with incompatible collector type %T", fullName, existing)
		}

		return histogramVec, nil
	}

	return v, nil
}

func gaugeVec(reg prometheus.Registerer, name, help string, labels ...string) (*prometheus.GaugeVec, error) {
	fullName := prometheus.BuildFQName(namespace, subsystem, name)

	g := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)

	existing, err := registerCollector(reg, g)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		gaugeVec, ok := existing.(*prometheus.GaugeVec)
		if !ok {
			return nil, fmt.Errorf("metric %q already registered with incompatible collector type %T", fullName, existing)
		}

		return gaugeVec, nil
	}

	return g, nil
}
