package metrics

import "fmt"

var MetricsCollector Collector

// Collector 指标收集器接口
type Collector interface {
	Counter(name string, value float64, labels map[string]string)
	Gauge(name string, value float64, labels map[string]string)
	Histogram(name string, value float64, labels map[string]string)
	RecordExecutionTime(componentName string, duration float64, labels map[string]string)
}

type defaultCollector struct{}

func NewCollector() {
	MetricsCollector = &defaultCollector{}
	return
}

func (c *defaultCollector) Counter(name string, value float64, labels map[string]string)   {}
func (c *defaultCollector) Gauge(name string, value float64, labels map[string]string)     {}
func (c *defaultCollector) Histogram(name string, value float64, labels map[string]string) {}

func (c *defaultCollector) RecordExecutionTime(componentName string, duration float64, labels map[string]string) {
	// 这里可以实现具体的记录逻辑，比如将数据发送到监控系统
	fmt.Println("RecordExecutionTime", componentName, duration*1000, labels)
}
