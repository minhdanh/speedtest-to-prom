package prometheus

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/snappy"
	pb "github.com/minhdanh/speedtest-to-prom/internal/prometheus"
	"google.golang.org/protobuf/proto"
)

// MetricValue represents a metric value with its labels
type MetricValue struct {
	Value  float64
	Labels map[string]string
}

func PushPrometheusMetrics(prometheusUrl, promUsername, promPassword string, metrics map[string][]MetricValue, labels []*pb.Label) error {
	credentials := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", promUsername, promPassword)))
	timestamp := time.Now().UnixMilli()

	var timeseriesSlice []*pb.TimeSeries

	for metricName, values := range metrics {
		for _, value := range values {
			// Create a copy of base labels
			metricLabels := append([]*pb.Label{}, labels...)

			// Add metric-specific labels
			metricLabels = append(metricLabels, &pb.Label{
				Name:  "__name__",
				Value: metricName,
			})

			// Add value-specific labels
			for k, v := range value.Labels {
				metricLabels = append(metricLabels, &pb.Label{
					Name:  k,
					Value: v,
				})
			}

			timeseries := &pb.TimeSeries{
				Labels: metricLabels,
				Samples: []*pb.Sample{
					{
						Value:     value.Value,
						Timestamp: timestamp,
					},
				},
			}
			timeseriesSlice = append(timeseriesSlice, timeseries)
		}
	}

	writeRequest := &pb.WriteRequest{
		Timeseries: timeseriesSlice,
	}

	// Serialize to protobuf
	data, err := proto.Marshal(writeRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf: %v", err)
	}

	// Compress with snappy
	compressed := snappy.Encode(nil, data)

	// Send request
	req, err := http.NewRequest("POST", prometheusUrl, bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("User-Agent", "sono-speedtest/1.0")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", credentials))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
