# Peggo exporter

Prometheus exporter for [peggo](https://github.com/umee-network/peggo) metrics.

This project follows the Prometheus community exporter pattern.

## Building and running

```
git clone https://github.com/stakefish/peggo-exporter.git
cd peggo-exporter
make build
./bin/peggo-exporter <flags>
```

To build the Docker image:

```
docker build -t peggo-exporter .

# for macOS docker desktop
docker buildx build --platform=linux/amd64 -t peggo-exporter .
```

### Flags

* `help` Show context-sensitive help (also try --help-long and --help-man).
* `peggo.peggo-rest-rpc` Peggo REST API URL. Default is `http://localhost:1317`.
* `peggo.cosmos-orchestrator-address` Cosmos orchestrator address. Default is empty string.
* `peggo.timeout` Peggo connect timeout. Default is `5s`.
* `web.listen-address` Address to listen on for web interface and telemetry. Default is `:5566`.
* `web.telemetry-path` Path under which to expose metrics. Default is `/metrics`.
* `version` Show application version.
* `log.level` Set logging level: one of `debug`, `info`, `warn`, `error`.
* `log.format` Set the log format: one of `logfmt`, `json`.
* `web.config.file` Configuration file to use TLS and/or basic authentication. The format of the file is described [in the exporter-toolkit repository](https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md).

### Environment Variables

* `PEGGO_REST_RPC` Peggo REST API URL. Default is `http://localhost:1317`.
* `COSMOS_ORCHESTRATOR_ADDRESS` Cosmos orchestrator address. Default is empty string.
* `PEGGO_TIMEOUT` Peggo connect timeout. Default is `5s`.
* `EXPORTER_WEB_LISTEN_ADDRESS` Address to listen on for web interface and telemetry. Default is `:5566`.
* `EXPORTER_WEB_TELEMETRY_PATH` Path under which to expose metrics. Default is `/metrics`.

## Mechanism
1. Get list of validators
2. Get orch address for each validator
3. Get event nonce for each orch address
4. Compare my own orch's event nonce with the others

Each operator will decide when to trigger an alarm based on these values.

A good baseline would be to trigger an alarm if own event nonce is not
moving (while others do) in the last 10 minutes.

Endpoints used:

```
http://localhost:1317/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED

http://localhost:1317/gravity/v1beta/query_delegate_keys_by_validator?validator_address=umeevaloper1tsd7h4erlx9wajg353dwjc56lrvlcnmeghnmk0

http://localhost:1317/gravity/v1beta/oracle/eventnonce/umee1avyd3vh2lfjs2q28h4nj9hqevcxyacfj7m3pz3
```
