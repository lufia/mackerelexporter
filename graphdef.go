package mackerel

import (
	"errors"
	"strings"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/unit"

	"github.com/mackerelio/mackerel-client-go"
)

// OpenTelemetry naming
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/api-metrics-user.md

const (
	UnitDimensionless = unit.Dimensionless
	UnitBytes         = unit.Bytes
	UnitMilliseconds  = unit.Milliseconds

	metricNameSep = "."
)

// GraphDefOptions represents options for customizing Mackerel's Graph Definition.
type GraphDefOptions struct {
	Name       string
	MetricName string
	Unit       unit.Unit
	Kind       core.NumberKind
}

// NewGraphDef returns Mackerel's Graph Definition that has only one metric in Metrics field.
// Each names in arguments must be normalized.
func NewGraphDef(name string, opts GraphDefOptions) (*mackerel.GraphDefsParam, error) {
	if opts.Unit == "" {
		opts.Unit = UnitDimensionless
	}
	switch {
	case opts.MetricName == "" && opts.Name == "":
		opts.MetricName = generalizeMetricName(name)
		opts.Name = opts.MetricName
	case opts.MetricName == "" && opts.Name != "":
		s, err := bindGraphMetricName(opts.Name, name)
		if err != nil {
			return nil, err
		}
		opts.MetricName = s
	case opts.MetricName != "" && opts.Name == "":
		opts.Name = opts.MetricName
	}
	if !MetricName(opts.MetricName).Match(name) {
		return nil, errMismatch
	}
	return &mackerel.GraphDefsParam{
		Name: "custom." + opts.Name,
		Unit: GraphUnit(opts.Unit),
		Metrics: []*mackerel.GraphDefsMetric{
			{Name: "custom." + opts.MetricName},
		},
	}, nil
}

func GraphUnit(u unit.Unit) string {
	// TODO(lufia): desc.NumberKind
	switch u {
	case unit.Dimensionless:
		return "float"
	case unit.Bytes:
		return "bytes"
	case unit.Milliseconds:
		return "float"
	default:
		return "integer"
	}
}

var errMismatch = errors.New("mismatched metric names")

// bindGraphMetricName returns s1 + rest of s2.
func bindGraphMetricName(s1, s2 string) (string, error) {
	a1 := strings.Split(s1, metricNameSep)
	a2 := strings.Split(s2, metricNameSep)
	if len(a1) > len(a2) {
		return "", errMismatch
	}
	t := strings.Join(a2[:len(a1)], metricNameSep)
	if !MetricName(s1).Match(t) {
		return "", errMismatch
	}
	copy(a2[:len(a1)], a1)
	return strings.Join(a2, metricNameSep), nil
}

// generalizeMetricName generalize "a.b" to "a.*" if s don't contain wildcards.
func generalizeMetricName(s string) string {
	if s == "" {
		return ""
	}
	a := strings.Split(s, metricNameSep)
	for _, stem := range a {
		if stem == "*" {
			return s
		}
	}
	if a[len(a)-1] == "#" {
		return s
	}
	a[len(a)-1] = "*"
	return strings.Join(a, metricNameSep)
}

// NormalizeMetricName returns normalized s.
func NormalizeMetricName(s string) string {
	normalize := func(c rune) rune {
		switch {
		case c >= '0' && c <= '9':
			return c
		case c >= 'a' && c <= 'z':
			return c
		case c >= 'A' && c <= 'Z':
			return c
		case c == '-' || c == '_' || c == '.' || c == '#' || c == '*':
			return c
		default:
			return '_'
		}
	}
	return strings.Map(normalize, s)
}

type MetricName string

func (g MetricName) Match(s string) bool {
	expr := strings.Split(string(g), metricNameSep)
	a := strings.Split(s, metricNameSep)
	if len(expr) != len(a) {
		return false
	}
	for i := range expr {
		if expr[i] == "#" || expr[i] == "*" {
			continue
		}
		if expr[i] != a[i] {
			return false
		}
	}
	return true
}

// see https://mackerel.io/docs/entry/spec/metrics
var systemMetrics = []MetricName{
	/* Linux */
	"loadavg1",
	"loadavg5",
	"loadavg15",
	"cpu.user.percentage",
	"cpu.iowait.percentage",
	"cpu.system.percentage",
	"cpu.idle.percentage",
	"cpu.nice.percentage",
	"cpu.irq.percentage",
	"cpu.softirq.percentage",
	"cpu.steal.percentage",
	"cpu.guest.percentage",
	"memory.used",
	"memory.available",
	"memory.total",
	"memory.swap_used",
	"memory.swap_cached",
	"memory.swap_total",
	"memory.free",
	"memory.buffers",
	"memory.cached",
	"memory.used",
	"memory.total",
	"memory.swap_used",
	"memory.swap_cached",
	"memory.swap_total",
	"disk.*.reads.delta",
	"disk.*.writes.delta",
	"interface.*.rxBytes.delta",
	"interface.*.txBytes.delta",
	"filesystem.*.size",
	"filesystem.*.used",

	/* Windows */
	"processor_queue_length",
	"cpu.user.percentage",
	"cpu.system.percentage",
	"cpu.idle.percentage",
	"memory.free",
	"memory.used",
	"memory.total",
	"memory.pagefile_free",
	"memory.pagefile_total",
	"disk.*.reads.delta",
	"disk.*.writes.delta",
	"interface.*.rxBytes.delta",
	"interface.*.txBytes.delta",
	"filesystem.*.size",
	"filesystem.*.used",
}

var systemMetricNames map[MetricName]struct{}

func init() {
	systemMetricNames = make(map[MetricName]struct{})
	for _, s := range systemMetrics {
		systemMetricNames[s] = struct{}{}
	}
}

// IsSystemMetric returns whether s is system metric in Mackerel.
func IsSystemMetric(s string) bool {
	s = NormalizeMetricName(s)
	for m := range systemMetricNames {
		if m.Match(s) {
			return true
		}
	}
	return false
}
