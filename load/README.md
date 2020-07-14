#### Load tests

##### Local run
Start BE API + loadtest migrations data
```
docker-compose -f be-api-compose-loadtest.yaml build
docker-compose -f be-api-compose-loadtest.yaml up
```

Run suite
```
go run load/cmd/load/main.go -gen_config load/gen_cfg/generator_block_explorer.yaml -config load/run_configs/${SUITE_NAME}.yaml
```
see results.csv/runners.log after test

Down compose to clear data in pg
```
docker-compose -f be-api-compose-loadtest.yaml down
```