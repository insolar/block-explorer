// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package transformer

import (
	"context"

	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/utils"
	"github.com/insolar/insolar/insolar"
	ins_record "github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
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
		sections := make([]types.Section, 0)
		prefix := jetID.Prefix()
		mainSection := &types.MainSection{
			Start: types.DropStart{
				PulseData:           pulseData,
				JetDropPrefix:       prefix,
				JetDropPrefixLength: uint(len(prefix)),
			},
			DropContinue: types.DropContinue{},
			Records:      records,
		}

		hash, err := hash(records)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot calculate JetDrop hash")
		}
		localJetDrop := &types.JetDrop{
			MainSection: mainSection,
			Sections:    sections,
			RawData:     hash,
		}
		result = append(result, localJetDrop)
	}

	return result, nil
}

// hash calculate hash from record's hash
func hash(r []types.Record) ([]byte, error) {
	l := len(r)
	data := make([][]byte, l)
	for i, record := range r {
		data[i] = record.Hash
	}
	return utils.Hash(data)
}

func getPulseData(rec *exporter.Record) (types.Pulse, error) {
	r := rec.GetRecord()
	pulse := r.ID.Pulse()
	time, err := pulse.AsApproximateTime()
	if err != nil {
		return types.Pulse{}, errors.Wrapf(err, "could not get pulse ApproximateTime")
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
		prevRecordReference = activate.PrevStateID().AsBytes()

	case *ins_record.Virtual_Amend:
		recordType = types.STATE
		amend := virtual.GetAmend()
		prototypeReference = amend.Image.Bytes()
		recordPayload = amend.Memory
		prevRecordReference = amend.PrevStateID().AsBytes()

	case *ins_record.Virtual_Deactivate:
		recordType = types.STATE
		deactivate := virtual.GetDeactivate()
		prototypeReference = deactivate.GetImage().AsBytes()
		prevRecordReference = deactivate.PrevStateID().AsBytes()

	case *ins_record.Virtual_Result:
		recordType = types.RESULT
		recordPayload = virtual.GetResult().Payload

	case *ins_record.Virtual_IncomingRequest:
		recordType = types.REQUEST
		object := virtual.GetIncomingRequest().GetObject()
		if object.IsObjectReference() {
			objectReference = object.Bytes()
		}
	case *ins_record.Virtual_OutgoingRequest:
		recordType = types.REQUEST
		object := virtual.GetOutgoingRequest().GetObject()
		if object.IsObjectReference() {
			objectReference = object.Bytes()
		}
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
