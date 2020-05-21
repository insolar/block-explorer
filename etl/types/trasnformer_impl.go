// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package types

import (
	"github.com/insolar/block-explorer/utils"
	"github.com/insolar/insolar/insolar"
	ins_record "github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
)

type Transform interface {
	Transform() (*JetDrop, error)
}

const (
	// delta between pulses
	pulseDelta uint16 = 10
)

// Transform transforms thr row JetDrops to canonical JetDrops
func (jd *PlatformJetDrops) Transform() ([]*JetDrop, error) {
	// if no records per pulse
	if len(jd.Records) == 0 {
		return make([]*JetDrop, 0), nil
	}

	pulseData, err := getPulseData(jd.Records[0])
	if err != nil {
		return nil, err
	}

	m, err := getRecords(jd.Records)
	if err != nil {
		return nil, err
	}

	result := make([]*JetDrop, 0)
	for jetID, records := range m {
		sections := make([]Section, 0)
		prefix := jetID.Prefix()
		mainSection := &MainSection{
			Start: DropStart{
				PulseData:           pulseData,
				JetDropPrefix:       prefix,
				JetDropPrefixLength: uint(len(prefix)),
			},
			DropContinue: DropContinue{},
			Records:      records,
		}

		hash, err := hash(records)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot calculate JetDrop hash")
		}
		localJetDrop := &JetDrop{
			MainSection: mainSection,
			Sections:    sections,
			RawData:     hash,
		}
		result = append(result, localJetDrop)
	}

	return result, nil
}

// hash calculate hash from record's hash
func hash(r []Record) ([]byte, error) {
	l := len(r)
	data := make([][]byte, l)
	for i, record := range r {
		data[i] = record.Hash
	}
	return utils.Hash(data)
}

func getPulseData(rec *exporter.Record) (Pulse, error) {
	r := rec.GetRecord()
	pulse := r.ID.Pulse()
	time, err := pulse.AsApproximateTime()
	if err != nil {
		return Pulse{}, errors.Wrapf(err, "could not get pulse ApproximateTime")
	}
	return Pulse{
		PulseNo:        int(pulse.AsUint32()),
		EpochPulseNo:   int(pulse.AsEpoch()),
		PulseTimestamp: time,
		NextPulseDelta: int(pulse.Next(pulseDelta)),
		PrevPulseDelta: int(pulse.Prev(pulseDelta)),
	}, nil
}

func getRecords(records []*exporter.Record) (map[insolar.JetID][]Record, error) {
	// map need to collect records by JetID
	res := make(map[insolar.JetID][]Record)
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

func transferToCanonicalRecord(r *exporter.Record) (Record, error) {
	var (
		recordType          RecordType
		ref                 Reference
		objectReference     Reference
		prototypeReference  Reference = make([]byte, 0)
		prevRecordReference Reference = make([]byte, 0)
		recordPayload       []byte    = make([]byte, 0)
		hash                []byte
		rawData             []byte
		order               uint32
	)

	ref = r.Record.ID.Bytes()
	hash = r.Record.ID.Hash()
	objectReference = r.Record.ObjectID.Hash()
	dAtA, err := r.Marshal()
	if err != nil {
		return Record{}, errors.Wrapf(err, "cannot get record raw data")
	}
	rawData = dAtA
	order = r.RecordNumber

	virtual := r.GetRecord().Virtual
	switch virtual.Union.(type) {
	case *ins_record.Virtual_Activate:
		recordType = STATE
		activate := virtual.GetActivate()
		prototypeReference = activate.Image.Bytes()
		recordPayload = activate.Memory
		prevRecordReference = activate.PrevStateID().AsBytes()

	case *ins_record.Virtual_Amend:
		recordType = STATE
		amend := virtual.GetAmend()
		prototypeReference = amend.Image.Bytes()
		recordPayload = amend.Memory
		prevRecordReference = amend.PrevState.AsBytes()

	case *ins_record.Virtual_Deactivate:
		recordType = STATE
		deactivate := virtual.GetDeactivate()
		prototypeReference = deactivate.GetImage().AsBytes()
		prevRecordReference = deactivate.PrevState.AsBytes()

	case *ins_record.Virtual_Result:
		recordType = RESULT
		recordPayload = virtual.GetResult().Payload

	case *ins_record.Virtual_IncomingRequest:
		recordType = REQUEST
		request := virtual.GetIncomingRequest()
		object := request.GetObject()
		if object.IsObjectReference() {
			objectReference = object.Bytes()
		}
	case *ins_record.Virtual_OutgoingRequest:
		recordType = REQUEST
		object := virtual.GetOutgoingRequest().GetObject()
		if object.IsObjectReference() {
			objectReference = object.Bytes()
		}
	}

	retRecord := Record{
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
