// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package testutils

import (
	"encoding/binary"
	"io"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/jet"
	insrecord "github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/pulse"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/etl/models"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateRequestRecord(pulse insolar.PulseNumber, objectID insolar.ID) *exporter.Record {
	r := GenerateRecordsSilence(1)[0]
	id := gen.IDWithPulse(pulse)
	r.Record.ID = id
	r.Record.ObjectID = objectID
	r.ShouldIterateFrom = nil
	reference := insolar.NewReference(id)
	r.Record.Virtual.Union = &insrecord.Virtual_IncomingRequest{
		IncomingRequest: &insrecord.IncomingRequest{
			Object: reference,
			Method: RandomString(20),
		},
	}
	return r
}

func GenerateVirtualActivateRecord(pulse insolar.PulseNumber, objectID, requestID insolar.ID) (record *exporter.Record) {
	r := GenerateRecordsSilence(1)[0]
	id := gen.IDWithPulse(pulse)
	r.Record.ID = id
	r.Record.ObjectID = objectID
	r.ShouldIterateFrom = nil
	requestRerence := insolar.NewReference(requestID)
	r.Record.Virtual.Union = &insrecord.Virtual_Activate{
		Activate: &insrecord.Activate{
			Image:   gen.Reference(),
			Request: *requestRerence,
		},
	}
	return r
}

func GenerateVirtualRequestRecord(pulse insolar.PulseNumber, objectID insolar.ID) (record *exporter.Record) {
	r := GenerateRecordsSilence(1)[0]
	id := gen.IDWithPulse(pulse)
	r.Record.ID = id
	r.Record.ObjectID = objectID
	r.ShouldIterateFrom = nil
	r.Record.Virtual.Union = &insrecord.Virtual_IncomingRequest{
		IncomingRequest: &insrecord.IncomingRequest{
			Polymorph: int32(1),
		},
	}
	return r
}

func GenerateVirtualResultRecord(pulse insolar.PulseNumber, objectID, requestID insolar.ID) (record *exporter.Record) {
	r := GenerateRecordsSilence(1)[0]
	id := gen.IDWithPulse(pulse)
	r.Record.ID = id
	r.Record.ObjectID = objectID
	r.ShouldIterateFrom = nil
	requestRerence := insolar.NewReference(requestID)
	r.Record.Virtual.Union = &insrecord.Virtual_Result{
		Result: &insrecord.Result{
			Request: *requestRerence,
			Object:  gen.ID(),
		},
	}
	return r
}

func GenerateVirtualAmendRecordsLinkedArray(pulse insolar.PulseNumber, jetID insolar.JetID, objectID, prevStateID insolar.ID, recordsCount int) []*exporter.Record {
	result := make([]*exporter.Record, recordsCount)
	for i := 0; i < recordsCount; i++ {
		r := GenerateVirtualAmendRecord(pulse, objectID, prevStateID)
		r.Record.JetID = jetID
		result[i] = r
		prevStateID = r.Record.ID
	}
	return result
}

func GenerateVirtualAmendRecord(pulse insolar.PulseNumber, objectID, prevStateID insolar.ID) *exporter.Record {
	r := GenerateRecordsSilence(1)[0]
	id := gen.IDWithPulse(pulse)
	r.Record.ID = id
	r.Record.ObjectID = objectID
	r.ShouldIterateFrom = nil
	r.Record.Virtual.Union = &insrecord.Virtual_Amend{
		Amend: &insrecord.Amend{
			Image:     gen.Reference(),
			PrevState: prevStateID,
		},
	}
	return r
}

func GenerateVirtualDeactivateRecord(pulse insolar.PulseNumber, objectID, prevStateID insolar.ID) *exporter.Record {
	r := GenerateRecordsSilence(1)[0]
	id := gen.IDWithPulse(pulse)
	r.Record.ID = id
	r.Record.ObjectID = objectID
	r.ShouldIterateFrom = nil
	r.Record.Virtual.Union = &insrecord.Virtual_Deactivate{
		Deactivate: &insrecord.Deactivate{
			PrevState: prevStateID,
		},
	}
	return r
}

type ObjectLifeline struct {
	StateRecords []RecordsByPulse
	SideRecords  []RecordsByPulse
	ObjID        insolar.ID
}

type RecordsByPulse struct {
	Pn      insolar.PulseNumber
	Records []*exporter.Record
}

func (l *ObjectLifeline) GetAllRecords() []*exporter.Record {
	r := l.GetStateRecords()
	for i := 0; i < len(l.SideRecords); i++ {
		r = append(r, l.SideRecords[i].Records...)
	}
	return r
}

func (l *ObjectLifeline) GetStateRecords() []*exporter.Record {
	r := make([]*exporter.Record, 0)
	for i := 0; i < len(l.StateRecords); i++ {
		r = append(r, l.StateRecords[i].Records...)
	}
	return r
}

func GenerateObjectLifeline(pulseCount, recordsInPulse int) ObjectLifeline {
	objectID := gen.ID()
	var prevState insolar.ID
	stateRecords := make([]RecordsByPulse, pulseCount)
	sideRecords := make([]RecordsByPulse, 1)
	pn := gen.PulseNumber()
	for i := 0; i < pulseCount; i++ {
		jetID := GenerateUniqueJetID()
		pn += 10
		records := make([]*exporter.Record, recordsInPulse)
		var amends []*exporter.Record
		if i == 0 {
			request := GenerateRequestRecord(pn, objectID)
			activate := GenerateVirtualActivateRecord(pn, objectID, request.Record.ID)
			activate.Record.JetID = jetID
			sideRecords[0] = RecordsByPulse{
				Pn:      pn,
				Records: []*exporter.Record{request},
			}
			prevState = activate.Record.ID
			records[0] = activate
			amends = GenerateVirtualAmendRecordsLinkedArray(pn, jetID, objectID, prevState, recordsInPulse-1)
			copy(records[1:], amends)
		} else {
			amends = GenerateVirtualAmendRecordsLinkedArray(pn, jetID, objectID, prevState, recordsInPulse)
			copy(records, amends)
		}
		if len(amends) > 0 {
			prevState = amends[len(amends)-1].Record.ID
		}
		if i == pulseCount-1 && recordsInPulse > 1 {
			prevState = amends[len(amends)-2].Record.ID
			deactivate := GenerateVirtualDeactivateRecord(pn, objectID, prevState)
			deactivate.Record.JetID = jetID
			records[len(records)-1] = deactivate
		}

		stateRecords[i] = RecordsByPulse{
			Pn:      pn,
			Records: records,
		}
	}

	return ObjectLifeline{
		StateRecords: stateRecords,
		SideRecords:  sideRecords,
		ObjID:        objectID,
	}

}

// GenerateRecords returns a function for generating record with error
func GenerateRecords(batchSize int) func() (record *exporter.Record, e error) {
	pn := gen.PulseNumber()
	cnt := 0
	eof := true
	randNum := func() int64 {
		return RandNumberOverRange(100, 500)
	}

	generateRecords := func() (record *exporter.Record, e error) {
		if !eof && cnt%batchSize == 0 {
			eof = true
			return &exporter.Record{}, io.EOF
		}
		cnt++
		eof = false
		return &exporter.Record{
			RecordNumber: uint32(cnt),
			Record: insrecord.Material{
				Polymorph: int32(randNum()),
				Virtual: insrecord.Virtual{
					Polymorph: int32(randNum()),
					Union: &insrecord.Virtual_Amend{
						Amend: &insrecord.Amend{
							Request: gen.Reference(),
						},
					},
					Signature: []byte{0, 1, 2},
				},
				ID:        gen.IDWithPulse(pn),
				ObjectID:  gen.ID(),
				JetID:     GenerateUniqueJetID(),
				Signature: []byte{0, 1, 2},
			},
			ShouldIterateFrom: &pn,
			Polymorph:         uint32(randNum()),
		}, nil
	}

	return generateRecords
}

func GenerateRecordsWithDifferencePulsesSilence(differentPulseSize, recordCount int) []*exporter.Record {
	record := GenerateRecordsWithDifferencePulses(differentPulseSize, recordCount, pulse.MinTimePulse)
	result := make([]*exporter.Record, 0)
	for i := 0; i < differentPulseSize*recordCount; i++ {
		r, err := record()
		if err != nil {
			continue
		}
		result = append(result, r)
	}
	return result
}

// GenerateRecordsFromOneJetSilence returns new generated records from one JetID
func GenerateRecordsFromOneJetSilence(differentPulseSize, recordCount int) []*exporter.Record {
	records := GenerateRecordsWithDifferencePulsesSilence(differentPulseSize, recordCount)
	jetID := GenerateUniqueJetID()
	for _, r := range records {
		r.Record.JetID = jetID
	}
	return records
}

// GenerateRecordsWithDifferencePulses generates records with recordCount for each pulse
func GenerateRecordsWithDifferencePulses(differentPulseSize, recordCount int, startpn int64) func() (record *exporter.Record, e error) {
	var mu = &sync.Mutex{}
	i := 0
	localRecordCount := 0
	var prevRecord *exporter.Record = GenerateRecordsSilence(1)[0]
	prevRecord.Record.ID = *insolar.NewID(insolar.PulseNumber(startpn), nil)
	fn := func() (*exporter.Record, error) {
		mu.Lock()
		defer mu.Unlock()
		if i < differentPulseSize {

			record := GenerateRecordsSilence(1)[0]
			record.ShouldIterateFrom = nil
			if localRecordCount < recordCount {
				localRecordCount++
				record.Record.ID = gen.IDWithPulse(prevRecord.Record.ID.Pulse())
				prevRecord = record
			} else {
				i++
				localRecordCount = 1
				record.Record.ID = gen.IDWithPulse(prevRecord.Record.ID.Pulse() + 10)
				prevRecord = record
			}
			return record, nil
		}
		return nil, io.EOF
	}
	return fn
}

func GenerateRecordInNextPulse(prevPulse insolar.PulseNumber) *exporter.Record {
	r := GenerateRecordsSilence(1)[0]
	nextPn := prevPulse + 10
	newID := gen.IDWithPulse(nextPn)
	r.Record.ID = newID
	r.ShouldIterateFrom = nil
	return r
}

// GenerateRecordsSilence returns new generated records without errors
func GenerateRecordsSilence(count int) []*exporter.Record {
	res := make([]*exporter.Record, count)
	f := GenerateRecords(count)
	for i := 0; i < count; {
		record, err := f()
		if err != nil {
			continue
		}
		res[i] = record
		i++
	}
	return res
}

var uniqueJetID = make(map[uint64]bool)
var mutex = &sync.Mutex{}

func GenerateUniqueJetID() insolar.JetID {
	for {
		jetID := gen.JetID()
		if !isUniqueJetId(jetID) {
			continue
		}
		return jetID
	}
}

func isUniqueJetId(jetID insolar.JetID) bool {
	id := binary.BigEndian.Uint64(jetID.Prefix())
	if id == 0 {
		return false
	}
	mutex.Lock()
	_, hasKey := uniqueJetID[id]
	if !hasKey {
		uniqueJetID[id] = true
		mutex.Unlock()
		return true
	}
	mutex.Unlock()
	return false
}

// RandNumberOverRange generates random number over a range
func RandNumberOverRange(min int64, max int64) int64 {
	return rand.Int63n(max-min+1) + min
}

// GenerateRandBytes generates random bytes array
func GenerateRandBytes() []byte {
	u, _ := uuid.NewV4()
	return u.Bytes()
}

// Generate random string with specified length
func RandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// Generate map of pulses and related list of splitted JetDrops
func GenerateJetIDTree(pn insolar.PulseNumber, depth int) map[insolar.PulseNumber][]insolar.JetID {
	timeout := time.After(5 * time.Second)
	result := make(map[insolar.PulseNumber][]insolar.JetID, 0)
	for {
		select {
		case <-timeout:
			return map[insolar.PulseNumber][]insolar.JetID{}
		default:
		}
		rootJetID := *insolar.NewJetID(20, gen.IDWithPulse(pn).Bytes())
		if !isUniqueJetId(rootJetID) {
			continue
		}
		result[pn] = []insolar.JetID{rootJetID}

		childs := siblings(rootJetID, pn, depth)
		for p, c := range childs {
			result[p] = c
		}

		return result
	}
}

func siblings(parent insolar.JetID, parentPn insolar.PulseNumber, depth int) map[insolar.PulseNumber][]insolar.JetID {
	if depth == 0 {
		return nil
	}

	pn := parentPn
	result := make(map[insolar.PulseNumber][]insolar.JetID, 0)
	left, right := jet.Siblings(parent)
	pn += 10
	result[pn] = []insolar.JetID{left, right}

	l := siblings(left, pn, depth-1)
	for k, v := range l {
		if jds := result[k]; jds == nil {
			result[k] = v
		} else {
			jds = append(jds, v...)
			result[k] = jds
		}
	}
	r := siblings(right, pn, depth-1)
	for k, v := range r {
		if jds := result[k]; jds == nil {
			result[k] = v
		} else {
			jds = append(jds, v...)
			result[k] = jds
		}
	}

	return result
}

// GenerateJetDropsWithSplit returns a jetdrops with splited by depth in different pulse
func GenerateJetDropsWithSplit(t *testing.T, pulseCount, jDCount int, depth int) ([]models.JetDrop, []models.Pulse) {
	pulses := make([]models.Pulse, pulseCount)
	pulse, err := InitPulseDB()
	require.NoError(t, err)
	pulses[0] = pulse
	pn := pulse.PulseNumber
	for i := 1; i < pulseCount; i++ {
		pulse, err := InitNextPulseDB(pn)
		require.NoError(t, err)
		pulses[i] = pulse
		pn = pulse.PulseNumber
	}

	drops := make([]models.JetDrop, 0)
	for j := 0; j < pulseCount; j++ {
		for i := 0; i < jDCount; i++ {
			jDrop := InitJetDropDB(pulses[j])
			drops = append(drops, jDrop)
			drops = append(drops, createChildren(pulses[j], jDrop.JetID, depth)...)
		}
	}

	return drops, pulses
}

// InitJetDropWithRecords create new JetDrop, generate random records, save SaveJetDropData
func InitJetDropWithRecords(t *testing.T, s interfaces.StorageSetter, recordAmount int, pulse models.Pulse) models.JetDrop {
	jetDrop := InitJetDropDB(pulse)
	jetDrop.RecordAmount = recordAmount
	record := make([]models.Record, recordAmount)
	for i := 0; i < recordAmount; i++ {
		record[i] = InitRecordDB(jetDrop)
	}
	err := s.SaveJetDropData(jetDrop, record, pulse.PulseNumber)
	require.NoError(t, err)
	return jetDrop
}

// createChildren is the recursion function which prepare jetdrops where jetID will be splited
func createChildren(pulse models.Pulse, jetID string, depth int) []models.JetDrop {
	if depth == 0 {
		return nil
	}
	drops := make([]models.JetDrop, 2)
	left, right := jetID+"0", jetID+"1"

	jDropLeft, jDropRight := InitJetDropDB(pulse), InitJetDropDB(pulse)
	jDropLeft.JetID, jDropRight.JetID = left, right
	drops[0], drops[1] = jDropLeft, jDropRight

	drops = append(drops, createChildren(pulse, left, depth-1)...)
	drops = append(drops, createChildren(pulse, right, depth-1)...)
	return drops
}
