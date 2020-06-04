module github.com/insolar/block-explorer

go 1.14

require (
	github.com/deepmap/oapi-codegen v1.3.8
	github.com/gogo/protobuf v1.3.1
	github.com/gojuno/minimock/v3 v3.0.6
	github.com/google/gofuzz v1.0.0
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/insolar/assured-ledger/ledger-core/v2 v2.0.0-20200512113104-4973d6ba44e9
	github.com/insolar/insconfig v0.0.0-20200430133349-77f6f1624abf
	github.com/insolar/insolar v1.5.2
	github.com/insolar/spec-insolar-block-explorer-api v0.0.0-20200604134220-4bb690ad4a35
	github.com/jinzhu/gorm v1.9.12
	github.com/labstack/echo/v4 v4.1.16
	github.com/ory/dockertest/v3 v3.5.2
	github.com/ory/go-acc v0.2.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/rs/zerolog v1.15.0
	github.com/stretchr/testify v1.4.0
	github.com/ugorji/go/codec v0.0.0-20181204163529-d75b2dcb6bc8
	go.opencensus.io v0.22.1
	golang.org/x/crypto v0.0.0-20200221231518-2aa609cf4a9d
	golang.org/x/exp v0.0.0-20190510132918-efd6b22b2522 // indirect
	google.golang.org/appengine v1.6.1 // indirect
	google.golang.org/grpc v1.22.0
	gopkg.in/gormigrate.v1 v1.6.0
	gotest.tools v2.2.0+incompatible // indirect
)

replace github.com/ugorji/go v1.1.4 => github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43
