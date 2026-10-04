package simplegraphite

import (
	"bytes"
	"fmt"
	// "log"
	"net"
	"strconv"
	"sync"
	"time"
)

// Graphite is a struct that defines the relevant properties of a graphite
// connection
type Graphite struct {
	Host     string
	Port     int
	Protocol string
	Timeout  time.Duration
	Prefix   string
	conn     net.Conn
	nop      bool
	// mux      sync.RWMutex
	ms chan Metric
}

var (
	buffPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, defaultMetricsBuffLen))
		},
	}
	// mPool = sync.Pool{New: func() interface{} {
	// 	return new(Metric)
	// }}
)

// defaultTimeout is the default number of seconds that we're willing to wait
// before forcing the connection establishment to fail
const defaultTimeout = 5

const defaultMetricsLen = 256

const defaultMetricsBuffLen = 128

var (
	space     = []byte(" ")
	breakLine = []byte("\n")
)

// IsNop is a getter for *graphite.Graphite.nop
func (graphite *Graphite) IsNop() bool {
	return graphite.nop
}

// Given a Graphite struct, Connect populates the Graphite.conn field with an
// appropriate TCP connection
func (graphite *Graphite) Connect() error {
	// graphite.mux.Lock()
	// defer graphite.mux.Unlock()
	if !graphite.IsNop() {
		if graphite.conn != nil {
			graphite.conn.Close()
		}

		address := fmt.Sprintf("%s:%d", graphite.Host, graphite.Port)

		if graphite.Timeout == 0 {
			graphite.Timeout = defaultTimeout * time.Second
		}

		conn, err := net.DialTimeout(graphite.Protocol, address, graphite.Timeout)
		if err != nil {
			return err
		}

		graphite.conn = conn

		go graphite.handle()
	}

	return nil
}

func (graphite *Graphite) handle() {
	for {
		select {
		case m, ok := <-graphite.ms:
			if !ok {
				return
			}
			graphite.sendMetricsV2(m)
			// mPool.Put(m)
		}
	}
}

// Given a Graphite struct, Disconnect closes the Graphite.conn field
func (graphite *Graphite) Disconnect() error {
	if graphite.IsNop() {
		return nil
	}
	close(graphite.ms)
	err := graphite.conn.Close()
	graphite.conn = nil
	return err
}

/*
// Given a Metric struct, the SendMetric method sends the supplied metric to the
// Graphite connection that the method is called upon
func (graphite *Graphite) SendMetric(m Metric) error {
	if graphite.IsNop() {
		return nil
	}

	select {
	case graphite.ms <- &m:
	default:
	}

	return nil
}

// Given a slice of Metrics, the SendMetrics method sends the metrics, as a
// batch, to the Graphite connection that the method is called upon
func (graphite *Graphite) SendMetrics(metrics ...Metric) error {
	if graphite.IsNop() {
		return nil
	}

	for _, m := range metrics {
		select {
		case graphite.ms <- &m:
		default:
		}
	}
	return nil
}

// sendMetrics is an internal function that is used to write to the TCP
// connection in order to communicate metrics to the remote Graphite host
func (graphite *Graphite) sendMetrics(metrics []Metric) error {
	zeroed_metric := Metric{} // ignore unintialized metrics
	if !graphite.IsNop() {
		buf := bytes.NewBufferString("")
		for _, metric := range metrics {
			if metric == zeroed_metric {
				continue // ignore unintialized metrics
			}
			if metric.Timestamp == 0 {
				metric.Timestamp = time.Now().Unix()
			}
			metric_name := ""
			if graphite.Prefix != "" {
				metric_name = fmt.Sprintf("%s.%s", graphite.Prefix, metric.Name)
			} else {
				metric_name = metric.Name
			}
			buf.WriteString(fmt.Sprintf("%s %s %d\n", metric_name, metric.Value, metric.Timestamp))
		}
		_, err := graphite.conn.Write(buf.Bytes())
		//fmt.Print("Sent msg:", buf.String(), "'")
		if err != nil {
			return err
		}
	} else {
		//for _, metric := range metrics {
		//	log.Printf("Graphite: %s\n", metric)
		//}
	}
	return nil
}
*/

func (graphite *Graphite) sendMetricsV2(metric Metric) {
	buf := buffPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer buffPool.Put(buf)

	if metric.Name == "" {
		return
	}

	// name
	if graphite.Prefix != "" {
		buf.WriteString(graphite.Prefix)
		buf.WriteString(".")
	}
	buf.WriteString(metric.Name)

	// value
	buf.Write(space)
	buf.WriteString(metric.Value)

	// timestamp
	buf.Write(space)
	if metric.Timestamp == 0 {
		metric.Timestamp = time.Now().Unix()
	}
	buf.WriteString(strconv.Itoa(int(metric.Timestamp)))
	buf.Write(breakLine)

	// "name value ts\n"
	graphite.conn.Write(buf.Bytes())
}

// The SimpleSend method can be used to just pass a metric name and value and
// have it be sent to the Graphite host with the current timestamp
func (graphite *Graphite) SimpleSend(stat string, value string) error {
	if graphite.IsNop() {
		return nil
	}

	m := Metric{
		Name:      stat,
		Value:     value,
		Timestamp: time.Now().Unix(),
	}

	select {
	case graphite.ms <- m:
	default:
	}

	return nil
}

// When a UDP connection to Graphite is required
func NewGraphiteUDP(host string, port int) (*Graphite, error) {
	return GraphiteFactory("udp", host, port, "")
}

// NewGraphiteNop is a factory method that returns a Graphite struct but will
// not actually try to send any packets to a remote host and, instead, will just
// log. This is useful if you want to use Graphite in a project but don't want
// to make Graphite a requirement for the project.
func NewGraphiteNop(host string, port int) *Graphite {
	graphiteNop, _ := GraphiteFactory("nop", host, port, "")
	return graphiteNop
}

func GraphiteFactory(protocol string, host string, port int, prefix string) (*Graphite, error) {
	var graphite *Graphite

	switch protocol {
	case "tcp":
		graphite = &Graphite{Host: host, Port: port, Protocol: "tcp", Prefix: prefix, ms: make(chan Metric, defaultMetricsLen)}
	case "udp":
		graphite = &Graphite{Host: host, Port: port, Protocol: "udp", ms: make(chan Metric, defaultMetricsLen)}
	case "nop":
		graphite = &Graphite{Host: host, Port: port, nop: true, ms: make(chan Metric, defaultMetricsLen)}
	}

	err := graphite.Connect()
	if err != nil {
		return nil, err
	}

	return graphite, nil
}
