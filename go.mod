module github.com/insolar/block-explorer

go 1.14

require (
	github.com/gogo/protobuf v1.3.1
	github.com/gojuno/minimock/v3 v3.0.6
	github.com/insolar/assured-ledger/ledger-core/v2 v2.0.0-20200512113104-4973d6ba44e9
	github.com/insolar/insconfig v0.0.0-20200430133349-77f6f1624abf
	github.com/insolar/insolar v1.5.2
	github.com/jinzhu/gorm v1.9.12
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.15.0
	github.com/stretchr/testify v1.4.0
	go.uber.org/goleak v1.0.0
	google.golang.org/grpc v1.21.0
	gopkg.in/gormigrate.v1 v1.6.0
)

replace github.com/ugorji/go v1.1.4 => github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43
