// Package metrics provides Prometheus metrics for Agent.
package metrics

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector collects and exposes Agent metrics.
type Collector struct {
	reg       *prometheus.Registry
	mu        sync.RWMutex
	startTime time.Time
	nodeID    string

	// Connection metrics
	connectionStatus    prometheus.Gauge
	connectionUptime    prometheus.Gauge
	connectionLatency   prometheus.Gauge
	reconnectTotal      prometheus.Counter
	reconnectFailTotal  prometheus.Counter

	// Traffic metrics
	trafficRX      prometheus.Counter
	trafficTX      prometheus.Counter
	trafficRXRate  prometheus.Gauge
	trafficTXRate  prometheus.Gauge
	packetRX       prometheus.Counter
	packetTX       prometheus.Counter

	// WireGuard metrics
	wgHandshake     prometheus.Gauge
	wgTxBytes       prometheus.Counter
	wgRxBytes       prometheus.Counter
	wgLatestHandshake prometheus.Gauge

	// NAT metrics
	natType        prometheus.Gauge
	publicIP       prometheus.GaugeVec
	stunQueryTotal prometheus.Counter
	stunQueryFail  prometheus.Counter

	// System metrics
	cpuUsage      prometheus.Gauge
	memoryUsage   prometheus.Gauge
	diskUsage     prometheus.Gauge
}

// NewCollector creates a new Agent metrics collector.
func NewCollector(nodeID string) *Collector {
	reg := prometheus.NewRegistry()

	c := &Collector{
		reg:       reg,
		startTime: time.Now(),
		nodeID:    nodeID,
	}

	// Connection metrics
	c.connectionStatus = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel_agent",
		Subsystem: "connection",
		Name:      "status",
		Help:      "Connection status (1=connected, 0=disconnected)",
	})

	c.connectionUptime = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel_agent",
		Subsystem: "connection",
		Name:      "uptime_seconds",
		Help:      "Connection uptime in seconds",
	})

	c.connectionLatency = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel_agent",
		Subsystem: "connection",
		Name:      "latency_ms",
		Help:      "Connection latency to coordinator in milliseconds",
	})

	c.reconnectTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel_agent",
		Subsystem: "connection",
		Name:      "reconnects_total",
		Help:      "Total reconnection attempts",
	})

	c.reconnectFailTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel_agent",
		Subsystem: "connection",
		Name:      "reconnects_failed_total",
		Help:      "Total failed reconnection attempts",
	})

	// Traffic metrics
	c.trafficRX = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel_agent",
		Subsystem: "traffic",
		Name:      "received_bytes_total",
		Help:      "Total bytes received",
	})

	c.trafficTX = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel_agent",
		Subsystem: "traffic",
		Name:      "sent_bytes_total",
		Help:      "Total bytes sent",
	})

	c.trafficRXRate = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel_agent",
		Subsystem: "traffic",
		Name:      "received_rate_bytes",
		Help:      "Current receive rate in bytes/sec",
	})

	c.trafficTXRate = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel_agent",
		Subsystem: "traffic",
		Name:      "sent_rate_bytes",
		Help:      "Current send rate in bytes/sec",
	})

	c.packetRX = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel_agent",
		Subsystem: "traffic",
		Name:      "packets_received_total",
		Help:      "Total packets received",
	})

	c.packetTX = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel_agent",
		Subsystem: "traffic",
		Name:      "packets_sent_total",
		Help:      "Total packets sent",
	})

	// WireGuard metrics
	c.wgHandshake = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel_agent",
		Subsystem: "wireguard",
		Name:      "handshake_status",
		Help:      "WireGuard handshake status (1=success, 0=pending)",
	})

	c.wgTxBytes = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel_agent",
		Subsystem: "wireguard",
		Name:      "transmit_bytes_total",
		Help:      "Total WireGuard transmit bytes",
	})

	c.wgRxBytes = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel_agent",
		Subsystem: "wireguard",
		Name:      "receive_bytes_total",
		Help:      "Total WireGuard receive bytes",
	})

	c.wgLatestHandshake = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel_agent",
		Subsystem: "wireguard",
		Name:      "latest_handshake_seconds",
		Help:      "Time since last handshake in seconds",
	})

	// NAT metrics
	c.natType = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel_agent",
		Subsystem: "nat",
		Name:      "type",
		Help:      "NAT type (0=unknown, 1=full_cone, 2=restricted, 3=port_restricted, 4=symmetric)",
	})

	c.publicIP = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "mycel_agent",
		Subsystem: "nat",
		Name:      "public_ip",
		Help:      "Public IP address (as label)",
	}, []string{"ip"})

	c.stunQueryTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel_agent",
		Subsystem: "nat",
		Name:      "stun_queries_total",
		Help:      "Total STUN queries",
	})

	c.stunQueryFail = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mycel_agent",
		Subsystem: "nat",
		Name:      "stun_queries_failed_total",
		Help:      "Total failed STUN queries",
	})

	// System metrics
	c.cpuUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel_agent",
		Subsystem: "system",
		Name:      "cpu_usage_percent",
		Help:      "CPU usage percentage",
	})

	c.memoryUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel_agent",
		Subsystem: "system",
		Name:      "memory_usage_bytes",
		Help:      "Memory usage in bytes",
	})

	c.diskUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mycel_agent",
		Subsystem: "system",
		Name:      "disk_usage_bytes",
		Help:      "Disk usage in bytes",
	})

	// Register all metrics
	reg.MustRegister(
		c.connectionStatus,
		c.connectionUptime,
		c.connectionLatency,
		c.reconnectTotal,
		c.reconnectFailTotal,
		c.trafficRX,
		c.trafficTX,
		c.trafficRXRate,
		c.trafficTXRate,
		c.packetRX,
		c.packetTX,
		c.wgHandshake,
		c.wgTxBytes,
		c.wgRxBytes,
		c.wgLatestHandshake,
		c.natType,
		&c.publicIP,
		c.stunQueryTotal,
		c.stunQueryFail,
		c.cpuUsage,
		c.memoryUsage,
		c.diskUsage,
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

// UpdateConnectionStatus updates connection status.
func (c *Collector) UpdateConnectionStatus(connected bool, uptime time.Duration, latencyMs float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if connected {
		c.connectionStatus.Set(1)
	} else {
		c.connectionStatus.Set(0)
	}
	c.connectionUptime.Set(uptime.Seconds())
	c.connectionLatency.Set(latencyMs)
}

// RecordReconnect records a reconnection attempt.
func (c *Collector) RecordReconnect(success bool) {
	c.reconnectTotal.Inc()
	if !success {
		c.reconnectFailTotal.Inc()
	}
}

// AddTrafficRX adds received bytes.
func (c *Collector) AddTrafficRX(bytes uint64, packets uint64) {
	c.trafficRX.Add(float64(bytes))
	c.packetRX.Add(float64(packets))
}

// AddTrafficTX adds sent bytes.
func (c *Collector) AddTrafficTX(bytes uint64, packets uint64) {
	c.trafficTX.Add(float64(bytes))
	c.packetTX.Add(float64(packets))
}

// UpdateTrafficRates updates current traffic rates.
func (c *Collector) UpdateTrafficRates(rxRate, txRate float64) {
	c.trafficRXRate.Set(rxRate)
	c.trafficTXRate.Set(txRate)
}

// UpdateWireGuardMetrics updates WireGuard metrics.
func (c *Collector) UpdateWireGuardMetrics(handshakeSuccess bool, txBytes, rxBytes uint64, handshakeAge time.Duration) {
	if handshakeSuccess {
		c.wgHandshake.Set(1)
	} else {
		c.wgHandshake.Set(0)
	}
	c.wgTxBytes.Add(float64(txBytes))
	c.wgRxBytes.Add(float64(rxBytes))
	c.wgLatestHandshake.Set(handshakeAge.Seconds())
}

// UpdateNATType updates NAT type.
func (c *Collector) UpdateNATType(natTypeValue int) {
	c.natType.Set(float64(natTypeValue))
}

// UpdatePublicIP updates public IP.
func (c *Collector) UpdatePublicIP(ip string) {
	c.publicIP.Reset()
	c.publicIP.WithLabelValues(ip).Set(1)
}

// RecordSTUNQuery records a STUN query.
func (c *Collector) RecordSTUNQuery(success bool) {
	c.stunQueryTotal.Inc()
	if !success {
		c.stunQueryFail.Inc()
	}
}

// UpdateSystemMetrics updates system metrics.
func (c *Collector) UpdateSystemMetrics(cpuPercent, memoryBytes, diskBytes float64) {
	c.cpuUsage.Set(cpuPercent)
	c.memoryUsage.Set(memoryBytes)
	c.diskUsage.Set(diskBytes)
}

// GetRegistry returns the Prometheus registry.
func (c *Collector) GetRegistry() *prometheus.Registry {
	return c.reg
}

// GetUptime returns the agent uptime.
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
	Connected       bool
	ConnectionUptime time.Duration
	ConnectionLatency float64
	TrafficRX       uint64
	TrafficTX       uint64
	PacketRX        uint64
	PacketTX        uint64
	WGHandshake     bool
	WGTxBytes       uint64
	WGRxBytes       uint64
	CPUUsage        float64
	MemoryUsage     float64
}

func (c *Collector) applySnapshot(s *MetricsSnapshot) {
	c.UpdateConnectionStatus(s.Connected, s.ConnectionUptime, s.ConnectionLatency)
	c.UpdateTrafficRates(0, 0) // Rates calculated separately
	c.UpdateWireGuardMetrics(s.WGHandshake, s.WGTxBytes, s.WGRxBytes, 0)
	c.UpdateSystemMetrics(s.CPUUsage, s.MemoryUsage, 0)
}
