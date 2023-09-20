// +build unit

package testutils

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	ins_record "github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/pulse"
	"github.com/stretchr/testify/require"
)

func TestGenerateRecords_CanReturnEOF(t *testing.T) {
	batchSize := 5
	f := GenerateRecords(batchSize)

	n := uint32(1)
	for i := 0; i < batchSize; i++ {
		record, err := f()
		require.NoError(t, err)
		require.Equal(t, n, record.RecordNumber)
		n++
	}
	res, err := f()
	require.Error(t, err)
	require.Equal(t, io.EOF, err)
	require.Equal(t, &exporter.Record{}, res)
}

func TestGenerateRecordsSilence_recordsAreUnique(t *testing.T) {
	count := 5
	records := GenerateRecordsSilence(count)
	require.Len(t, records, count)
	for i, r := range records {
		require.Equal(t, uint32(i+1), r.RecordNumber)
	}
}

func TestGenerateUniqueJetIDFunction(t *testing.T) {
	ids := len(uniqueJetID)
	idFirst := GenerateUniqueJetID()
	require.NotEmpty(t, idFirst)
	require.Len(t, uniqueJetID, ids+1)

	idSecond := GenerateUniqueJetID()
	require.NotEqual(t, idFirst, idSecond)
	require.NotEmpty(t, idSecond)
	require.Len(t, uniqueJetID, ids+2)
}

func TestGenerateRecordsWithDifferencePulses(t *testing.T) {
	tests := []struct {
		differentPulseSize int
		recordCount        int
	}{
		{
			differentPulseSize: 1,
			recordCount:        1,
		}, {
			differentPulseSize: 1,
			recordCount:        2,
		}, {
			differentPulseSize: 2,
			recordCount:        1,
		}, {
			differentPulseSize: 2,
			recordCount:        2,
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("pulse-size=%d,record-count=%d", test.differentPulseSize, test.recordCount), func(t *testing.T) {
			fn := GenerateRecordsWithDifferencePulses(test.differentPulseSize, test.recordCount, int64(pulse.MinTimePulse))
			lastPn := uint32(0)
			for i := 0; i < test.differentPulseSize*test.recordCount+1; i++ {
				record, _ := fn()
				require.NotNil(t, record)
				pn := record.Record.ID.Pulse().AsUint32()
				require.GreaterOrEqual(t, pn, lastPn)
				lastPn = pn
			}

			_, err := fn()
			require.EqualError(t, err, io.EOF.Error())
		})
	}
}

func TestGenerateObjectLifeline(t *testing.T) {
	pulsesNumber := 5
	recordsNumber := 10
	lifeline := GenerateObjectLifeline(pulsesNumber, recordsNumber)
	require.Len(t, lifeline.StateRecords, pulsesNumber)
	require.Len(t, lifeline.SideRecords, 1)

	objID := lifeline.ObjID
	allRecords := make([]*exporter.Record, 0)
	var prevPn insolar.PulseNumber
	prevPn = 0
	for i := 0; i < pulsesNumber; i++ {
		pn := lifeline.StateRecords[i].Pn
		require.Greater(t, pn.AsUint32(), prevPn.AsUint32())
		prevPn = pn

		records := lifeline.StateRecords[i].Records
		require.Len(t, records, recordsNumber)
		allRecords = append(allRecords, records...)
	}

	var amendCount int
	var activateCount int
	var deactivateCount int
	var unknown int
	for _, r := range allRecords {
		require.Equal(t, objID, r.Record.ObjectID)

		virtual := r.Record.Virtual
		switch virtual.Union.(type) {
		case *ins_record.Virtual_Amend:
			amendCount++
		case *ins_record.Virtual_Activate:
			activateCount++
		case *ins_record.Virtual_Deactivate:
			deactivateCount++
		default:
			unknown++
		}
	}
	require.Equal(t, 0, unknown)
	require.Equal(t, pulsesNumber*recordsNumber-2, amendCount) // 2 = one activate record + one deactivate record
	require.Equal(t, 1, activateCount)
	require.Equal(t, 1, deactivateCount)

	sideRecords := make([]*exporter.Record, 0)
	for i := 0; i < len(lifeline.SideRecords); i++ {
		sideRecords = append(sideRecords, lifeline.SideRecords[i].Records...)
	}
	var incomingCount int
	for _, r := range sideRecords {
		require.Equal(t, objID, r.Record.ObjectID)

		virtual := r.Record.Virtual
		switch virtual.Union.(type) {
		case *ins_record.Virtual_IncomingRequest:
			incomingCount++
		default:
			unknown++
		}
	}
	require.Equal(t, 1, incomingCount)
	require.Equal(t, 0, unknown)

	all := lifeline.GetAllRecords()
	require.Len(t, all, pulsesNumber*recordsNumber+1)
	sr := lifeline.GetStateRecords()
	require.Len(t, sr, pulsesNumber*recordsNumber)
}

func TestChildren(t *testing.T) {
	tests := map[string]struct {
		jetID    string
		depth    int
		children []string
	}{
		"depth 1": {"0", 1, []string{"00", "01"}},
		"depth 2": {"0", 2, []string{"00", "01", "000", "001", "010", "011"}},
		"depth 3": {"0", 3, []string{
			"00", "01",
			"000", "001", "010", "011",
			"0000", "0001", "0010", "0011", "0100", "0101", "0110", "0111"}},
	}

	pulse, err := InitPulseDB()
	require.NoError(t, err)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := createChildren(pulse, test.jetID, test.depth)
			require.Len(t, result, len(test.children))
			for i := 0; i < len(test.children); i++ {
				require.Contains(t, test.children, result[i].JetID)
			}
		})
	}
}

func TestGenerateJetDropsWithSplit(t *testing.T) {
	tests := map[string]struct {
		pulseCount int
		jDCount    int
		depth      int
		total      int // (2^(depth + 1) - 1) * jc * pc
	}{
		"pc=1, jdc=1, depth=0, total=1":   {1, 1, 0, 1},
		"pc=1, jdc=1, depth=1, total=3":   {1, 1, 1, 3},
		"pc=2, jdc=1, depth=1, total=6":   {2, 1, 1, 6},
		"pc=1, jdc=2, depth=2, total=14":  {1, 2, 2, 14},
		"pc=2, jdc=2, depth=2, total=28":  {2, 2, 2, 28},
		"pc=2, jdc=2, depth=4, total=124": {2, 2, 4, 124},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			drops, _ := GenerateJetDropsWithSplit(t, test.pulseCount, test.jDCount, test.depth)
			require.Len(t, drops, test.total)
		})
	}
}

func TestRandomString(t *testing.T) {
	l := 100
	str := RandomString(l)
	require.Equal(t, l, len(str))
	require.True(t, strings.ContainsAny(string(letterRunes), str))
}

func TestGenerateJetIDTree(t *testing.T) {
	tests := map[string]struct {
		depth int
		total int // (2^(depth + 1) - 1)
	}{
		"depth=0, total=1":  {0, 1},
		"depth=1, total=3":  {1, 3},
		"depth=2, total=7":  {2, 7},
		"depth=3, total=15": {3, 15},
		"depth=4, total=31": {4, 31},
		"depth=5, total=63": {5, 63},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			jDcount := 0
			pn := gen.PulseNumber()
			res := GenerateJetIDTree(pn, tc.depth)
			for _, v := range res {
				jDcount += len(v)
			}
			require.Equal(t, tc.total, jDcount)
		})
	}
}
