### speedtest-to-prom

Perform a speedtest and export results to a remote Prometheus server.

### Install speedtest client

See: https://www.speedtest.net/apps/cli#ubuntu

```
wget https://install.speedtest.net/app/cli/ookla-speedtest-1.2.0-linux-armhf.tgz
```

### How to use

```
speedtest -f json | ./speedtest-to-prom
```

### Sample metrics

| Metric                         | Description                  | Labels                          |
| ------------------------------ | ---------------------------- | ------------------------------- |
| `speedtest_ping_latency`       | Network ping latency         | `high`, `jitter`, `low`,        |
| `speedtest_download_bandwidth` | Download speed measurement   | `bytes`, `elapsed`,             |
| `speedtest_download_latency`   | Latency during download test | `high`, `iqm`, `jitter`, `low`, |
| `speedtest_upload_bandwidth`   | Upload speed measurement     | `bytes`, `elapsed`,             |
| `speedtest_upload_latency`     | Latency during upload test   | `high`, `iqm`, `jitter`, `low`, |
| `speedtest_packet_loss`        | Percentage of packet loss    |                                 |

All metrics include base labels: `isp`, `result_id`, `result_url`, `server_id`, and `server_name`.

### Development

Install protobuf:

```
brew install protobuf
```

Generate code:

```
protoc --go_out=. prometheus.proto
```
