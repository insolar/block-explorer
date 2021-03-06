package mock

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/block-explorer/etl/types"
)

// ControllerMock implements interfaces.Controller
type ControllerMock struct {
	t minimock.Tester

	funcSetJetDropData          func(pulse types.Pulse, jetID string)
	inspectFuncSetJetDropData   func(pulse types.Pulse, jetID string)
	afterSetJetDropDataCounter  uint64
	beforeSetJetDropDataCounter uint64
	SetJetDropDataMock          mControllerMockSetJetDropData

	funcStart          func(ctx context.Context) (err error)
	inspectFuncStart   func(ctx context.Context)
	afterStartCounter  uint64
	beforeStartCounter uint64
	StartMock          mControllerMockStart

	funcStop          func(ctx context.Context) (err error)
	inspectFuncStop   func(ctx context.Context)
	afterStopCounter  uint64
	beforeStopCounter uint64
	StopMock          mControllerMockStop
}

// NewControllerMock returns a mock for interfaces.Controller
func NewControllerMock(t minimock.Tester) *ControllerMock {
	m := &ControllerMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.SetJetDropDataMock = mControllerMockSetJetDropData{mock: m}
	m.SetJetDropDataMock.callArgs = []*ControllerMockSetJetDropDataParams{}

	m.StartMock = mControllerMockStart{mock: m}
	m.StartMock.callArgs = []*ControllerMockStartParams{}

	m.StopMock = mControllerMockStop{mock: m}
	m.StopMock.callArgs = []*ControllerMockStopParams{}

	return m
}

type mControllerMockSetJetDropData struct {
	mock               *ControllerMock
	defaultExpectation *ControllerMockSetJetDropDataExpectation
	expectations       []*ControllerMockSetJetDropDataExpectation

	callArgs []*ControllerMockSetJetDropDataParams
	mutex    sync.RWMutex
}

// ControllerMockSetJetDropDataExpectation specifies expectation struct of the Controller.SetJetDropData
type ControllerMockSetJetDropDataExpectation struct {
	mock   *ControllerMock
	params *ControllerMockSetJetDropDataParams

	Counter uint64
}

// ControllerMockSetJetDropDataParams contains parameters of the Controller.SetJetDropData
type ControllerMockSetJetDropDataParams struct {
	pulse types.Pulse
	jetID string
}

// Expect sets up expected params for Controller.SetJetDropData
func (mmSetJetDropData *mControllerMockSetJetDropData) Expect(pulse types.Pulse, jetID string) *mControllerMockSetJetDropData {
	if mmSetJetDropData.mock.funcSetJetDropData != nil {
		mmSetJetDropData.mock.t.Fatalf("ControllerMock.SetJetDropData mock is already set by Set")
	}

	if mmSetJetDropData.defaultExpectation == nil {
		mmSetJetDropData.defaultExpectation = &ControllerMockSetJetDropDataExpectation{}
	}

	mmSetJetDropData.defaultExpectation.params = &ControllerMockSetJetDropDataParams{pulse, jetID}
	for _, e := range mmSetJetDropData.expectations {
		if minimock.Equal(e.params, mmSetJetDropData.defaultExpectation.params) {
			mmSetJetDropData.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmSetJetDropData.defaultExpectation.params)
		}
	}

	return mmSetJetDropData
}

// Inspect accepts an inspector function that has same arguments as the Controller.SetJetDropData
func (mmSetJetDropData *mControllerMockSetJetDropData) Inspect(f func(pulse types.Pulse, jetID string)) *mControllerMockSetJetDropData {
	if mmSetJetDropData.mock.inspectFuncSetJetDropData != nil {
		mmSetJetDropData.mock.t.Fatalf("Inspect function is already set for ControllerMock.SetJetDropData")
	}

	mmSetJetDropData.mock.inspectFuncSetJetDropData = f

	return mmSetJetDropData
}

// Return sets up results that will be returned by Controller.SetJetDropData
func (mmSetJetDropData *mControllerMockSetJetDropData) Return() *ControllerMock {
	if mmSetJetDropData.mock.funcSetJetDropData != nil {
		mmSetJetDropData.mock.t.Fatalf("ControllerMock.SetJetDropData mock is already set by Set")
	}

	if mmSetJetDropData.defaultExpectation == nil {
		mmSetJetDropData.defaultExpectation = &ControllerMockSetJetDropDataExpectation{mock: mmSetJetDropData.mock}
	}

	return mmSetJetDropData.mock
}

//Set uses given function f to mock the Controller.SetJetDropData method
func (mmSetJetDropData *mControllerMockSetJetDropData) Set(f func(pulse types.Pulse, jetID string)) *ControllerMock {
	if mmSetJetDropData.defaultExpectation != nil {
		mmSetJetDropData.mock.t.Fatalf("Default expectation is already set for the Controller.SetJetDropData method")
	}

	if len(mmSetJetDropData.expectations) > 0 {
		mmSetJetDropData.mock.t.Fatalf("Some expectations are already set for the Controller.SetJetDropData method")
	}

	mmSetJetDropData.mock.funcSetJetDropData = f
	return mmSetJetDropData.mock
}

// SetJetDropData implements interfaces.Controller
func (mmSetJetDropData *ControllerMock) SetJetDropData(pulse types.Pulse, jetID string) {
	mm_atomic.AddUint64(&mmSetJetDropData.beforeSetJetDropDataCounter, 1)
	defer mm_atomic.AddUint64(&mmSetJetDropData.afterSetJetDropDataCounter, 1)

	if mmSetJetDropData.inspectFuncSetJetDropData != nil {
		mmSetJetDropData.inspectFuncSetJetDropData(pulse, jetID)
	}

	mm_params := &ControllerMockSetJetDropDataParams{pulse, jetID}

	// Record call args
	mmSetJetDropData.SetJetDropDataMock.mutex.Lock()
	mmSetJetDropData.SetJetDropDataMock.callArgs = append(mmSetJetDropData.SetJetDropDataMock.callArgs, mm_params)
	mmSetJetDropData.SetJetDropDataMock.mutex.Unlock()

	for _, e := range mmSetJetDropData.SetJetDropDataMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return
		}
	}

	if mmSetJetDropData.SetJetDropDataMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmSetJetDropData.SetJetDropDataMock.defaultExpectation.Counter, 1)
		mm_want := mmSetJetDropData.SetJetDropDataMock.defaultExpectation.params
		mm_got := ControllerMockSetJetDropDataParams{pulse, jetID}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmSetJetDropData.t.Errorf("ControllerMock.SetJetDropData got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		return

	}
	if mmSetJetDropData.funcSetJetDropData != nil {
		mmSetJetDropData.funcSetJetDropData(pulse, jetID)
		return
	}
	mmSetJetDropData.t.Fatalf("Unexpected call to ControllerMock.SetJetDropData. %v %v", pulse, jetID)

}

// SetJetDropDataAfterCounter returns a count of finished ControllerMock.SetJetDropData invocations
func (mmSetJetDropData *ControllerMock) SetJetDropDataAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSetJetDropData.afterSetJetDropDataCounter)
}

// SetJetDropDataBeforeCounter returns a count of ControllerMock.SetJetDropData invocations
func (mmSetJetDropData *ControllerMock) SetJetDropDataBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSetJetDropData.beforeSetJetDropDataCounter)
}

// Calls returns a list of arguments used in each call to ControllerMock.SetJetDropData.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmSetJetDropData *mControllerMockSetJetDropData) Calls() []*ControllerMockSetJetDropDataParams {
	mmSetJetDropData.mutex.RLock()

	argCopy := make([]*ControllerMockSetJetDropDataParams, len(mmSetJetDropData.callArgs))
	copy(argCopy, mmSetJetDropData.callArgs)

	mmSetJetDropData.mutex.RUnlock()

	return argCopy
}

// MinimockSetJetDropDataDone returns true if the count of the SetJetDropData invocations corresponds
// the number of defined expectations
func (m *ControllerMock) MinimockSetJetDropDataDone() bool {
	for _, e := range m.SetJetDropDataMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SetJetDropDataMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSetJetDropDataCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSetJetDropData != nil && mm_atomic.LoadUint64(&m.afterSetJetDropDataCounter) < 1 {
		return false
	}
	return true
}

// MinimockSetJetDropDataInspect logs each unmet expectation
func (m *ControllerMock) MinimockSetJetDropDataInspect() {
	for _, e := range m.SetJetDropDataMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to ControllerMock.SetJetDropData with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SetJetDropDataMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSetJetDropDataCounter) < 1 {
		if m.SetJetDropDataMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to ControllerMock.SetJetDropData")
		} else {
			m.t.Errorf("Expected call to ControllerMock.SetJetDropData with params: %#v", *m.SetJetDropDataMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSetJetDropData != nil && mm_atomic.LoadUint64(&m.afterSetJetDropDataCounter) < 1 {
		m.t.Error("Expected call to ControllerMock.SetJetDropData")
	}
}

type mControllerMockStart struct {
	mock               *ControllerMock
	defaultExpectation *ControllerMockStartExpectation
	expectations       []*ControllerMockStartExpectation

	callArgs []*ControllerMockStartParams
	mutex    sync.RWMutex
}

// ControllerMockStartExpectation specifies expectation struct of the Controller.Start
type ControllerMockStartExpectation struct {
	mock    *ControllerMock
	params  *ControllerMockStartParams
	results *ControllerMockStartResults
	Counter uint64
}

// ControllerMockStartParams contains parameters of the Controller.Start
type ControllerMockStartParams struct {
	ctx context.Context
}

// ControllerMockStartResults contains results of the Controller.Start
type ControllerMockStartResults struct {
	err error
}

// Expect sets up expected params for Controller.Start
func (mmStart *mControllerMockStart) Expect(ctx context.Context) *mControllerMockStart {
	if mmStart.mock.funcStart != nil {
		mmStart.mock.t.Fatalf("ControllerMock.Start mock is already set by Set")
	}

	if mmStart.defaultExpectation == nil {
		mmStart.defaultExpectation = &ControllerMockStartExpectation{}
	}

	mmStart.defaultExpectation.params = &ControllerMockStartParams{ctx}
	for _, e := range mmStart.expectations {
		if minimock.Equal(e.params, mmStart.defaultExpectation.params) {
			mmStart.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmStart.defaultExpectation.params)
		}
	}

	return mmStart
}

// Inspect accepts an inspector function that has same arguments as the Controller.Start
func (mmStart *mControllerMockStart) Inspect(f func(ctx context.Context)) *mControllerMockStart {
	if mmStart.mock.inspectFuncStart != nil {
		mmStart.mock.t.Fatalf("Inspect function is already set for ControllerMock.Start")
	}

	mmStart.mock.inspectFuncStart = f

	return mmStart
}

// Return sets up results that will be returned by Controller.Start
func (mmStart *mControllerMockStart) Return(err error) *ControllerMock {
	if mmStart.mock.funcStart != nil {
		mmStart.mock.t.Fatalf("ControllerMock.Start mock is already set by Set")
	}

	if mmStart.defaultExpectation == nil {
		mmStart.defaultExpectation = &ControllerMockStartExpectation{mock: mmStart.mock}
	}
	mmStart.defaultExpectation.results = &ControllerMockStartResults{err}
	return mmStart.mock
}

//Set uses given function f to mock the Controller.Start method
func (mmStart *mControllerMockStart) Set(f func(ctx context.Context) (err error)) *ControllerMock {
	if mmStart.defaultExpectation != nil {
		mmStart.mock.t.Fatalf("Default expectation is already set for the Controller.Start method")
	}

	if len(mmStart.expectations) > 0 {
		mmStart.mock.t.Fatalf("Some expectations are already set for the Controller.Start method")
	}

	mmStart.mock.funcStart = f
	return mmStart.mock
}

// When sets expectation for the Controller.Start which will trigger the result defined by the following
// Then helper
func (mmStart *mControllerMockStart) When(ctx context.Context) *ControllerMockStartExpectation {
	if mmStart.mock.funcStart != nil {
		mmStart.mock.t.Fatalf("ControllerMock.Start mock is already set by Set")
	}

	expectation := &ControllerMockStartExpectation{
		mock:   mmStart.mock,
		params: &ControllerMockStartParams{ctx},
	}
	mmStart.expectations = append(mmStart.expectations, expectation)
	return expectation
}

// Then sets up Controller.Start return parameters for the expectation previously defined by the When method
func (e *ControllerMockStartExpectation) Then(err error) *ControllerMock {
	e.results = &ControllerMockStartResults{err}
	return e.mock
}

// Start implements interfaces.Controller
func (mmStart *ControllerMock) Start(ctx context.Context) (err error) {
	mm_atomic.AddUint64(&mmStart.beforeStartCounter, 1)
	defer mm_atomic.AddUint64(&mmStart.afterStartCounter, 1)

	if mmStart.inspectFuncStart != nil {
		mmStart.inspectFuncStart(ctx)
	}

	mm_params := &ControllerMockStartParams{ctx}

	// Record call args
	mmStart.StartMock.mutex.Lock()
	mmStart.StartMock.callArgs = append(mmStart.StartMock.callArgs, mm_params)
	mmStart.StartMock.mutex.Unlock()

	for _, e := range mmStart.StartMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmStart.StartMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmStart.StartMock.defaultExpectation.Counter, 1)
		mm_want := mmStart.StartMock.defaultExpectation.params
		mm_got := ControllerMockStartParams{ctx}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmStart.t.Errorf("ControllerMock.Start got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmStart.StartMock.defaultExpectation.results
		if mm_results == nil {
			mmStart.t.Fatal("No results are set for the ControllerMock.Start")
		}
		return (*mm_results).err
	}
	if mmStart.funcStart != nil {
		return mmStart.funcStart(ctx)
	}
	mmStart.t.Fatalf("Unexpected call to ControllerMock.Start. %v", ctx)
	return
}

// StartAfterCounter returns a count of finished ControllerMock.Start invocations
func (mmStart *ControllerMock) StartAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmStart.afterStartCounter)
}

// StartBeforeCounter returns a count of ControllerMock.Start invocations
func (mmStart *ControllerMock) StartBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmStart.beforeStartCounter)
}

// Calls returns a list of arguments used in each call to ControllerMock.Start.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmStart *mControllerMockStart) Calls() []*ControllerMockStartParams {
	mmStart.mutex.RLock()

	argCopy := make([]*ControllerMockStartParams, len(mmStart.callArgs))
	copy(argCopy, mmStart.callArgs)

	mmStart.mutex.RUnlock()

	return argCopy
}

// MinimockStartDone returns true if the count of the Start invocations corresponds
// the number of defined expectations
func (m *ControllerMock) MinimockStartDone() bool {
	for _, e := range m.StartMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.StartMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterStartCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcStart != nil && mm_atomic.LoadUint64(&m.afterStartCounter) < 1 {
		return false
	}
	return true
}

// MinimockStartInspect logs each unmet expectation
func (m *ControllerMock) MinimockStartInspect() {
	for _, e := range m.StartMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to ControllerMock.Start with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.StartMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterStartCounter) < 1 {
		if m.StartMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to ControllerMock.Start")
		} else {
			m.t.Errorf("Expected call to ControllerMock.Start with params: %#v", *m.StartMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcStart != nil && mm_atomic.LoadUint64(&m.afterStartCounter) < 1 {
		m.t.Error("Expected call to ControllerMock.Start")
	}
}

type mControllerMockStop struct {
	mock               *ControllerMock
	defaultExpectation *ControllerMockStopExpectation
	expectations       []*ControllerMockStopExpectation

	callArgs []*ControllerMockStopParams
	mutex    sync.RWMutex
}

// ControllerMockStopExpectation specifies expectation struct of the Controller.Stop
type ControllerMockStopExpectation struct {
	mock    *ControllerMock
	params  *ControllerMockStopParams
	results *ControllerMockStopResults
	Counter uint64
}

// ControllerMockStopParams contains parameters of the Controller.Stop
type ControllerMockStopParams struct {
	ctx context.Context
}

// ControllerMockStopResults contains results of the Controller.Stop
type ControllerMockStopResults struct {
	err error
}

// Expect sets up expected params for Controller.Stop
func (mmStop *mControllerMockStop) Expect(ctx context.Context) *mControllerMockStop {
	if mmStop.mock.funcStop != nil {
		mmStop.mock.t.Fatalf("ControllerMock.Stop mock is already set by Set")
	}

	if mmStop.defaultExpectation == nil {
		mmStop.defaultExpectation = &ControllerMockStopExpectation{}
	}

	mmStop.defaultExpectation.params = &ControllerMockStopParams{ctx}
	for _, e := range mmStop.expectations {
		if minimock.Equal(e.params, mmStop.defaultExpectation.params) {
			mmStop.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmStop.defaultExpectation.params)
		}
	}

	return mmStop
}

// Inspect accepts an inspector function that has same arguments as the Controller.Stop
func (mmStop *mControllerMockStop) Inspect(f func(ctx context.Context)) *mControllerMockStop {
	if mmStop.mock.inspectFuncStop != nil {
		mmStop.mock.t.Fatalf("Inspect function is already set for ControllerMock.Stop")
	}

	mmStop.mock.inspectFuncStop = f

	return mmStop
}

// Return sets up results that will be returned by Controller.Stop
func (mmStop *mControllerMockStop) Return(err error) *ControllerMock {
	if mmStop.mock.funcStop != nil {
		mmStop.mock.t.Fatalf("ControllerMock.Stop mock is already set by Set")
	}

	if mmStop.defaultExpectation == nil {
		mmStop.defaultExpectation = &ControllerMockStopExpectation{mock: mmStop.mock}
	}
	mmStop.defaultExpectation.results = &ControllerMockStopResults{err}
	return mmStop.mock
}

//Set uses given function f to mock the Controller.Stop method
func (mmStop *mControllerMockStop) Set(f func(ctx context.Context) (err error)) *ControllerMock {
	if mmStop.defaultExpectation != nil {
		mmStop.mock.t.Fatalf("Default expectation is already set for the Controller.Stop method")
	}

	if len(mmStop.expectations) > 0 {
		mmStop.mock.t.Fatalf("Some expectations are already set for the Controller.Stop method")
	}

	mmStop.mock.funcStop = f
	return mmStop.mock
}

// When sets expectation for the Controller.Stop which will trigger the result defined by the following
// Then helper
func (mmStop *mControllerMockStop) When(ctx context.Context) *ControllerMockStopExpectation {
	if mmStop.mock.funcStop != nil {
		mmStop.mock.t.Fatalf("ControllerMock.Stop mock is already set by Set")
	}

	expectation := &ControllerMockStopExpectation{
		mock:   mmStop.mock,
		params: &ControllerMockStopParams{ctx},
	}
	mmStop.expectations = append(mmStop.expectations, expectation)
	return expectation
}

// Then sets up Controller.Stop return parameters for the expectation previously defined by the When method
func (e *ControllerMockStopExpectation) Then(err error) *ControllerMock {
	e.results = &ControllerMockStopResults{err}
	return e.mock
}

// Stop implements interfaces.Controller
func (mmStop *ControllerMock) Stop(ctx context.Context) (err error) {
	mm_atomic.AddUint64(&mmStop.beforeStopCounter, 1)
	defer mm_atomic.AddUint64(&mmStop.afterStopCounter, 1)

	if mmStop.inspectFuncStop != nil {
		mmStop.inspectFuncStop(ctx)
	}

	mm_params := &ControllerMockStopParams{ctx}

	// Record call args
	mmStop.StopMock.mutex.Lock()
	mmStop.StopMock.callArgs = append(mmStop.StopMock.callArgs, mm_params)
	mmStop.StopMock.mutex.Unlock()

	for _, e := range mmStop.StopMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmStop.StopMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmStop.StopMock.defaultExpectation.Counter, 1)
		mm_want := mmStop.StopMock.defaultExpectation.params
		mm_got := ControllerMockStopParams{ctx}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmStop.t.Errorf("ControllerMock.Stop got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmStop.StopMock.defaultExpectation.results
		if mm_results == nil {
			mmStop.t.Fatal("No results are set for the ControllerMock.Stop")
		}
		return (*mm_results).err
	}
	if mmStop.funcStop != nil {
		return mmStop.funcStop(ctx)
	}
	mmStop.t.Fatalf("Unexpected call to ControllerMock.Stop. %v", ctx)
	return
}

// StopAfterCounter returns a count of finished ControllerMock.Stop invocations
func (mmStop *ControllerMock) StopAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmStop.afterStopCounter)
}

// StopBeforeCounter returns a count of ControllerMock.Stop invocations
func (mmStop *ControllerMock) StopBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmStop.beforeStopCounter)
}

// Calls returns a list of arguments used in each call to ControllerMock.Stop.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmStop *mControllerMockStop) Calls() []*ControllerMockStopParams {
	mmStop.mutex.RLock()

	argCopy := make([]*ControllerMockStopParams, len(mmStop.callArgs))
	copy(argCopy, mmStop.callArgs)

	mmStop.mutex.RUnlock()

	return argCopy
}

// MinimockStopDone returns true if the count of the Stop invocations corresponds
// the number of defined expectations
func (m *ControllerMock) MinimockStopDone() bool {
	for _, e := range m.StopMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.StopMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterStopCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcStop != nil && mm_atomic.LoadUint64(&m.afterStopCounter) < 1 {
		return false
	}
	return true
}

// MinimockStopInspect logs each unmet expectation
func (m *ControllerMock) MinimockStopInspect() {
	for _, e := range m.StopMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to ControllerMock.Stop with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.StopMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterStopCounter) < 1 {
		if m.StopMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to ControllerMock.Stop")
		} else {
			m.t.Errorf("Expected call to ControllerMock.Stop with params: %#v", *m.StopMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcStop != nil && mm_atomic.LoadUint64(&m.afterStopCounter) < 1 {
		m.t.Error("Expected call to ControllerMock.Stop")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *ControllerMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockSetJetDropDataInspect()

		m.MinimockStartInspect()

		m.MinimockStopInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *ControllerMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *ControllerMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockSetJetDropDataDone() &&
		m.MinimockStartDone() &&
		m.MinimockStopDone()
}
