# Generic Block Explorer

This repository contains a backend and API implementation for the Generic Block Explorer (GBE) application.

The GBE's backend pulls records from [Insolar](https://github.com/insolar/insolar)'s API and stores them in a relational format. Records are Insolar's smallest data unit that underpins all data produced by smart contracts. So, GBE does not recognize any entities at business level (users, transactions), only the ones at logic level (requests, execution results, object states).

GBE provides an API of its own, optimized for data visualization. The [GBE's frontend](https://github.com/insolar/frontend-block-explorer) (developed separately) then puts up a web face that allows the user to inspect the data in a friendly way.

## Install

To manually deploy the GBE's backend, first install:
 
- [Go 1.14](https://golang.org/dl/)
- [PostgeSQL 12+](https://www.postgresql.org/download/)

Or you can use Docker of the latest version if you prefer a containerized deployment.

## Deploy

Deploy with docker-compose:

```
make config && docker-compose up -d
```

Alternatively, deploy manually by completing the following steps:

1. Resolve dependencies and build binaries:

   ```
   make all
   ```

2. Migrate the data from Insolar and configure the deployment:

   ```
   make config migrate
   ```

   **Note**: You can change the default configuration in `.artifacts/*.yaml` files. For example, connection parameters between Insolar and the backend's database.

3. Start the backend and API:

   ```
   ./bin/block-explorer --config=.artifacts/block-explorer.yaml
   ./bin/api --config=.artifacts/api.yaml
   ```

## Monitor metrics

Start the metrics server:

```
cd ./scripts/monitor && docker-compose up -d
```

Open Grafana at http://localhost:3000/ with `admin:pass` default credentials.

Open Prometheus at http://localhost:9090/.

## Learn what's under the hood

GBE consists of the following components:

**Extractor**. Fetches data from Insolar and sends the data to the transformer.

**Transformer**. Receives data from the extractor, transforms the data into GBE entities (records, lifelines, pulses, jets, and jet drops), and sends them to the processor.

**Processor**. Processes the entities pulse-by-pulse and stores them in the storage (an internal component).

**Controller**. Searches for data missing in GBE's database—pulses and their records. If found, the controller asks the extractor to re-request the missing data.

## Run tests

You can run several kinds of tests against the backend: unit, integration, load tests, and benchmarks.

### Unit and integration tests

To run unit tests, say:

```
make unit
```

Integration tests require Docker. Make sure you run Docker and say the following to run the tests:

```
make integration
make test-heavy-mock-integration
```

### Load tests

To run load tests, see their [README](load/README.md).

### Benchmarks

To run benchmarks, say:

```
make bench
make bench-integration
```

To compare benchmarks between the latest commit and your newest one, follow these steps:

1. Install [cob](https://github.com/knqyf263/cob):

   ```
   curl -sfL https://raw.githubusercontent.com/knqyf263/cob/master/install.sh | sudo sh -s -- -b /usr/local/bin
   ```

2. (**Required**) Commit your changes.

3. Compare benchmarks between the latest two commits by running:

   ```
   make bench-compare
   make bench-compare-integration
   ```
