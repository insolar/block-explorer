package interfaces

import (
	"context"
	"time"

	"github.com/insolar/insolar/ledger/heavy/exporter"

	"github.com/insolar/block-explorer/etl/types"
	"google.golang.org/grpc"

	"github.com/insolar/block-explorer/etl/models"
)

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.Starter -o ./mock -s _mock.go -g
type Starter interface {
	// Start starts the main thread
	Start(ctx context.Context) error
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.Stopper -o ./mock -s _mock.go -g
type Stopper interface {
	// Stops stops the main thread
	Stop(ctx context.Context) error
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.JetDropsExtractor -o ./mock -s _mock.go -g
// JetDropsExtractor represents the main functions of working with Platform
type JetDropsExtractor interface {
	Starter
	Stopper
	// GetJetDrops stores JetDrop data in the main JetDrop channel
	GetJetDrops(ctx context.Context) <-chan *types.PlatformPulseData
	// LoadJetDrops loads JetDrop data between pulse numbers: (fromPulseNumber, toPulseNumber]
	LoadJetDrops(ctx context.Context, fromPulseNumber int64, toPulseNumber int64) error
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.PulseExtractor -o ./mock -s _mock.go -g
// PulseExtractor represents the methods for getting Pulse
type PulseExtractor interface {
	// GetCurrentPulse returns the current Pulse number
	GetCurrentPulse(ctx context.Context) (uint32, error)
	// GetNextFinalizedPulse requests full pulse info
	GetNextFinalizedPulse(ctx context.Context, p int64) (*exporter.FullPulse, error)
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.ConnectionManager -o ./mock -s _mock.go -g
// ConnectionManager represents management of connection to Platform
type ConnectionManager interface {
	Starter
	Stopper
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.Transformer -o ./mock -s _mock.go -g
// Transformer represents a transformation raw data from the Platform to conan type
type Transformer interface {
	Starter
	Stopper
	// GetJetDropsChannel returns the channel where canonical data will be stored
	GetJetDropsChannel() <-chan *types.JetDrop
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.Client -o ./mock -s _mock.go -g
// Client represents a connection to the Platform
type Client interface {
	// GetGRPCConn returns a configured GRPC connection
	GetGRPCConn() *grpc.ClientConn
	// NotifyShutdown sends a notification to the channel when connection has errors
	NotifyShutdown(ctx context.Context, stopChannel chan<- struct{}, waitForStateChange time.Duration)
}

// Processor saves canonical data to database
type Processor interface {
	Starter
	Stopper
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.Controller -o ./mock -s _mock.go -g
// Controller tracks drops integrity and makes calls to reload data
type Controller interface {
	Starter
	Stopper
	// Save information about saved jetdrops
	SetJetDropData(pulse types.Pulse, jetID string)
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.StorageSetter -o ./mock -s _mock.go -g
// StorageSetter saves data to database
type StorageSetter interface {
	// SaveJetDropData saves provided jetDrop and records to db in one transaction.
	// increase jet_drop_amount and record_amount
	SaveJetDropData(jetDrop models.JetDrop, records []models.Record, pulseNumber int64) error
	// SavePulse saves provided pulse to db.
	SavePulse(pulse models.Pulse) error
	// CompletePulse update pulse with provided number to completeness in db.
	CompletePulse(pulseNumber int64) error
	// SequencePulse update pulse with provided number to sequential in db.
	SequencePulse(pulseNumber int64) error
}

// StorageAPIFetcher gets data from database
type StorageAPIFetcher interface {
	// GetRecord returns record with provided reference from db.
	GetRecord(ref models.Reference) (models.Record, error)
	// GetPulse returns pulse with provided pulse number from db.
	GetPulse(pulseNumber int64) (models.Pulse, error)
	// GetPulse returns pulses from db.
	GetPulses(fromPulse *int64, timestampLte, timestampGte, pulseNumberLte, pulseNumberLt, pulseNumberGte, pulseNumberGt *int64, sortByAsc bool, limit, offset int) ([]models.Pulse, int, error)
	// GetJetDropsWithParams returns jetDrops for provided pulse with limit and offset.
	GetJetDropsWithParams(pulse models.Pulse, fromJetDropID *models.JetDropID, limit int, offset int) ([]models.JetDrop, int, error)
	// GetJetDropByID returns JetDrop by JetDropID, with slices of previous jetDrops and next jetDrops.
	GetJetDropByID(id models.JetDropID) (models.JetDrop, []models.JetDrop, []models.JetDrop, error)
	// GetJetDropsByJetID returns jetDrops for provided jetID sorting and filtering by pulseNumber.
	GetJetDropsByJetID(jetID string, pulseNumberLte, pulseNumberLt, pulseNumberGte, pulseNumberGt *int64, limit int, sortByPnAsc bool) ([]models.JetDrop, int, error)
	// GetLifeline returns records for provided object reference, ordered by desc by pulse number and order fields.
	GetLifeline(objRef []byte, fromIndex *string, pulseNumberLt, pulseNumberGt, timestampLte, timestampGte *int64, limit, offset int, sortByIndexAsc bool) ([]models.Record, int, error)
	// GetRecordsByJetDrop returns records for provided jet drop, ordered by order field.
	GetRecordsByJetDrop(jetDropID models.JetDropID, fromIndex, recordType *string, limit, offset int) ([]models.Record, int, error)
	// GetNextSavedPulse returns first pulse with pulse number bigger then fromPulseNumber from db.
	GetNextSavedPulse(fromPulseNumber models.Pulse, completedOnly bool) (models.Pulse, error)
	// GetJetDrops returns jetDrops for provided pulse from db.
	GetJetDrops(pulse models.Pulse) ([]models.JetDrop, error)
}

type StorageFetcher interface {
	// GetIncompletePulses returns pulses that are not complete from db.
	GetIncompletePulses() ([]models.Pulse, error)
	// GetSequentialPulse returns max pulse that have is_sequential as true from db.
	GetSequentialPulse() (models.Pulse, error)
	// GetPulseByPrev returns pulse with provided prev pulse number from db.
	GetPulseByPrev(prevPulse models.Pulse) (models.Pulse, error)
	// GetNextSavedPulse returns first pulse with pulse number bigger then fromPulseNumber from db.
	GetNextSavedPulse(fromPulseNumber models.Pulse, completedOnly bool) (models.Pulse, error)
	// GetJetDrops returns jetDrops for provided pulse from db.
	GetJetDrops(pulse models.Pulse) ([]models.JetDrop, error)
}

// StorageExporterFetcher represents the methods for exporter-api
type StorageExporterFetcher interface {
	GetNextCompletePulseFilterByPrototypeReference(prevPulse int64, prototypes [][]byte) (models.Pulse, error)
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.Storage -o ./mock -s _mock.go -g
// Storage manipulates data in database
type Storage interface {
	StorageSetter
	StorageFetcher
	StorageExporterFetcher
}
