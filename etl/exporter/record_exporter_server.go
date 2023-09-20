package exporter

type RecordServer struct {
}

func NewRecordServer() *RecordServer {
	return &RecordServer{}
}

func (s *RecordServer) GetRecords(*GetRecordsRequest, RecordExporter_GetRecordsServer) error {
	return nil
}
