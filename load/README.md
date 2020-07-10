#### Load tests

##### Local run
Start BE API + loadtest migrations data
```
docker-compose -f be-api-compose.yaml up
```

Run suite
```
go run load/cmd/load/main.go -gen_config load/gen_cfg/generator_block_explorer.yaml -config load/run_configs/${SUITE_NAME}.yaml
```
see results.csv/runners.log after test