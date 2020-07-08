# block-explorer

#### Load test setup
for debug run grafana/graphite locally
```
docker run -d -p 8181:80 -p 8125:8125/udp -p 8126:8126 --publish=2003:2003 --name kamon-grafana-dashboard kamon/grafana_graphite
```
build and run tests
```
docker-compose -f be-api-compose.yaml up
loadcli -gen_config load/gen_cfg/generator_block_explorer.yaml b darwin
./load_suite -gen_config load/gen_cfg/generator_block_explorer.yaml -config load/run_configs/get_jet_drop_by_id.yaml
```
if new tests were added, generate dashboards for handles
```
loadcli -gen_config load/gen_cfg/generator_block_explorer.yaml d
```

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
