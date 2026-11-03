# MQTT to Prometheus

This stack runs [mqtt2prometheus](https://github.com/hikhvar/mqtt2prometheus) alongside the health exporters and subscribes to the Qingping Air Monitor Lite topics published by the local Mosquitto broker. Its metrics are exposed on port `9641` for the Prometheus Docker service discovery already used by this stack.

## Customize the exporter

Edit `stacks/health-exporters/mqtt2prometheus/config.yaml` before deploying the stack. The directory is mounted read-only at `/config` in the exporter. After changing the file, redeploy or restart the `mqtt2prometheus` service so it loads the new configuration.

The config currently maps the CGDN1 JSON payload fields to Prometheus gauges. The topic uses a wildcard, so the same mapping applies to any Qingping device publishing under `qingping/<device-id>/up`; the device ID is exported as the `sensor` label.

`mqtt2prometheus` accepts one config file per process and does not discover or merge multiple config files in a directory. Put all metric mappings for this exporter in `config.yaml`. To use a different config, replace that file or run another exporter service with its own config file and metrics port.
