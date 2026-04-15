// Package metrics provides Prometheus metrics for Coordinator.
package metrics

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector collects and exposes Prometheus metrics.
type Collector struct {
	reg       *prometheus.Registry
	mu        sync.RWMutex
	startTime time.Time

	// Node metrics
	nodeTotal      prometheus.Gauge
	nodeOnline     prometheus.Gauge
	nodeOffline    prometheus.Gauge
	nodeLatency    prometheus.GaugeVec
	nodeUptime     prometheus.GaugeVec

	// Traffic metrics
	trafficRX      prometheus.Counter
	trafficTX      prometheus.Counter
	trafficRXRate  prometheus.Gauge
	trafficTXRate  prometheus.Gauge

	// Connection pool metrics
	poolTotal      prometheus.Gauge
	poolInUse      prometheus.Gauge
	poolIdle       prometheus.Gauge
	poolAcquires   prometheus.Counter
	poolErrors     prometheus.Counter

	// NAT traversal metrics
	natPunchTotal   prometheus.Counter
	natPunchSuccess prometheus.Counter
	natPunchFail    prometheus.Counter
	natType         prometheus.GaugeVec

	// ACL metrics
	aclRulesTotal   prometheus.Gauge
	aclCheckTotal   prometheus.Counter
	aclDenyTotal    prometheus.Counter
}

// NewCollector creates a new metrics collector.
func NewCollector() *Collector {
	reg := prometheus.NewRegistry()

	c := &Collector{
		reg:       reg,
		startTime: time.Now(),
	}

	// Initialize node metrics
	c.nodeTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel",
		Subsystem: "node",
		Name:      "total",
		Help:      "Total number of registered nodes",
	})

	c.nodeOnline = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel",
		Subsystem: "node",
		Name:      "online",
		Help:      "Number of online nodes",
	})

	c.nodeOffline = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel",
		Subsystem: "node",
		Name:      "offline",
		Help:      "Number of offline nodes",
	})

	c.nodeLatency = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "mycel",
		Subsystem: "node",
		Name:      "latency_ms",
		Help:      "Node latency in milliseconds",
	}, []string{"node_id", "network_id"})

	c.nodeUptime = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "mycel",
		Subsystem: "node",
		Name:      "uptime_seconds",
		Help:      "Node uptime in seconds",
	}, []string{"node_id", "network_id"})

	// Initialize traffic metrics
	c.trafficRX = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel",
		Subsystem: "traffic",
		Name:      "received_bytes_total",
		Help:      "Total bytes received",
	})

	c.trafficTX = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel",
		Subsystem: "traffic",
		Name:      "sent_bytes_total",
		Help:      "Total bytes sent",
	})

	c.trafficRXRate = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel",
		Subsystem: "traffic",
		Name:      "received_rate_bytes",
		Help:      "Current receive rate in bytes/sec",
	})

	c.trafficTXRate = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel",
		Subsystem: "traffic",
		Name:      "sent_rate_bytes",
		Help:      "Current send rate in bytes/sec",
	})

	// Initialize pool metrics
	c.poolTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel",
		Subsystem: "pool",
		Name:      "connections_total",
		Help:      "Total connections in pool",
	})

	c.poolInUse = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel",
		Subsystem: "pool",
		Name:      "connections_in_use",
		Help:      "Connections currently in use",
	})

	c.poolIdle = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel",
		Subsystem: "pool",
		Name:      "connections_idle",
		Help:      "Idle connections in pool",
	})

	c.poolAcquires = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel",
		Subsystem: "pool",
		Name:      "acquires_total",
		Help:      "Total connection acquires",
	})

	c.poolErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel",
		Subsystem: "pool",
		Name:      "errors_total",
		Help:      "Total pool errors",
	})

	// Initialize NAT metrics
	c.natPunchTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel",
		Subsystem: "nat",
		Name:      "punch_total",
		Help:      "Total NAT punch attempts",
	})

	c.natPunchSuccess = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel",
		Subsystem: "nat",
		Name:      "punch_success_total",
		Help:      "Successful NAT punch attempts",
	})

	c.natPunchFail = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel",
		Subsystem: "nat",
		Name:      "punch_fail_total",
		Help:      "Failed NAT punch attempts",
	})

	c.natType = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "mycel",
		Subsystem: "nat",
		Name:      "type",
		Help:      "NAT type (0=unknown, 1=full_cone, 2=restricted, 3=port_restricted, 4=symmetric)",
	}, []string{"node_id"})

	// Initialize ACL metrics
	c.aclRulesTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel",
		Subsystem: "acl",
		Name:      "rules_total",
		Help:      "Total ACL rules",
	})

	c.aclCheckTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel",
		Subsystem: "acl",
		Name:      "checks_total",
		Help:      "Total ACL checks",
	})

	c.aclDenyTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel",
		Subsystem: "acl",
		Name:      "denies_total",
		Help:      "Total ACL denies",
	})

	// Register all metrics
	reg.MustRegister(
		c.nodeTotal,
		c.nodeOnline,
		c.nodeOffline,
		&c.nodeLatency,
		&c.nodeUptime,
		c.trafficRX,
		c.trafficTX,
		c.trafficRXRate,
		c.trafficTXRate,
		c.poolTotal,
		c.poolInUse,
		c.poolIdle,
		c.poolAcquires,
		c.poolErrors,
		c.natPunchTotal,
		c.natPunchSuccess,
		c.natPunchFail,
		&c.natType,
		c.aclRulesTotal,
		c.aclCheckTotal,
		c.aclDenyTotal,
	)

	return c
}

// Handler returns the HTTP handler for metrics endpoint.
func (c *Collector) Handler() http.Handler {
	return promhttp.HandlerFor(c.reg, promhttp.HandlerOpts{
		Registry:          c.reg,
		EnableOpenMetrics: true,
	})
}

// ServeHTTP implements http.Handler for /metrics endpoint.
func (c *Collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.Handler().ServeHTTP(w, r)
}

// UpdateNodeMetrics updates node-related metrics.
func (c *Collector) UpdateNodeMetrics(total, online, offline int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.nodeTotal.Set(float64(total))
	c.nodeOnline.Set(float64(online))
	c.nodeOffline.Set(float64(offline))
}

// UpdateNodeLatency updates latency for a specific node.
func (c *Collector) UpdateNodeLatency(nodeID, networkID string, latencyMs float64) {
	c.nodeLatency.WithLabelValues(nodeID, networkID).Set(latencyMs)
}

// UpdateNodeUptime updates uptime for a specific node.
func (c *Collector) UpdateNodeUptime(nodeID, networkID string, uptime time.Duration) {
	c.nodeUptime.WithLabelValues(nodeID, networkID).Set(uptime.Seconds())
}

// AddTrafficRX adds received bytes.
func (c *Collector) AddTrafficRX(bytes uint64) {
	c.trafficRX.Add(float64(bytes))
}

// AddTrafficTX adds sent bytes.
func (c *Collector) AddTrafficTX(bytes uint64) {
	c.trafficTX.Add(float64(bytes))
}

// UpdateTrafficRates updates current traffic rates.
func (c *Collector) UpdateTrafficRates(rxRate, txRate float64) {
	c.trafficRXRate.Set(rxRate)
	c.trafficTXRate.Set(txRate)
}

// UpdatePoolMetrics updates connection pool metrics.
func (c *Collector) UpdatePoolMetrics(total, inUse, idle int, acquires, errors uint64) {
	c.poolTotal.Set(float64(total))
	c.poolInUse.Set(float64(inUse))
	c.poolIdle.Set(float64(idle))
	c.poolAcquires.Add(float64(acquires))
	c.poolErrors.Add(float64(errors))
}

// RecordNATPunch records a NAT punch attempt.
func (c *Collector) RecordNATPunch(success bool) {
	c.natPunchTotal.Inc()
	if success {
		c.natPunchSuccess.Inc()
	} else {
		c.natPunchFail.Inc()
	}
}

// UpdateNATType updates NAT type for a node.
func (c *Collector) UpdateNATType(nodeID string, natTypeValue int) {
	c.natType.WithLabelValues(nodeID).Set(float64(natTypeValue))
}

// UpdateACLMetrics updates ACL metrics.
func (c *Collector) UpdateACLMetrics(rulesTotal, checks, denies uint64) {
	c.aclRulesTotal.Set(float64(rulesTotal))
	c.aclCheckTotal.Add(float64(checks))
	c.aclDenyTotal.Add(float64(denies))
}

// UpdateSubnetMetrics updates subnet metrics (placeholder for future implementation).
func (c *Collector) UpdateSubnetMetrics(subnetID string, allocated, available int) {
	// Future: add subnet-specific metrics
}

// GetRegistry returns the Prometheus registry.
func (c *Collector) GetRegistry() *prometheus.Registry {
	return c.reg
}

// GetUptime returns the coordinator uptime.
func (c *Collector) GetUptime() time.Duration {
	return time.Since(c.startTime)
}

// StartBackgroundCollection starts background metrics collection.
func (c *Collector) StartBackgroundCollection(ctx context.Context, interval time.Duration, collectorFunc func() (*MetricsSnapshot, error)) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				snapshot, err := collectorFunc()
				if err != nil {
					continue
				}
				c.applySnapshot(snapshot)
			}
		}
	}()
}

// MetricsSnapshot represents a point-in-time metrics snapshot.
type MetricsSnapshot struct {
	NodeTotal   int
	NodeOnline  int
	NodeOffline int
	TrafficRX   uint64
	TrafficTX   uint64
	PoolTotal   int
	PoolInUse   int
	PoolIdle    int
	PoolAcquires uint64
	PoolErrors  uint64
	ACLRules    uint64
	ACLChecks   uint64
	ACLDenies   uint64
}

func (c *Collector) applySnapshot(s *MetricsSnapshot) {
	c.UpdateNodeMetrics(s.NodeTotal, s.NodeOnline, s.NodeOffline)
	c.UpdatePoolMetrics(s.PoolTotal, s.PoolInUse, s.PoolIdle, s.PoolAcquires, s.PoolErrors)
	c.UpdateACLMetrics(s.ACLRules, s.ACLChecks, s.ACLDenies)
}
