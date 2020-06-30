// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package transformer

import (
	"bytes"
	"context"

	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/ugorji/go/codec"
	"golang.org/x/crypto/sha3"

	"github.com/insolar/insolar/insolar"
	ins_record "github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"

	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/instrumentation/belogger"
)

const (
	// delta between pulses
	pulseDelta uint16 = 10
)

// Transform transforms thr row JetDrops to canonical JetDrops
func Transform(ctx context.Context, jd *types.PlatformJetDrops) ([]*types.JetDrop, error) {
	// if no records per pulse
	if len(jd.Records) == 0 {
		return make([]*types.JetDrop, 0), nil
	}

	pulseData, err := getPulseData(jd.Records[0])
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get pulse data from record")
	}

	m, err := getRecords(jd.Records)
	if err != nil {
		return nil, err
	}

	result := make([]*types.JetDrop, 0)
	for jetID, records := range m {
		localJetDrop, err := getJetDrop(ctx, jetID, records, pulseData)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot create jet drop for jetID %s", jetID.DebugString())
		}
		if localJetDrop == nil {
			continue
		}
		result = append(result, localJetDrop)
	}

	return result, nil
}

func getJetDrop(ctx context.Context, jetID insolar.JetID, records []types.Record, pulseData types.Pulse) (*types.JetDrop, error) {
	sections := make([]types.Section, 0)
	var prefix string
	if jetID.IsValid() {
		prefix = converter.JetIDToString(jetID)
	}

	records, err := sortRecords(records)
	if err != nil {
		belogger.FromContext(ctx).Errorf("cannot sort records in JetDrop %s, error: %s", jetID.DebugString(), err.Error())
		return nil, nil
	}

	mainSection := &types.MainSection{
		Start: types.DropStart{
			PulseData:           pulseData,
			JetDropPrefix:       prefix,
			JetDropPrefixLength: uint(len(prefix)),
		},
		DropContinue: types.DropContinue{},
		Records:      records,
	}

	localJetDrop := types.JetDrop{
		MainSection: mainSection,
		Sections:    sections,
	}

	rawData, err := serialize(localJetDrop.MainSection)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot calculate JetDrop hash")
	}
	localJetDrop.RawData = rawData
	hash := sha3.Sum224(rawData)
	localJetDrop.Hash = hash[:]
	return &localJetDrop, nil
}

func serialize(o interface{}) ([]byte, error) {
	ch := new(codec.CborHandle)
	var data []byte
	err := codec.NewEncoderBytes(&data, ch).Encode(o)
	return data, errors.Wrap(err, "[ Serialize ]")
}

// sortRecords sorts state records for every object in order of change
func sortRecords(records []types.Record) ([]types.Record, error) {
	lenBefore := len(records)
	recordsByObjAndPrevRef, recordsByObjAndRef, sortedRecords := initRecordsMapsByObj(records)
	for objRef, recordsByRef := range recordsByObjAndRef {
		// if there is only one record, we don't need to sort
		if len(recordsByRef) == 1 {
			for _, r := range recordsByRef {
				sortedRecords = append(sortedRecords, r)
			}
			continue
		}
		var headRecord *types.Record
		// finding first record (head), it doesn't refer to any other record
		recordsByPrevRef := recordsByObjAndPrevRef[objRef]
		for _, r := range recordsByPrevRef {
			_, ok := recordsByRef[restoreInsolarID(r.PrevRecordReference)]
			if !ok {
				headRecord = &r // nolint
				break
			}
		}
		if headRecord == nil {
			return nil, errors.Errorf("cannot find head record for object %s", objRef)
		}
		// add records to result array in correct order
		key := restoreInsolarID(headRecord.Ref)
		sortedRecords = append(sortedRecords, *headRecord)
		for i := 1; len(recordsByPrevRef) != i; i++ {
			r, ok := recordsByPrevRef[key]
			if !ok {
				return nil, errors.Errorf("cannot find record with prev record %s, object %s", key, objRef)
			}
			sortedRecords = append(sortedRecords, r)
			key = restoreInsolarID(r.Ref)
		}
	}
	lenAfter := len(sortedRecords)
	if lenBefore != lenAfter {
		return nil, errors.Errorf("Number of records before sorting (%d) changes after (%d)", lenBefore, lenAfter)
	}

	return sortedRecords, nil
}

func initRecordsMapsByObj(records []types.Record) (
	byPrevRef map[string]map[string]types.Record,
	byRef map[string]map[string]types.Record,
	notState []types.Record,
) {
	var notStateRecords []types.Record
	recordsByObjAndPrevRef := map[string]map[string]types.Record{}
	recordsByObjAndRef := map[string]map[string]types.Record{}
	for _, r := range records {
		if r.Type != types.STATE {
			notStateRecords = append(notStateRecords, r)
			continue
		}
		if recordsByObjAndRef[restoreInsolarID(r.ObjectReference)] == nil {
			recordsByObjAndRef[restoreInsolarID(r.ObjectReference)] = map[string]types.Record{}
			recordsByObjAndPrevRef[restoreInsolarID(r.ObjectReference)] = map[string]types.Record{}
		}
		recordsByObjAndRef[restoreInsolarID(r.ObjectReference)][restoreInsolarID(r.Ref)] = r
		recordsByObjAndPrevRef[restoreInsolarID(r.ObjectReference)][restoreInsolarID(r.PrevRecordReference)] = r
	}
	return recordsByObjAndPrevRef, recordsByObjAndRef, notStateRecords
}

func restoreInsolarID(b []byte) string {
	emptyByte := make([]byte, len(b))
	if bytes.Equal(b, []byte{}) || bytes.Contains(b, emptyByte) {
		b = nil
	}
	return insolar.NewIDFromBytes(b).String()
}

func getPulseData(rec *exporter.Record) (types.Pulse, error) {
	r := rec.GetRecord()
	pulse := r.ID.Pulse()
	time, err := pulse.AsApproximateTime()
	if err != nil {
		return types.Pulse{}, errors.Wrapf(err, "could not get pulse ApproximateTime. pulse: %v", pulse.String())
	}
	return types.Pulse{
		PulseNo:        int(pulse.AsUint32()),
		EpochPulseNo:   int(pulse.AsEpoch()),
		PulseTimestamp: time.Unix(),
		NextPulseDelta: int(pulseDelta),
		PrevPulseDelta: int(pulseDelta),
	}, nil
}

func getRecords(records []*exporter.Record) (map[insolar.JetID][]types.Record, error) {
	// map need to collect records by JetID
	res := make(map[insolar.JetID][]types.Record)
	for _, r := range records {
		record, err := transferToCanonicalRecord(r)
		if err != nil {
			if err == UnsupportedRecordTypeError {
				// just skip this records
				continue
			}
			return res, err
		}
		// collect records with some jetID
		res[r.Record.JetID] = append(res[r.Record.JetID], record)
	}

	return res, nil
}

func transferToCanonicalRecord(r *exporter.Record) (types.Record, error) {
	var (
		recordType          types.RecordType
		ref                 types.Reference
		objectReference     types.Reference
		prototypeReference  types.Reference = make([]byte, 0)
		prevRecordReference types.Reference = make([]byte, 0)
		recordPayload       []byte          = make([]byte, 0)
		hash                []byte
		rawData             []byte
		order               uint32
	)

	ref = r.Record.ID.Bytes()
	hash = r.Record.ID.Hash()
	objectReference = r.Record.ObjectID.Bytes()
	data, err := r.Marshal()
	if err != nil {
		return types.Record{}, errors.Wrapf(err, "cannot get record raw data")
	}
	rawData = data
	order = r.RecordNumber

	virtual := r.GetRecord().Virtual
	switch virtual.Union.(type) {
	case *ins_record.Virtual_Activate:
		recordType = types.STATE
		activate := virtual.GetActivate()
		prototypeReference = activate.Image.Bytes()
		recordPayload = activate.Memory
		objectReference = activate.Request.GetLocal().Bytes()

	case *ins_record.Virtual_Amend:
		recordType = types.STATE
		amend := virtual.GetAmend()
		prototypeReference = amend.Image.Bytes()
		recordPayload = amend.Memory
		prevRecordReference = amend.PrevStateID().Bytes()
		if bytes.Equal(objectReference, insolar.NewEmptyID().Bytes()) {
			objectReference = amend.Request.GetLocal().Bytes()
		}

	case *ins_record.Virtual_Deactivate:
		recordType = types.STATE
		deactivate := virtual.GetDeactivate()
		prototypeReference = deactivate.GetImage().Bytes()
		prevRecordReference = deactivate.PrevStateID().Bytes()

	case *ins_record.Virtual_Result:
		recordType = types.RESULT
		recordPayload = virtual.GetResult().Payload

	case *ins_record.Virtual_IncomingRequest:
		recordType = types.REQUEST
		object := virtual.GetIncomingRequest().GetObject()
		if object != nil && object.IsObjectReference() {
			objectReference = object.GetLocal().Bytes()
		}
	case *ins_record.Virtual_OutgoingRequest:
		recordType = types.REQUEST
		object := virtual.GetOutgoingRequest().GetObject()
		if object != nil && object.IsObjectReference() {
			objectReference = object.GetLocal().Bytes()
		}
	default:
		// skip unnecessary record
		return types.Record{}, UnsupportedRecordTypeError
	}

	retRecord := types.Record{
		Type:                recordType,
		Ref:                 ref,
		ObjectReference:     objectReference,
		PrototypeReference:  prototypeReference,
		PrevRecordReference: prevRecordReference,
		RecordPayload:       recordPayload,
		Hash:                hash,
		RawData:             rawData,
		Order:               order,
	}

	return retRecord, nil
}
