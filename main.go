package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/snappy"
	pb "github.com/minhdanh/speedtest-to-prom/internal/prometheus" // Replace with your actual module name
	"google.golang.org/protobuf/proto"
)

// SpeedTestResult represents the JSON structure
// SpeedTestResult represents the JSON structure
type SpeedTestResult struct {
	Ping struct {
		Jitter  float64 `json:"jitter"`
		Latency float64 `json:"latency"`
		Low     float64 `json:"low"`
		High    float64 `json:"high"`
	} `json:"ping"`
	Download struct {
		Bandwidth int64 `json:"bandwidth"`
		Bytes     int64 `json:"bytes"`
		Elapsed   int64 `json:"elapsed"`
		Latency   struct {
			Iqm    float64 `json:"iqm"`
			Low    float64 `json:"low"`
			High   float64 `json:"high"`
			Jitter float64 `json:"jitter"`
		} `json:"latency"`
	} `json:"download"`
	Upload struct {
		Bandwidth int64 `json:"bandwidth"`
		Bytes     int64 `json:"bytes"`
		Elapsed   int64 `json:"elapsed"`
		Latency   struct {
			Iqm    float64 `json:"iqm"`
			Low    float64 `json:"low"`
			High   float64 `json:"high"`
			Jitter float64 `json:"jitter"`
		} `json:"latency"`
	} `json:"upload"`
	PacketLoss float64 `json:"packetLoss"`
	ISP        string  `json:"isp"`
	Server     struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Location string `json:"location"`
	} `json:"server"`
	Result struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"result"`
}

// parseLabels parses label string in format "key1=value1,key2=value2"
func parseLabels(labelStr string) ([]*pb.Label, error) {
	var labels []*pb.Label
	if labelStr == "" {
		return labels, nil
	}

	pairs := strings.Split(labelStr, ",")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid label format: %s", pair)
		}
		labels = append(labels, &pb.Label{
			Name:  strings.TrimSpace(kv[0]),
			Value: strings.TrimSpace(kv[1]),
		})
	}
	return labels, nil
}

func pushPrometheusMetrics(promUsername, promPassword string, metrics map[string][]MetricValue, labels []*pb.Label) error {
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
	prometheusUrl := os.Getenv("PROM_URL")
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

// MetricValue represents a metric value with its labels
type MetricValue struct {
	Value  float64
	Labels map[string]string
}

func main() {
	// Define command-line flags
	labelsFlag := flag.String("labels", "", "Additional labels in format 'key1=value1,key2=value2'")
	flag.Parse()

	// Read JSON from stdin
	reader := bufio.NewReader(os.Stdin)
	var input bytes.Buffer

	_, err := io.Copy(&input, reader)
	if err != nil {
		log.Fatalf("Error reading input: %v", err)
	}

	var result SpeedTestResult
	if err := json.Unmarshal(input.Bytes(), &result); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	// Parse additional labels
	additionalLabels, err := parseLabels(*labelsFlag)
	if err != nil {
		log.Fatalf("Error parsing labels: %v", err)
	}

	// Combine default and additional labels
	labels := []*pb.Label{
		{Name: "isp", Value: result.ISP},
		{Name: "server_id", Value: strconv.Itoa(result.Server.ID)},
		{Name: "server_name", Value: result.Server.Name},
		{Name: "result_id", Value: result.Result.ID},
		{Name: "result_url", Value: result.Result.URL},
	}
	labels = append(labels, additionalLabels...)

	// Create metrics map with labels
	metrics := map[string][]MetricValue{
		"speedtest_ping_latency": {
			{
				Value: result.Ping.Latency,
				Labels: map[string]string{
					"jitter": fmt.Sprintf("%.2f", result.Ping.Jitter),
					"low":    fmt.Sprintf("%.2f", result.Ping.Low),
					"high":   fmt.Sprintf("%.2f", result.Ping.High),
				},
			},
		},
		"speedtest_download_bandwidth": {
			{
				Value: float64(result.Download.Bandwidth),
				Labels: map[string]string{
					"bytes":   fmt.Sprintf("%d", result.Download.Bytes),
					"elapsed": fmt.Sprintf("%d", result.Download.Elapsed),
				},
			},
		},
		"speedtest_download_latency": {
			{
				Value: result.Download.Latency.Iqm,
				Labels: map[string]string{
					"jitter": fmt.Sprintf("%.2f", result.Download.Latency.Jitter),
					"low":    fmt.Sprintf("%.2f", result.Download.Latency.Low),
					"high":   fmt.Sprintf("%.2f", result.Download.Latency.High),
					"iqm":    fmt.Sprintf("%.2f", result.Download.Latency.Iqm),
				},
			},
		},
		"speedtest_upload_bandwidth": {
			{
				Value: float64(result.Upload.Bandwidth),
				Labels: map[string]string{
					"bytes":   fmt.Sprintf("%d", result.Upload.Bytes),
					"elapsed": fmt.Sprintf("%d", result.Upload.Elapsed),
				},
			},
		},
		"speedtest_upload_latency": {
			{
				Value: result.Upload.Latency.Iqm,
				Labels: map[string]string{
					"jitter": fmt.Sprintf("%.2f", result.Upload.Latency.Jitter),
					"low":    fmt.Sprintf("%.2f", result.Upload.Latency.Low),
					"high":   fmt.Sprintf("%.2f", result.Upload.Latency.High),
					"iqm":    fmt.Sprintf("%.2f", result.Upload.Latency.Iqm),
				},
			},
		},
		"speedtest_packet_loss": {
			{
				Value:  result.PacketLoss,
				Labels: map[string]string{},
			},
		},
	}

	// Get credentials from environment variables
	promUsername := os.Getenv("PROM_USERNAME")
	promPassword := os.Getenv("PROM_PASSWORD")
	if promUsername == "" || promPassword == "" {
		log.Fatal("PROM_USERNAME and PROM_PASSWORD environment variables must be set")
	}

	err = pushPrometheusMetrics(promUsername, promPassword, metrics, labels)
	if err != nil {
		log.Fatalf("Error pushing metrics: %v", err)
	}

	log.Println("Metrics pushed successfully")
}
