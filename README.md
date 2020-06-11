# block-explorer

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
