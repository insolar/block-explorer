module github.com/insolar/block-explorer

go 1.14

require (
	github.com/antihax/optional v1.0.0
	github.com/deepmap/oapi-codegen v1.3.8 // indirect
	github.com/fortytw2/leaktest v1.3.0
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/gojuno/minimock/v3 v3.0.6
	github.com/google/gofuzz v1.0.0
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/insolar/assured-ledger/ledger-core/v2 v2.0.0-20200512113104-4973d6ba44e9
	github.com/insolar/insconfig v0.0.0-20200430133349-77f6f1624abf
	github.com/insolar/insolar v1.5.2
	github.com/insolar/spec-insolar-block-explorer-api v1.2.0
	github.com/jinzhu/gorm v1.9.12
	github.com/kelindar/binary v1.0.9
	github.com/kr/pretty v0.2.0 // indirect
	github.com/labstack/echo/v4 v4.1.16
	github.com/mitchellh/mapstructure v1.3.2 // indirect
	github.com/ory/dockertest/v3 v3.5.2
	github.com/pelletier/go-toml v1.8.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/rs/zerolog v1.15.0
	github.com/spf13/cast v1.3.1 // indirect
	github.com/stretchr/testify v1.5.1
	go.opencensus.io v0.22.1
	golang.org/x/crypto v0.0.0-20200510223506-06a226fb4e37
	golang.org/x/exp v0.0.0-20190510132918-efd6b22b2522 // indirect
	golang.org/x/sys v0.0.0-20200602225109-6fdc65e7d980 // indirect
	google.golang.org/appengine v1.6.1 // indirect
	google.golang.org/grpc v1.22.0
	gopkg.in/gormigrate.v1 v1.6.0
	gopkg.in/ini.v1 v1.57.0 // indirect
	gotest.tools v2.2.0+incompatible // indirect
)

replace github.com/ugorji/go v1.1.4 => github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43
