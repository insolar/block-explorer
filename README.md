# Generic Block Explorer

This repository contains a backend and API implementation for the Generic Block Explorer application.

The explorer's backend pulls records from [Insolar](https://github.com/insolar/insolar)'s API and stores them in a relational format. Records are Insolar's smallest data unit that underpins all data produced by smart contracts. So, Generic Block Explorer does not recognize any entities at business level (users, transactions), only the ones at logic level (requests, execution results, object states).

Block Explorer provides an API of its own, optimized for data visualization. The Block Explorer's frontend (developed separately) then puts up a web face that allows the user to inspect the data in a friendly way.

## Install

To manually deploy the Block Explorer backend, first install:
 
- [Go 1.14](https://golang.org/dl/)
- [PostgeSQL 12+](https://www.postgresql.org/download/)

Or you can use Docker of the latest version, if you are fine with a containerized deployment.

## Deploy

To deploy the backend, complete the following steps:

1. Resolve dependencies and build

   ```
   make all
   ```

2. Migrate the data:

   ```
   make migrate
   ```
Change connection params to platform and DB  
All information about config params you can find in `.artifacts/*.yaml` files  

#### Start backend and API
```
./bin/block-explorer --config=.artifacts/block-explorer.yaml
./bin/api --config=.artifacts/api.yaml
```
Frontend located here https://github.com/insolar/frontend-block-explorer

#### with docker-compose
You can set up everything with docker-compose
```
docker-compose up -d
```

## How to start metric server
```
cd ./scripts/monitor && docker-compose up -d
```
Grafana: http://localhost:3000/ admin:pass  
Prometheus http://localhost:9090/ 

## Internal components
#### Extractor
Extractor made for fetching data from platform and send to transformer
#### Transformer
Transformer receives data from extractor, transform it into original GBE entities and send to processor
#### Processor
Processor maintains storing GBE entities to DB. It uses storage
#### Controller
Controller implements logic of searching missing data  
It searches for missing pulses in db and missing records in existing pulse data. If found it asks extractor for re-request data

## Tests
#### Unit/Integration
Integration tests needs docker
```
make unit
make integration
make test-heavy-mock-integration
```

#### Loadtests
See [readme](load/README.md)

#### Benchmarks
```
make bench
make bench-integration
```
To run comparative benchmarks install [cob](https://github.com/knqyf263/cob):
```
curl -sfL https://raw.githubusercontent.com/knqyf263/cob/master/install.sh | sudo sh -s -- -b /usr/local/bin
```
To compare benchmarks between latest two commits one *MUST COMMIT* changes and then run
```
make bench-compare
make bench-compare-integration
```
