## speedtest-to-prom

Perform a speedtest and export results to a remote Prometheus server.

### Usage

First install Speedtest CLI is maintained by the Ookla team.
See: https://www.speedtest.net/apps/cli

Then set up a cron job like this:

```
speedtest -f json | ./speedtest-to-prom
```

The following environment variables are required for `speedtest-to-prom` to function correctly:

```
PROM_URL=https://<prometheus hostname>/api/v1/write
PROM_USERNAME=<basic auth username>
PROM_PASSWORD=<basic auth password>
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

### Dashboard

![Dashboard](/grafana-dashboard.png)

### Development

Install protobuf:

```
brew install protobuf
```

Generate code:

```
protoc --go_out=. prometheus.proto
```
