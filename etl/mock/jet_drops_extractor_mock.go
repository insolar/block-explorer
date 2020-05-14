package mock

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/insolar/ledger/heavy/exporter"
)

// JetDropsExtractorMock implements etl.JetDropsExtractor
type JetDropsExtractorMock struct {
	t minimock.Tester

	funcGetRecords          func() (ch1 <-chan exporter.Record, err error)
	inspectFuncGetRecords   func()
	afterGetRecordsCounter  uint64
	beforeGetRecordsCounter uint64
	GetRecordsMock          mJetDropsExtractorMockGetRecords
}

// NewJetDropsExtractorMock returns a mock for etl.JetDropsExtractor
func NewJetDropsExtractorMock(t minimock.Tester) *JetDropsExtractorMock {
	m := &JetDropsExtractorMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.GetRecordsMock = mJetDropsExtractorMockGetRecords{mock: m}

	return m
}

type mJetDropsExtractorMockGetRecords struct {
	mock               *JetDropsExtractorMock
	defaultExpectation *JetDropsExtractorMockGetRecordsExpectation
	expectations       []*JetDropsExtractorMockGetRecordsExpectation
}

// JetDropsExtractorMockGetRecordsExpectation specifies expectation struct of the JetDropsExtractor.GetRecords
type JetDropsExtractorMockGetRecordsExpectation struct {
	mock *JetDropsExtractorMock

	results *JetDropsExtractorMockGetRecordsResults
	Counter uint64
}

// JetDropsExtractorMockGetRecordsResults contains results of the JetDropsExtractor.GetRecords
type JetDropsExtractorMockGetRecordsResults struct {
	ch1 <-chan exporter.Record
	err error
}

// Expect sets up expected params for JetDropsExtractor.GetRecords
func (mmGetRecords *mJetDropsExtractorMockGetRecords) Expect() *mJetDropsExtractorMockGetRecords {
	if mmGetRecords.mock.funcGetRecords != nil {
		mmGetRecords.mock.t.Fatalf("JetDropsExtractorMock.GetRecords mock is already set by Set")
	}

	if mmGetRecords.defaultExpectation == nil {
		mmGetRecords.defaultExpectation = &JetDropsExtractorMockGetRecordsExpectation{}
	}

	return mmGetRecords
}

// Inspect accepts an inspector function that has same arguments as the JetDropsExtractor.GetRecords
func (mmGetRecords *mJetDropsExtractorMockGetRecords) Inspect(f func()) *mJetDropsExtractorMockGetRecords {
	if mmGetRecords.mock.inspectFuncGetRecords != nil {
		mmGetRecords.mock.t.Fatalf("Inspect function is already set for JetDropsExtractorMock.GetRecords")
	}

	mmGetRecords.mock.inspectFuncGetRecords = f

	return mmGetRecords
}

// Return sets up results that will be returned by JetDropsExtractor.GetRecords
func (mmGetRecords *mJetDropsExtractorMockGetRecords) Return(ch1 <-chan exporter.Record, err error) *JetDropsExtractorMock {
	if mmGetRecords.mock.funcGetRecords != nil {
		mmGetRecords.mock.t.Fatalf("JetDropsExtractorMock.GetRecords mock is already set by Set")
	}

	if mmGetRecords.defaultExpectation == nil {
		mmGetRecords.defaultExpectation = &JetDropsExtractorMockGetRecordsExpectation{mock: mmGetRecords.mock}
	}
	mmGetRecords.defaultExpectation.results = &JetDropsExtractorMockGetRecordsResults{ch1, err}
	return mmGetRecords.mock
}

//Set uses given function f to mock the JetDropsExtractor.GetRecords method
func (mmGetRecords *mJetDropsExtractorMockGetRecords) Set(f func() (ch1 <-chan exporter.Record, err error)) *JetDropsExtractorMock {
	if mmGetRecords.defaultExpectation != nil {
		mmGetRecords.mock.t.Fatalf("Default expectation is already set for the JetDropsExtractor.GetRecords method")
	}

	if len(mmGetRecords.expectations) > 0 {
		mmGetRecords.mock.t.Fatalf("Some expectations are already set for the JetDropsExtractor.GetRecords method")
	}

	mmGetRecords.mock.funcGetRecords = f
	return mmGetRecords.mock
}

// GetRecords implements etl.JetDropsExtractor
func (mmGetRecords *JetDropsExtractorMock) GetRecords() (ch1 <-chan exporter.Record, err error) {
	mm_atomic.AddUint64(&mmGetRecords.beforeGetRecordsCounter, 1)
	defer mm_atomic.AddUint64(&mmGetRecords.afterGetRecordsCounter, 1)

	if mmGetRecords.inspectFuncGetRecords != nil {
		mmGetRecords.inspectFuncGetRecords()
	}

	if mmGetRecords.GetRecordsMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetRecords.GetRecordsMock.defaultExpectation.Counter, 1)

		mm_results := mmGetRecords.GetRecordsMock.defaultExpectation.results
		if mm_results == nil {
			mmGetRecords.t.Fatal("No results are set for the JetDropsExtractorMock.GetRecords")
		}
		return (*mm_results).ch1, (*mm_results).err
	}
	if mmGetRecords.funcGetRecords != nil {
		return mmGetRecords.funcGetRecords()
	}
	mmGetRecords.t.Fatalf("Unexpected call to JetDropsExtractorMock.GetRecords.")
	return
}

// GetRecordsAfterCounter returns a count of finished JetDropsExtractorMock.GetRecords invocations
func (mmGetRecords *JetDropsExtractorMock) GetRecordsAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetRecords.afterGetRecordsCounter)
}

// GetRecordsBeforeCounter returns a count of JetDropsExtractorMock.GetRecords invocations
func (mmGetRecords *JetDropsExtractorMock) GetRecordsBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetRecords.beforeGetRecordsCounter)
}

// MinimockGetRecordsDone returns true if the count of the GetRecords invocations corresponds
// the number of defined expectations
func (m *JetDropsExtractorMock) MinimockGetRecordsDone() bool {
	for _, e := range m.GetRecordsMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetRecordsMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetRecordsCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetRecords != nil && mm_atomic.LoadUint64(&m.afterGetRecordsCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetRecordsInspect logs each unmet expectation
func (m *JetDropsExtractorMock) MinimockGetRecordsInspect() {
	for _, e := range m.GetRecordsMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to JetDropsExtractorMock.GetRecords")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetRecordsMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetRecordsCounter) < 1 {
		m.t.Error("Expected call to JetDropsExtractorMock.GetRecords")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetRecords != nil && mm_atomic.LoadUint64(&m.afterGetRecordsCounter) < 1 {
		m.t.Error("Expected call to JetDropsExtractorMock.GetRecords")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *JetDropsExtractorMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockGetRecordsInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *JetDropsExtractorMock) MinimockWait(timeout mm_time.Duration) {
	timeoutCh := mm_time.After(timeout)
	for {
		if m.minimockDone() {
			return
		}
		select {
		case <-timeoutCh:
			m.MinimockFinish()
			return
		case <-mm_time.After(10 * mm_time.Millisecond):
		}
	}
}

func (m *JetDropsExtractorMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockGetRecordsDone()
}
