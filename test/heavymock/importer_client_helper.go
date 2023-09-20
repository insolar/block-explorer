package heavymock

import (
	"context"
	"io"

	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
)

func ImportRecords(client HeavymockImporterClient, records []*exporter.Record) error {
	stream, err := client.Import(context.Background())
	if err != nil {
		return err
	}

	for _, record := range records {
		if record == nil {
			return errors.New("unable to send nil record")
		}
		if err := stream.Send(record); err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "Error sending to stream")
		}
	}

	reply, err := stream.CloseAndRecv()
	if reply == nil || !reply.Ok {
		return errors.Wrap(err, "Error in Importer reply")
	}
	return nil
}

func ReceiveRecords(client exporter.RecordExporterClient, request *exporter.GetRecords) ([]*exporter.Record, error) {
	stream, err := client.Export(context.Background(), request)
	if err != nil {
		return nil, errors.Wrap(err, "Error when sending client request")
	}

	records := make([]*exporter.Record, 0)
	for {
		record, err := stream.Recv()
		if err == io.EOF {
			break
		}
		records = append(records, record)
		if err != nil {
			return nil, errors.Wrap(err, "Unexpected error")
		}
	}
	return records, nil
}
