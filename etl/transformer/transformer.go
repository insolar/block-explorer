// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package transformer

import (
	"context"

	"github.com/insolar/insolar/pulse"

	"github.com/insolar/block-explorer/instrumentation"
	"github.com/insolar/block-explorer/instrumentation/converter"

	"github.com/insolar/insolar/applicationbase/genesisrefs"
	"github.com/insolar/insolar/insolar"
	ins_record "github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"

	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/instrumentation/belogger"

	"github.com/kelindar/binary"
)

// Transform transforms thr row JetDrops to canonical JetDrops
func Transform(ctx context.Context, jd *types.PlatformJetDrops) ([]*types.JetDrop, error) {
	pulseData := getPulseData(jd.Pulse)

	m, err := getRecords(jd)
	if err != nil {
		return nil, err
	}

	log := belogger.FromContext(ctx).WithField("service", "transformer")
	for _, jet := range jd.Pulse.Jets {
		jetid := jet.JetID
		if _, ok := m[jetid]; ok {
			log.Debug("full ", jetid.DebugString())
			continue
		}
		m[jetid] = nil
		log.Debug("empty ", jetid.DebugString())
	}

	result := make([]*types.JetDrop, 0)
	for _, jet := range jd.Pulse.Jets {
		jetid := jet.JetID
		records := m[jetid]
		localJetDrop, err := getJetDrop(ctx, jetid, records, pulseData, jet.Hash, jet.PrevDropHashes)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot create jet drop for jetID %s", jetid.DebugString())
		}
		if localJetDrop == nil {
			continue
		}
		result = append(result, localJetDrop)
	}

	return result, nil
}

func getJetDrop(ctx context.Context, jetID insolar.JetID, records []types.Record, pulseData types.Pulse, hash []byte, prevDropHash [][]byte) (*types.JetDrop, error) {
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
		DropContinue: types.DropContinue{
			PrevDropHash: prevDropHash,
		},
		Records: records,
	}

	localJetDrop := types.JetDrop{
		MainSection: mainSection,
		Sections:    sections,
		Hash:        hash,
	}

	rawData, err := serialize(localJetDrop.MainSection)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot calculate JetDrop hash")
	}
	localJetDrop.RawData = rawData

	return &localJetDrop, nil
}

func serialize(o interface{}) ([]byte, error) {
	return binary.Marshal(o)
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
	if instrumentation.IsEmpty(b) {
		b = nil
	}
	return insolar.NewIDFromBytes(b).String()
}

func getPulseData(pn *exporter.FullPulse) types.Pulse {
	pulse := pn.PulseNumber
	return types.Pulse{
		PulseNo:         int64(pulse.AsUint32()),
		EpochPulseNo:    int64(pulse.AsEpoch()),
		PulseTimestamp:  pn.GetPulseTimestamp(),
		NextPulseNumber: int64(pn.NextPulseNumber.AsUint32()),
		PrevPulseNumber: int64(pn.PrevPulseNumber.AsUint32()),
	}
}

// getRecords - order records to map by jetid
func getRecords(jd *types.PlatformJetDrops) (map[insolar.JetID][]types.Record, error) {
	// map need to collect records by JetID
	res := make(map[insolar.JetID][]types.Record)
	if jd == nil {
		return res, nil
	}

	if len(jd.Records) == 0 && jd.Pulse != nil {
		// we don't have a record but have a pulse
		for _, jet := range jd.Pulse.Jets {
			res[jet.JetID] = nil
		}
		return res, nil
	}

	for _, r := range jd.Records {
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

	// TODO: maybe ne need to check the records jetID's with jd.Pulse.Jets
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
		if r.Record.ID.Pulse() == pulse.MinTimePulse {
			objectReference = activate.Request.GetLocal().Bytes()
		}

	case *ins_record.Virtual_Amend:
		recordType = types.STATE
		amend := virtual.GetAmend()
		prototypeReference = amend.Image.Bytes()
		recordPayload = amend.Memory
		prevRecordReference = amend.PrevStateID().Bytes()
		if r.Record.ID.Pulse() == pulse.MinTimePulse {
			objectReference = amend.Request.GetLocal().Bytes()
		}

	case *ins_record.Virtual_Deactivate:
		recordType = types.STATE
		deactivate := virtual.GetDeactivate()
		prevRecordReference = deactivate.PrevStateID().Bytes()

	case *ins_record.Virtual_Result:
		recordType = types.RESULT
		result := virtual.GetResult()
		recordPayload = result.Payload
		if r.Record.ID.Pulse() == pulse.MinTimePulse {
			objectReference = result.GetObject().Bytes()
		}

	case *ins_record.Virtual_IncomingRequest:
		recordType = types.REQUEST
		incomingRequest := virtual.GetIncomingRequest()
		if r.Record.ID.Pulse() == pulse.MinTimePulse {
			objectReference = genesisrefs.GenesisRef(incomingRequest.Method).GetLocal().Bytes()
		}

	case *ins_record.Virtual_OutgoingRequest:
		recordType = types.REQUEST

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
