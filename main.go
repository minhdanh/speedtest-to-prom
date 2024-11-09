package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	pb "github.com/minhdanh/speedtest-to-prom/internal/prometheus"
	prometheus "github.com/minhdanh/speedtest-to-prom/pkg/prometheus"
)

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
	metrics := map[string][]prometheus.MetricValue{
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
	promUrl := os.Getenv("PROM_URL")
	promUsername := os.Getenv("PROM_USERNAME")
	promPassword := os.Getenv("PROM_PASSWORD")
	if promUsername == "" || promPassword == "" {
		log.Fatal("PROM_USERNAME and PROM_PASSWORD environment variables must be set")
	}

	err = prometheus.PushPrometheusMetrics(promUrl, promUsername, promPassword, metrics, labels)
	if err != nil {
		log.Fatalf("Error pushing metrics: %v", err)
	}

	log.Println("Metrics pushed successfully")
}
