package base

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/antonmedv/expr"
	log "github.com/iter8-tools/iter8/base/log"
	"github.com/montanaflynn/stats"
	"helm.sh/helm/v3/pkg/time"
)

// Task is the building block of an experiment spec
// An experiment spec is a sequence of tasks
type Task interface {
	// validateInputs for this task
	validateInputs() error

	// initializeDefaults of the input values to this task
	initializeDefaults()

	// run this task
	run(exp *Experiment) error
}

// ExperimentSpec specifies the set of tasks in this experiment
type ExperimentSpec []Task

// Experiment struct containing spec and result
type Experiment struct {
	// Spec is the sequence of tasks that constitute this experiment
	Spec ExperimentSpec `json:"spec" yaml:"spec"`

	// Result is the current results from this experiment.
	// The experiment may not have completed in which case results may be partial.
	Result *ExperimentResult `json:"result" yaml:"result"`

	// driver enables interacting with experiment result stored externally
	driver Driver
}

// ExperimentResult defines the current results from the experiment
type ExperimentResult struct {
	// Revision of this experiment
	Revision int `json:"revision,omitempty" yaml:"revision,omitempty"`

	// StartTime is the time when the experiment run started
	StartTime time.Time `json:"startTime" yaml:"startTime"`

	// NumLoops is the number of iterations this experiment has been running for
	NumLoops int `json:"numLoops" yaml:"numLoops"`

	// NumCompletedTasks is the number of completed tasks
	NumCompletedTasks int `json:"numCompletedTasks" yaml:"numCompletedTasks"`

	// Failure is true if any of its tasks failed
	Failure bool `json:"failure" yaml:"failure"`

	// Insights produced in this experiment
	Insights *Insights `json:"insights,omitempty" yaml:"insights,omitempty"`

	// Iter8Version is the version of Iter8 CLI that created this result object
	Iter8Version string `json:"iter8Version" yaml:"iter8Version"`
}

// Insights records the number of versions in this experiment,
// metric values and SLO indicators for each version,
// metrics metadata for all metrics, and
// SLO definitions for all SLOs
type Insights struct {
	// NumVersions is the number of app versions detected by Iter8
	NumVersions int `json:"numVersions" yaml:"numVersions"`

	// MetricsInfo identifies the metrics involved in this experiment
	MetricsInfo map[string]MetricMeta `json:"metricsInfo,omitempty" yaml:"metricsInfo,omitempty"`

	// NonHistMetricValues:
	// the outer slice must be the same length as the number of app versions
	// the map key must match name of a metric in MetricsInfo
	// the inner slice contains the list of all observed metric values for given version and given metric; float value [i]["foo/bar"][k] is the [k]th observation for version [i] for the metric bar under backend foo.
	// this struct is meant exclusively for metrics of type other than histogram
	NonHistMetricValues []map[string][]float64 `json:"nonHistMetricValues,omitempty" yaml:"nonHistMetricValues,omitempty"`

	// HistMetricValues:
	// the outer slice must be the same length as the number of app versions
	// the map key must match name of a histogram metric in MetricsInfo
	// the inner slice contains the list of all observed histogram buckets for a given version and given metric; value [i]["foo/bar"][k] is the [k]th observed bucket for version [i] for the hist metric `bar` under backend `foo`.
	HistMetricValues []map[string][]HistBucket `json:"histMetricValues,omitempty" yaml:"histMetricValues,omitempty"`

	// SLOs involved in this experiment
	SLOs *SLOLimits `json:"SLOs,omitempty" yaml:"SLOs,omitempty"`

	// SLOsSatisfied indicator matrices that show if upper and lower SLO limits are satisfied
	SLOsSatisfied *SLOResults `json:"SLOsSatisfied,omitempty" yaml:"SLOsSatisfied,omitempty"`
}

// MetricMeta describes a metric
type MetricMeta struct {
	// Description is a human readable description of the metric
	Description string `json:"description" yaml:"description"`
	// Units for this metric (if any)
	Units *string `json:"units,omitempty" yaml:"units,omitempty"`
	// Type of the metric. Example: counter
	Type MetricType `json:"type" yaml:"type"`
}

// SLO is a service level objective
type SLO struct {
	// Metric is the fully qualified metric name in the backendName/metricName format
	Metric string `json:"metric" yaml:"metric"`

	// Limit is the acceptable limit for this metric
	Limit float64 `json:"limit" yaml:"limit"`
}

// SLOLimits specify upper or lower limits for metrics
type SLOLimits struct {
	// Upper limits for metrics
	Upper []SLO `json:"upper,omitempty" yaml:"upper,omitempty"`

	// Lower limits for metrics
	Lower []SLO `json:"lower,omitempty" yaml:"lower,omitempty"`
}

// SLOResults specify the results of SLO evaluations
type SLOResults struct {
	// Upper limits for metrics
	// Upper[i][j] specifies if upper SLO i is satisfied by version j
	Upper [][]bool `json:"upper,omitempty" yaml:"upper,omitempty"`

	// Lower limits for metrics
	// Lower[i][j] specifies if lower SLO i is satisfied by version j
	Lower [][]bool `json:"lower,omitempty" yaml:"lower,omitempty"`
}

// TaskMeta provides common fields used across all tasks
type TaskMeta struct {
	// Task is the name of the task
	Task *string `json:"task,omitempty" yaml:"task,omitempty"`
	// Run is the script used in a run task
	// Specify either Task or Run but not both
	Run *string `json:"run,omitempty" yaml:"run,omitempty"`
	// If is the condition used to determine if this task needs to run
	// If the condition is not satisfied, then it is skipped in an experiment
	// Example: SLOs()
	If *string `json:"if,omitempty" yaml:"if,omitempty"`
}

// taskMetaWith enables unmarshaling of tasks
type taskMetaWith struct {
	// TaskMeta has fields common to all tasks
	TaskMeta
	// With is the raw representation of task inputs
	With map[string]interface{} `json:"with,omitempty" yaml:"with,omitempty"`
}

// UnmarshallJSON will unmarshal an experiment spec from bytes
// This is a custom JSON unmarshaler
func (s *ExperimentSpec) UnmarshalJSON(data []byte) error {
	var v []taskMetaWith
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	log.Logger.Tracef("unmarshaled %v tasks into task meta", len(v))

	for _, t := range v {
		if (t.Task == nil || len(*t.Task) == 0) && (t.Run == nil) {
			err := fmt.Errorf("invalid task found without a task name or a run command")
			log.Logger.Error(err)
			return err
		}

		// get byte data for this task
		tBytes, _ := json.Marshal(t)
		var tsk Task
		// this is a run task
		if t.Run != nil {
			log.Logger.Debug("found run task: ", *t.Run)
			rt := &runTask{}
			json.Unmarshal(tBytes, rt)
			tsk = rt
		} else {
			// this is some other task
			switch *t.Task {
			case ReadinessTaskName:
				rt := &readinessTask{}
				json.Unmarshal(tBytes, rt)
				tsk = rt
			case CustomMetricsTaskName:
				cdt := &customMetricsTask{}
				err := json.Unmarshal(tBytes, cdt)
				if err != nil {
					e := errors.New("json unmarshal error")
					log.Logger.WithStackTrace(err.Error()).Error(e)
					return e
				}
				tsk = cdt
			case CollectHTTPTaskName:
				cht := &collectHTTPTask{}
				err := json.Unmarshal(tBytes, cht)
				if err != nil {
					e := errors.New("json unmarshal error")
					log.Logger.WithStackTrace(err.Error()).Error(e)
					return e
				}
				tsk = cht
			case CollectGRPCTaskName:
				cgt := &collectGRPCTask{}
				err := json.Unmarshal(tBytes, cgt)
				if err != nil {
					e := errors.New("json unmarshal error")
					log.Logger.WithStackTrace(err.Error()).Error(e)
					return e
				}
				tsk = cgt
			case AssessTaskName:
				at := &assessTask{}
				err := json.Unmarshal(tBytes, at)
				if err != nil {
					e := errors.New("json unmarshal error")
					log.Logger.WithStackTrace(err.Error()).Error(e)
					return e
				}
				tsk = at
			default:
				log.Logger.Error("unknown task: " + *t.Task)
				return errors.New("unknown task: " + *t.Task)
			}
		}
		n := append(*s, tsk)
		*s = n
		log.Logger.Trace("appended to experiment spec")
	}
	log.Logger.Trace("constructed experiment spec of length: ", len(*s))
	return nil
}

// metricTypeMatch checks if metric value is a match for its type
func metricTypeMatch(t MetricType, val interface{}) bool {
	switch v := val.(type) {
	case float64:
		if t == CounterMetricType || t == GaugeMetricType {
			return true
		} else {
			return false
		}
	case []float64:
		if t == SampleMetricType {
			return true
		} else {
			return false
		}
	case []HistBucket:
		if t == HistogramMetricType {
			return true
		} else {
			return false
		}
	default:
		log.Logger.Error("unsupported type for metric value: ", v)
		return false
	}
}

// updateMetricValueScalar updates a scalar metric value for a given version
func (in *Insights) updateMetricValueScalar(m string, i int, val float64) {
	in.NonHistMetricValues[i][m] = append(in.NonHistMetricValues[i][m], val)
}

// updateMetricValueVector updates a vector metric value for a given version
func (in *Insights) updateMetricValueVector(m string, i int, val []float64) {
	in.NonHistMetricValues[i][m] = append(in.NonHistMetricValues[i][m], val...)
}

// updateMetricValueHist updates a histogram metric value for a given version
func (in *Insights) updateMetricValueHist(m string, i int, val []HistBucket) {
	in.HistMetricValues[i][m] = append(in.HistMetricValues[i][m], val...)
}

// registerMetric registers a new metric by adding its meta data
func (in *Insights) registerMetric(m string, mm MetricMeta) error {
	if old, ok := in.MetricsInfo[m]; ok && !reflect.DeepEqual(old, mm) {
		err := fmt.Errorf("old and new metric meta for %v differ", m)
		log.Logger.WithStackTrace(fmt.Sprintf("old: %v \nnew: %v", old, mm)).Error(err)
		return err
	}
	in.MetricsInfo[m] = mm
	return nil
}

// updateMetric registers a metric and adds a metric value for a given version
// metric names will be normalized
func (in *Insights) updateMetric(m string, mm MetricMeta, i int, val interface{}) error {
	var err error
	if !metricTypeMatch(mm.Type, val) {
		err = fmt.Errorf("metric value and type are incompatible; name: %v meta: %v version: %v value: %v", m, mm, i, val)
		log.Logger.Error(err)
		return err
	}

	if in.NumVersions <= i {
		err := fmt.Errorf("insufficient number of versions %v with version index %v", in.NumVersions, i)
		log.Logger.Error(err)
		return err
	}

	nm, err := NormalizeMetricName(m)
	if err != nil {
		return err
	}

	err = in.registerMetric(nm, mm)
	if err != nil {
		return err
	}

	switch mm.Type {
	case CounterMetricType, GaugeMetricType:
		in.updateMetricValueScalar(nm, i, val.(float64))
	case SampleMetricType:
		in.updateMetricValueVector(nm, i, val.([]float64))
	case HistogramMetricType:
		in.updateMetricValueHist(nm, i, val.([]HistBucket))
	default:
		err := fmt.Errorf("unknown metric type %v", mm.Type)
		log.Logger.Error(err)
	}
	return nil
}

// setSLOs sets the SLOs field in insights
// if this function is called multiple times (example, due to looping), then
// it is intended to be called with the same argument each time
func (in *Insights) setSLOs(slos *SLOLimits) error {
	if in.SLOs != nil {
		if reflect.DeepEqual(in.SLOs, slos) {
			return nil
		} else {
			e := fmt.Errorf("old and new value of slos conflict")
			log.Logger.WithStackTrace(fmt.Sprint("old: ", in.SLOs, "new: ", slos)).Error(e)
			return e
		}
	}
	// LHS will be nil
	in.SLOs = slos
	return nil
}

// initializeSLOsSatisfied initializes the SLOs satisfied field
func (e *Experiment) initializeSLOsSatisfied() error {
	if e.Result.Insights.SLOsSatisfied != nil {
		return nil // already initialized
	}
	// LHS will be nil
	e.Result.Insights.SLOsSatisfied = &SLOResults{
		Upper: make([][]bool, 0),
		Lower: make([][]bool, 0),
	}
	if e.Result.Insights.SLOs != nil {
		e.Result.Insights.SLOsSatisfied.Upper = make([][]bool, len(e.Result.Insights.SLOs.Upper))
		for i := 0; i < len(e.Result.Insights.SLOs.Upper); i++ {
			e.Result.Insights.SLOsSatisfied.Upper[i] = make([]bool, e.Result.Insights.NumVersions)
		}
		e.Result.Insights.SLOsSatisfied.Lower = make([][]bool, len(e.Result.Insights.SLOs.Lower))
		for i := 0; i < len(e.Result.Insights.SLOs.Lower); i++ {
			e.Result.Insights.SLOsSatisfied.Lower[i] = make([]bool, e.Result.Insights.NumVersions)
		}
	}
	return nil
}

// initResults initializes the results section of an experiment
func (e *Experiment) initResults(revision int) {
	e.Result = &ExperimentResult{
		Revision:          revision,
		StartTime:         time.Now(),
		NumLoops:          0,
		NumCompletedTasks: 0,
		Failure:           false,
		Iter8Version:      MajorMinor,
	}
}

// initInsightsWithNumVersions is also going to initialize insights data structure
// insights data structure contains metrics data structures, so this will also
// init metrics
func (r *ExperimentResult) initInsightsWithNumVersions(n int) error {
	if r.Insights != nil {
		if r.Insights.NumVersions != n {
			err := fmt.Errorf("inconsistent number for app versions; old (%v); new (%v)", r.Insights.NumVersions, n)
			log.Logger.Error(err)
			return err
		}
	} else {
		r.Insights = &Insights{
			NumVersions: n,
		}
	}
	r.Insights.initMetrics()
	return nil
}

// initMetrics initializes the data structes inside insights that will hold metrics
func (in *Insights) initMetrics() error {
	if in.NonHistMetricValues != nil || in.HistMetricValues != nil {
		if len(in.NonHistMetricValues) != in.NumVersions || len(in.HistMetricValues) != in.NumVersions {
			err := fmt.Errorf("inconsistent number for app versions in non hist metric values (%v), hist metric values (%v), num versions (%v)", len(in.NonHistMetricValues), len(in.HistMetricValues), in.NumVersions)
			log.Logger.Error(err)
			return err
		}
		if len(in.NonHistMetricValues[0])+len(in.HistMetricValues[0]) != len(in.MetricsInfo) {
			err := fmt.Errorf("inconsistent number for metrics in non hist metric values (%v), hist metric values (%v), metrics info (%v)", len(in.NonHistMetricValues[0]), len(in.HistMetricValues[0]), len(in.MetricsInfo))
			log.Logger.Error(err)
			return err
		}
		return nil
	}
	// at this point, there are no known metrics, but there are in.NumVersions
	// initialize metrics info
	in.MetricsInfo = make(map[string]MetricMeta)
	// initialize non hist metric values for each version
	in.NonHistMetricValues = make([]map[string][]float64, in.NumVersions)
	// initialize hist metric values for each version
	in.HistMetricValues = make([]map[string][]HistBucket, in.NumVersions)
	for i := 0; i < in.NumVersions; i++ {
		in.NonHistMetricValues[i] = make(map[string][]float64)
		in.HistMetricValues[i] = make(map[string][]HistBucket)
	}
	return nil
}

// getCounterOrGaugeMetricFromValuesMap gets the value of the given counter or gauge metric, for the given version, from metric values map
func (in *Insights) getCounterOrGaugeMetricFromValuesMap(i int, m string) *float64 {
	if mm, ok := in.MetricsInfo[m]; ok {
		log.Logger.Tracef("found metric info for %v", m)
		if (mm.Type != CounterMetricType) && (mm.Type != GaugeMetricType) {
			log.Logger.Errorf("metric %v is not of type counter or gauge", m)
			return nil
		}
		l := len(in.NonHistMetricValues)
		if l <= i {
			log.Logger.Warnf("metric values not found for version %v; initialized for %v versions", i, l)
			return nil
		}
		log.Logger.Tracef("metric values found for version %v", i)
		// grab the metric value and return
		if vals, ok := in.NonHistMetricValues[i][m]; ok {
			log.Logger.Tracef("found metric value for version %v and metric %v", i, m)
			if len(vals) > 0 {
				return float64Pointer(vals[len(vals)-1])
			}
		}
		log.Logger.Infof("could not find metric value for version %v and metric %v", i, m)
	}
	log.Logger.Infof("could not find metric info for %v", m)
	return nil
}

// getSampleAggregation aggregates the given base metric for the given version (i) with the given aggregation (a)
func (in *Insights) getSampleAggregation(i int, baseMetric string, a string) *float64 {
	at := AggregationType(a)
	vals := in.NonHistMetricValues[i][baseMetric]
	if len(vals) == 0 {
		log.Logger.Infof("metric %v for version %v has no sample", baseMetric, i)
	}
	if len(vals) == 1 {
		log.Logger.Warnf("metric %v for version %v has sample of size 1", baseMetric, i)
		return float64Pointer(vals[0])
	}
	switch at {
	case MeanAggregator:
		agg, err := stats.Mean(vals)
		if err == nil {
			return float64Pointer(agg)
		} else {
			log.Logger.WithStackTrace(err.Error()).Errorf("aggregation error for version %v, metric %v, and aggregation func %v", i, baseMetric, a)
			return nil
		}
	case StdDevAggregator:
		agg, err := stats.StandardDeviation(vals)
		if err == nil {
			return float64Pointer(agg)
		} else {
			log.Logger.WithStackTrace(err.Error()).Errorf("aggregation error version %v, metric %v, and aggregation func %v", i, baseMetric, a)
			return nil
		}
	case MinAggregator:
		agg, err := stats.Min(vals)
		if err == nil {
			return float64Pointer(agg)
		} else {
			log.Logger.WithStackTrace(err.Error()).Errorf("aggregation error version %v, metric %v, and aggregation func %v", i, baseMetric, a)
			return nil
		}
	case MaxAggregator:
		agg, err := stats.Mean(vals)
		if err == nil {
			return float64Pointer(agg)
		} else {
			log.Logger.WithStackTrace(err.Error()).Errorf("aggregation error version %v, metric %v, and aggregation func %v", i, baseMetric, a)
			return nil
		}
	default: // don't do anything
	}

	// at this point, 'a' must be a percentile aggregator
	if strings.HasPrefix(a, "p") {
		b := strings.TrimPrefix(a, "p")
		// b must be a percent
		if match, _ := regexp.MatchString(decimalRegex, b); match {
			// extract percent
			if percent, err := strconv.ParseFloat(b, 64); err != nil {
				log.Logger.WithStackTrace(err.Error()).Errorf("error extracting percent from aggregation func %v", a)
				return nil
			} else {
				// compute percentile
				agg, err := stats.Percentile(vals, percent)
				if err == nil {
					return float64Pointer(agg)
				} else {
					log.Logger.WithStackTrace(err.Error()).Errorf("aggregation error version %v, metric %v, and aggregation func %v", i, baseMetric, a)
					return nil
				}
			}
		} else {
			log.Logger.Errorf("unable to extract percent from agggregation func %v", a)
			return nil
		}
	} else {
		log.Logger.Errorf("invalid aggregation %v", a)
		return nil
	}
}

// aggregateMetric returns the aggregated metric value for a given version and metric
func (in *Insights) aggregateMetric(i int, m string) *float64 {
	s := strings.Split(m, "/")
	baseMetric := s[0] + "/" + s[1]
	if m, ok := in.MetricsInfo[baseMetric]; ok {
		log.Logger.Tracef("found metric %v used for aggregation", baseMetric)
		if m.Type == SampleMetricType {
			log.Logger.Tracef("metric %v used for aggregation is a sample metric", baseMetric)
			return in.getSampleAggregation(i, baseMetric, s[2])
		} else {
			log.Logger.Errorf("metric %v used for aggregation is not a sample metric", baseMetric)
			return nil
		}
	} else {
		log.Logger.Warnf("could not find metric %v used for aggregation", baseMetric)
		return nil
	}
}

// NormalizeMetricName normalizes percentile values in metric names
func NormalizeMetricName(m string) (string, error) {
	preHTTP := httpMetricPrefix + "/" + builtInHTTPLatencyPercentilePrefix
	preGRPC := gRPCMetricPrefix + "/" + gRPCLatencySampleMetricName + "/" + PercentileAggregatorPrefix
	pre := ""
	if strings.HasPrefix(m, preHTTP) { // built-in http percentile metric
		pre = preHTTP
	} else if strings.HasPrefix(m, preGRPC) { // built-in gRPC percentile metric
		pre = preGRPC
	}
	if len(pre) > 0 {
		remainder := strings.TrimPrefix(m, pre)
		if percent, e := strconv.ParseFloat(remainder, 64); e != nil {
			err := fmt.Errorf("cannot extract percent from metric %v", m)
			log.Logger.WithStackTrace(e.Error()).Error(err)
			return m, err
		} else {
			// return percent normalized metric name
			return fmt.Sprintf("%v%v", pre, percent), nil
		}
	} else {
		// already normalized
		return m, nil
	}
}

// ScalarMetricValue gets the value of the given scalar metric for the given version
func (in *Insights) ScalarMetricValue(i int, m string) *float64 {
	s := strings.Split(m, "/")
	if len(s) == 3 {
		log.Logger.Tracef("%v is an aggregated metric", m)
		return in.aggregateMetric(i, m)
	} else if len(s) == 2 { // this appears to be a non-aggregated metric
		if nm, err := NormalizeMetricName(m); err != nil {
			return nil
		} else {
			return in.getCounterOrGaugeMetricFromValuesMap(i, nm)
		}
	} else {
		log.Logger.Errorf("invalid metric name %v", m)
		log.Logger.Error("metric names must be of the form a/b or a/b/c, where a is the id of the metrics backend, b is the id of a metric name, and c is a valid aggregation function")
		return nil
	}
}

// GetMetricsInfo gets metric meta for the given normalized metric name
func (in *Insights) GetMetricsInfo(nm string) (*MetricMeta, error) {
	s := strings.Split(nm, "/")

	// this is an aggregated metric
	if len(s) == 3 {
		log.Logger.Tracef("%v is an aggregated metric", nm)
		vm := s[0] + "/" + s[1]
		mm, ok := in.MetricsInfo[vm]
		if !ok {
			err := fmt.Errorf("unable to find info for vector metric: %v", vm)
			log.Logger.Error(err)
			return nil, err
		}
		// determine type of aggregation
		aggType := CounterMetricType
		if AggregationType(s[2]) != CountAggregator {
			aggType = GaugeMetricType
		}
		// format aggregator text
		formattedAggregator := s[2] + " value"
		if strings.HasPrefix(s[2], PercentileAggregatorPrefix) {
			percent := strings.TrimPrefix(s[2], PercentileAggregatorPrefix)
			formattedAggregator = fmt.Sprintf("%v-th percentile value", percent)
		}
		// return metrics meta
		return &MetricMeta{
			Description: fmt.Sprintf("%v of %v", formattedAggregator, vm),
			Units:       mm.Units,
			Type:        aggType,
		}, nil
	}

	// this is a non-aggregated metric
	if len(s) == 2 {
		mm, ok := in.MetricsInfo[nm]
		if !ok {
			err := fmt.Errorf("unable to find info for scalar metric: %v", nm)
			log.Logger.Error(err)
			return nil, err
		}
		return &mm, nil
	}

	err := fmt.Errorf("invalid metric name %v; metric names must be of the form a/b or a/b/c, where a is the id of the metrics backend, b is the id of a metric name, and c is a valid aggregation function", nm)
	log.Logger.Error(err)
	return nil, err
}

// Driver enables interacting with experiment result stored externally
type Driver interface {
	// Read the experiment
	Read() (*Experiment, error)

	// Write the experiment
	Write(e *Experiment) error

	// GetRevision returns the experiment revision
	GetRevision() int
}

// Completed returns true if the experiment is complete
func (exp *Experiment) Completed() bool {
	if exp != nil {
		if exp.Result != nil {
			if exp.Result.NumCompletedTasks == len(exp.Spec) {
				return true
			}
		}
	}
	return false
}

// NoFailure returns true if no task in the experiment has failed
func (exp *Experiment) NoFailure() bool {
	return exp != nil && exp.Result != nil && !exp.Result.Failure
}

// getSLOsSatisfiedBy returns the set of versions which satisfy SLOs
func (exp *Experiment) getSLOsSatisfiedBy() []int {
	if exp == nil {
		log.Logger.Warning("nil experiment")
		return nil
	}
	if exp.Result == nil {
		log.Logger.Warning("nil experiment result")
		return nil
	}
	if exp.Result.Insights == nil {
		log.Logger.Warning("nil insights in experiment result")
		return nil
	}
	if exp.Result.Insights.NumVersions == 0 {
		log.Logger.Warning("experiment does not involve any versions")
		return nil
	}
	if exp.Result.Insights.SLOs == nil {
		log.Logger.Info("experiment does not involve any SLOs")
		sat := []int{}
		for j := 0; j < exp.Result.Insights.NumVersions; j++ {
			sat = append(sat, j)
		}
		return sat
	}
	log.Logger.Debug("experiment involves at least one version and at least one SLO")
	log.Logger.Trace(exp.Result.Insights.SLOs)
	log.Logger.Trace(exp.Result.Insights.SLOsSatisfied)
	log.Logger.Trace(exp.Result.Insights.NonHistMetricValues)
	sat := []int{}
	for j := 0; j < exp.Result.Insights.NumVersions; j++ {
		satThis := true
		for i := 0; i < len(exp.Result.Insights.SLOs.Upper); i++ {
			satThis = satThis && exp.Result.Insights.SLOsSatisfied.Upper[i][j]
			if !satThis {
				break
			}
		}
		for i := 0; i < len(exp.Result.Insights.SLOs.Lower); i++ {
			satThis = satThis && exp.Result.Insights.SLOsSatisfied.Lower[i][j]
			if !satThis {
				break
			}
		}
		if satThis {
			sat = append(sat, j)
		}
	}
	return sat
}

// SLOs returns true if all versions satisfy SLOs
func (exp *Experiment) SLOs() bool {
	if exp == nil || exp.Result == nil || exp.Result.Insights == nil {
		log.Logger.Warning("experiment, or result, or insights is nil")
		return false
	}
	sby := exp.getSLOsSatisfiedBy()
	return exp.Result.Insights.NumVersions == len(sby)
}

// run the experiment
func (exp *Experiment) run(driver Driver) error {
	var err error
	exp.driver = driver
	if exp.Result == nil {
		err = errors.New("experiment with nil result section cannot be run")
		log.Logger.Error(err)
		return err
	}

	log.Logger.Debug("exp result exists now ... ")

	exp.incrementNumLoops()
	log.Logger.Debugf("experiment loop %d started ...", exp.Result.NumLoops)
	err = driver.Write(exp)
	if err != nil {
		return err
	}

	log.Logger.Debugf("attempting to execute %v tasks", len(exp.Spec))
	for i, t := range exp.Spec {
		log.Logger.Info("task " + fmt.Sprintf("%v: %v", i+1, *getName(t)) + " : started")
		shouldRun := true
		// if task has a condition
		if cond := getIf(t); cond != nil {
			// condition evaluates to false ... then shouldRun is false
			program, err := expr.Compile(*cond, expr.Env(exp), expr.AsBool())
			if err != nil {
				log.Logger.WithStackTrace(err.Error()).Error("unable to compile if clause")
				return err
			}

			output, err := expr.Run(program, exp)
			if err != nil {
				log.Logger.WithStackTrace(err.Error()).Error("unable to run if clause")
				return err
			}

			shouldRun = output.(bool)
		}
		if shouldRun {
			err = t.run(exp)
			if err != nil {
				log.Logger.Error("task " + fmt.Sprintf("%v: %v", i+1, *getName(t)) + " : " + "failure")
				exp.failExperiment()
				e := driver.Write(exp)
				if e != nil {
					return e
				}
				return err
			}
			log.Logger.Info("task " + fmt.Sprintf("%v: %v", i+1, *getName(t)) + " : " + "completed")
		} else {
			log.Logger.WithStackTrace(fmt.Sprint("false condition: ", *getIf(t))).Info("task " + fmt.Sprintf("%v: %v", i+1, *getName(t)) + " : " + "skipped")
		}

		exp.incrementNumCompletedTasks()
		err = driver.Write(exp)
		if err != nil {
			return err
		}
	}
	return nil
}

// failExperiment sets the experiment failure status to true
func (e *Experiment) failExperiment() {
	e.Result.Failure = true
}

// incrementNumCompletedTasks increments the number of completed tasks in the experimeent
func (e *Experiment) incrementNumCompletedTasks() {
	e.Result.NumCompletedTasks++
}

// incrementNumLoops increments the number of loops (experiment iterations)
func (e *Experiment) incrementNumLoops() {
	e.Result.NumLoops++
}

// getIf returns the condition (if any) which determine
// whether of not if this task needs to run
func getIf(t Task) *string {
	var jsonBytes []byte
	var tm TaskMeta
	// convert t to jsonBytes
	jsonBytes, _ = json.Marshal(t)
	// convert jsonBytes to TaskMeta
	_ = json.Unmarshal(jsonBytes, &tm)
	return tm.If
}

// getName returns the name of this task
func getName(t Task) *string {
	var jsonBytes []byte
	var tm TaskMeta
	// convert t to jsonBytes
	jsonBytes, _ = json.Marshal(t)
	// convert jsonBytes to TaskMeta
	_ = json.Unmarshal(jsonBytes, &tm)

	if tm.Task == nil {
		if tm.Run != nil {
			return StringPointer(RunTaskName)
		}
	} else {
		return tm.Task
	}
	log.Logger.Error("task spec with no name or run value")
	return nil
}

// BuildExperiment builds an experiment
func BuildExperiment(driver Driver) (*Experiment, error) {
	e, err := driver.Read()
	if err != nil {
		return nil, err
	}
	return e, nil
}

// RunExperiment runs an experiment
func RunExperiment(reuseResult bool, driver Driver) error {
	if exp, err := BuildExperiment(driver); err != nil {
		return err
	} else {
		if !reuseResult {
			exp.initResults(driver.GetRevision())
		}
		return exp.run(driver)
	}
}
