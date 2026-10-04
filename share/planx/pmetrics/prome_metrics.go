package pmetrics

import (
	"time"

	"github.com/nghichtu91/platform/share/planx"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

const (
	Counter = iota
	Gauge
	Histogram
	Summary
)

const CannotGetValue = -1

var (
	serverStateRegistry *prometheus.Registry // 用以注册服务器Collector（指标收集器，每个收集器对应一个metrics）。
)

func init() {
	serverStateRegistry = prometheus.NewRegistry()
}

// NewCounter 创建一个单调递增注册到serverStateRegistry中的Counter指标。
//
// Param-name:指标名称。
//
// Return-prometheus.Counter:被创建的指标。Return-error:创建时的报错信息。
func NewCounter(name string) (prometheus.Counter, error) {
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: name,
	})

	if err := serverStateRegistry.Register(counter); err != nil {
		return nil, err
	}

	return counter, nil
}

// NewGauge 创建一个用于计数注册到serverStateRegistry中的Gauge指标。
//
// Param-name:指标名称。
//
// Return-prometheus.Gauge:被创建的指标。Return-error:创建时的报错信息。
func NewGauge(name string) (prometheus.Gauge, error) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: name,
	})

	if err := serverStateRegistry.Register(gauge); err != nil {
		return nil, err
	}

	return gauge, nil
}

// NewHistogram 创建一个注册到serverStateRegistry中的直方图指标。
//
// Param-name:指标名称。
//
// Return-prometheus.Histogram:被创建的指标。Return-error:创建时的报错信息。
func NewHistogram(name string) (prometheus.Histogram, error) {
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: name,
	})

	if err := serverStateRegistry.Register(histogram); err != nil {
		return nil, err
	}

	return histogram, nil
}

//// Deprecated
//func NewPushAloneGuage(name string, value int64) prometheus.Gauge {
//	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
//		Name: name,
//	})
//	gauge.Set(float64(value))
//	return gauge
//}

// GetMetricValueByType 获取贮藏在指标内的数值。
//
// Param-collector:需要获取值的指标收集器。Param-collectorType：需要获取值的指标收集器的类型（Counter、Gauge、Histogram、Summary）。
//
// Note:Histogram、Summary have no implementation。
func GetMetricValueByType(collector prometheus.Collector, collectorType int) int64 {
	if collector == nil {
		return CannotGetValue
	}
	cMetric := &dto.Metric{}
	pMetricChan := make(chan prometheus.Metric, 1)
	defer close(pMetricChan)
	collector.Collect(pMetricChan)
	select {
	case pMetric := <-pMetricChan:
		if err := pMetric.Write(cMetric); err != nil {
			tilogs.L().Warnf("<Promethues>  Metrics  GetMetricValueByType ERR :%v", err)
		}
	case <-time.After(planx.InnerTimeOut):
		return CannotGetValue
	}

	switch collectorType {
	case Counter:
		if cMetric.Counter == nil {
			return CannotGetValue
		}
		return int64(*cMetric.Counter.Value)
	case Gauge:
		if cMetric.Gauge == nil {
			return CannotGetValue
		}
		return int64(*cMetric.Gauge.Value)
	case Histogram:
		// TODO Implementation.
		return CannotGetValue
	case Summary:
		// TODO Implementation.
		return CannotGetValue
	default:
		return CannotGetValue
	}
}

// GetCollectorName 获取指标的PushGateWay展示名字。
//
// Param-jobName:向PushGateWay推送的Job名称（一般由项目名+大区号+区服号（如果有的）组成）。Param-collectorName：指标名称。
//
// Note：为了方便Grafana查询，不再拼接jobName。
func GetCollectorName(jobName string, collectorName string) string {
	return collectorName
}
