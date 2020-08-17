# Generic Block Explorer
This is backend and API for block-explorer application  
It was created to for easily visualise and inspect data from https://github.com/insolar/insolar  
The main idea is to get records stored in insolar using it's API and save it in relational-style format  
It doesn't know anything about smart contracts  

## Requirements
 - go 1.14
 - PostgeSQL 12+

## How to start
#### Resolve dependencies and build
```
make all
```
#### Create tables in DB
```
make config migrate
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
make config && docker-compose up -d
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
