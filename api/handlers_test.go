// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration

package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/jet"
	"github.com/insolar/insolar/pulse"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/server"
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/etl/storage"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/insolar/block-explorer/testutils"
)

const (
	apihost = ":14800"
)

var testDB *gorm.DB

func TestMain(t *testing.M) {
	var dbCleaner func()
	var err error
	testDB, dbCleaner, err = testutils.SetupDB()
	if err != nil {
		belogger.FromContext(context.Background()).Fatal(err)
	}

	e := echo.New()

	s := storage.NewStorage(testDB)

	blockExplorerAPI := NewServer(context.Background(), s, configuration.API{})

	server.RegisterHandlers(e, blockExplorerAPI)
	stopped := make(chan struct{})
	go func() {
		err := e.Start(apihost)
		if err != http.ErrServerClosed {
			dbCleaner()
			e.Logger.Fatal(err)
		}
		stopped <- struct{}{}
	}()
	// TODO: wait until API started
	time.Sleep(5 * time.Second)

	retCode := t.Run()

	dbCleaner()

	if err := e.Close(); err != nil {
		e.Logger.Fatal(err)
	}
	<-stopped

	os.Exit(retCode)
}

func TestObjectLifeline_HappyPath(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

	// insert records
	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	jetDrop := testutils.InitJetDropDB(pulse)
	err = testutils.CreateJetDrop(testDB, jetDrop)
	require.NoError(t, err)

	objRef := gen.Reference()

	genRecords := testutils.OrderedRecords(t, testDB, jetDrop, *objRef.GetLocal(), 3)
	testutils.OrderedRecords(t, testDB, jetDrop, gen.ID(), 3)

	// request records for objRef
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + objRef.String() + "/states?limit=20")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.RecordsResponse
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.EqualValues(t, 3, int(*received.Total))
	require.Len(t, *received.Result, 3)
	// check desc order by default
	require.Equal(t, insolar.NewIDFromBytes(genRecords[0].Reference).String(), *(*received.Result)[2].Reference)
	require.Equal(t, insolar.NewIDFromBytes(genRecords[1].Reference).String(), *(*received.Result)[1].Reference)
	require.Equal(t, insolar.NewIDFromBytes(genRecords[2].Reference).String(), *(*received.Result)[0].Reference)
}

func TestObjectLifeline_TimestampRange(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

	// insert records
	firstPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, firstPulse)
	require.NoError(t, err)
	firstJetDrop := testutils.InitJetDropDB(firstPulse)
	err = testutils.CreateJetDrop(testDB, firstJetDrop)
	require.NoError(t, err)

	secondPulse := models.Pulse{
		PulseNumber: firstPulse.PulseNumber + 10,
		Timestamp:   firstPulse.Timestamp + 10,
	}
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)
	secondJetDrop := testutils.InitJetDropDB(secondPulse)
	secondJetDrop.JetID = firstJetDrop.JetID
	err = testutils.CreateJetDrop(testDB, secondJetDrop)
	require.NoError(t, err)

	thirdPulse := models.Pulse{
		PulseNumber: secondPulse.PulseNumber + 10,
		Timestamp:   secondPulse.Timestamp + 10,
	}
	err = testutils.CreatePulse(testDB, thirdPulse)
	require.NoError(t, err)
	thirdJetDrop := testutils.InitJetDropDB(thirdPulse)
	thirdJetDrop.JetID = firstJetDrop.JetID
	err = testutils.CreateJetDrop(testDB, thirdJetDrop)
	require.NoError(t, err)

	fourthPulse := models.Pulse{
		PulseNumber: thirdPulse.PulseNumber + 10,
		Timestamp:   thirdPulse.Timestamp + 10,
	}
	err = testutils.CreatePulse(testDB, fourthPulse)
	require.NoError(t, err)
	fourthJetDrop := testutils.InitJetDropDB(fourthPulse)
	fourthJetDrop.JetID = firstJetDrop.JetID
	err = testutils.CreateJetDrop(testDB, fourthJetDrop)
	require.NoError(t, err)

	objRef := gen.Reference()

	// pulse is later
	testutils.OrderedRecords(t, testDB, firstJetDrop, *objRef.GetLocal(), 2)
	genRecordsSecond := testutils.OrderedRecords(t, testDB, secondJetDrop, *objRef.GetLocal(), 2)
	genRecordsThird := testutils.OrderedRecords(t, testDB, thirdJetDrop, *objRef.GetLocal(), 2)
	// pulse is greater
	testutils.OrderedRecords(t, testDB, fourthJetDrop, *objRef.GetLocal(), 2)
	// incorrect object, correct pulse
	testutils.OrderedRecords(t, testDB, secondJetDrop, gen.ID(), 2)

	// request records for objRef
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + objRef.String() + "/states?limit=20" +
		fmt.Sprintf("&timestamp_lte=%d&timestamp_gte=%d", thirdPulse.Timestamp, secondPulse.Timestamp))

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.RecordsResponse
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.EqualValues(t, 4, int(*received.Total))
	require.Len(t, *received.Result, 4)
	// check desc order by default
	require.Equal(t, insolar.NewIDFromBytes(genRecordsSecond[0].Reference).String(), *(*received.Result)[3].Reference)
	require.Equal(t, insolar.NewIDFromBytes(genRecordsSecond[1].Reference).String(), *(*received.Result)[2].Reference)
	require.Equal(t, insolar.NewIDFromBytes(genRecordsThird[0].Reference).String(), *(*received.Result)[1].Reference)
	require.Equal(t, insolar.NewIDFromBytes(genRecordsThird[1].Reference).String(), *(*received.Result)[0].Reference)
}

func TestObjectLifeline_SortAsc(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

	// insert records
	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	jetDrop := testutils.InitJetDropDB(pulse)
	err = testutils.CreateJetDrop(testDB, jetDrop)
	require.NoError(t, err)

	objRef := gen.Reference()

	genRecords := testutils.OrderedRecords(t, testDB, jetDrop, *objRef.GetLocal(), 3)
	testutils.OrderedRecords(t, testDB, jetDrop, gen.ID(), 3)

	// request records for objRef
	sortAsc := string(server.SortByIndex_index_asc)
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + objRef.String() + "/states?sort_by=" + url.QueryEscape(sortAsc))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.RecordsResponse
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.EqualValues(t, 3, int(*received.Total))
	require.Len(t, *received.Result, 3)
	// check asc order
	require.Equal(t, insolar.NewIDFromBytes(genRecords[0].Reference).String(), *(*received.Result)[0].Reference)
	require.Equal(t, insolar.NewIDFromBytes(genRecords[1].Reference).String(), *(*received.Result)[1].Reference)
	require.Equal(t, insolar.NewIDFromBytes(genRecords[2].Reference).String(), *(*received.Result)[2].Reference)
}

func TestObjectLifeline_Limit_Error(t *testing.T) {
	// request records with too big limit
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + gen.Reference().String() + "/states?limit=200000000")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.CodeValidationError
	err = json.Unmarshal(bodyBytes, &received)
	fmt.Println(string(bodyBytes))
	require.NoError(t, err)

	expected := server.CodeValidationError{
		Code:    NullableString(http.StatusText(http.StatusBadRequest)),
		Message: NullableString(InvalidParamsMessage),
		ValidationFailures: &[]server.CodeValidationFailures{{
			FailureReason: NullableString("should be in range [1, 1000]"),
			Property:      NullableString("limit"),
		}},
	}
	require.Equal(t, expected, received)
}

func TestObjectLifeline_Offset_Error(t *testing.T) {
	// request records with negative offset
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + gen.Reference().String() + "/states?offset=-10")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.CodeValidationError
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	expected := server.CodeValidationError{
		Code:    NullableString(http.StatusText(http.StatusBadRequest)),
		Message: NullableString(InvalidParamsMessage),
		ValidationFailures: &[]server.CodeValidationFailures{{
			FailureReason: NullableString("should not be negative"),
			Property:      NullableString("offset"),
		}},
	}
	require.Equal(t, expected, received)
}

func TestObjectLifeline_Sort_Error(t *testing.T) {
	// request records with wrong sort param
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + gen.Reference().String() + "/states?sort_by=not_supported_sort")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.CodeValidationError
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	expected := server.CodeValidationError{
		Code:    NullableString(http.StatusText(http.StatusBadRequest)),
		Message: NullableString(InvalidParamsMessage),
		ValidationFailures: &[]server.CodeValidationFailures{{
			FailureReason: NullableString("should be 'index_desc' or 'index_asc'"),
			Property:      NullableString("sort_by"),
		}},
	}
	require.Equal(t, expected, received)
}

func TestObjectLifeline_NoRecords(t *testing.T) {
	// request records for object without records
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + gen.Reference().String() + "/states")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.RecordsResponse
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	require.EqualValues(t, 0, int(*received.Total))
	require.Len(t, *received.Result, 0)
}

func TestObjectLifeline_ReferenceFormat_Error(t *testing.T) {
	// request records with wrong format object reference
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + "not_valid_reference" + "/states")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.CodeValidationError
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	expected := server.CodeValidationError{
		Code:    NullableString(http.StatusText(http.StatusBadRequest)),
		Message: NullableString(InvalidParamsMessage),
		ValidationFailures: &[]server.CodeValidationFailures{{
			FailureReason: NullableString("wrong format"),
			Property:      NullableString("object_reference"),
		}},
	}

	require.Equal(t, expected, received)
}

func TestObjectLifeline_ReferenceEmpty_Error(t *testing.T) {
	// request records with empty object reference
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + "  " + "/states")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.CodeValidationError
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	expected := server.CodeValidationError{
		Code:    NullableString(http.StatusText(http.StatusBadRequest)),
		Message: NullableString(InvalidParamsMessage),
		ValidationFailures: &[]server.CodeValidationFailures{{
			FailureReason: NullableString("empty reference"),
			Property:      NullableString("object_reference"),
		}},
	}

	require.Equal(t, expected, received)
}

func TestObjectLifeline_Index_Error(t *testing.T) {
	// request records with wrong format from_index param
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + gen.Reference().String() + "/states?from_index=not_valid_index")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.CodeValidationError
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	expected := server.CodeValidationError{
		Code:    NullableString(http.StatusText(http.StatusBadRequest)),
		Message: NullableString(InvalidParamsMessage),
		ValidationFailures: &[]server.CodeValidationFailures{{
			FailureReason: NullableString("invalid"),
			Property:      NullableString("from_index"),
		}},
	}

	require.Equal(t, expected, received)
}

func TestPulse_HappyPath(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

	// insert pulses
	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	notExpectedPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, notExpectedPulse)
	require.NoError(t, err)

	// request pulse for pulseNumber
	resp, err := http.Get("http://" + apihost + fmt.Sprintf("/api/v1/pulses/%d", pulse.PulseNumber))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.PulseResponse
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.EqualValues(t, pulse.PulseNumber, *received.PulseNumber)
	require.False(t, *received.IsComplete)
	require.EqualValues(t, 0, *received.JetDropAmount)
	require.EqualValues(t, 0, *received.RecordAmount)
}

func TestPulse_PulseWithRecords(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

	// insert data
	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	s := storage.NewStorage(testDB)

	_ = testutils.InitJetDropWithRecords(t, s, 5, pulse)
	_ = testutils.InitJetDropWithRecords(t, s, 1, pulse)

	// request pulse for pulseNumber
	resp, err := http.Get("http://" + apihost + fmt.Sprintf("/api/v1/pulses/%d", pulse.PulseNumber))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.PulseResponse
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.EqualValues(t, pulse.PulseNumber, *received.PulseNumber)
	require.False(t, *received.IsComplete)
	require.EqualValues(t, 2, *received.JetDropAmount)
	require.EqualValues(t, 6, *received.RecordAmount)
}

func TestPulse_Pulse_NotExist(t *testing.T) {
	// request pulse for not existed pulse number
	resp, err := http.Get("http://" + apihost + fmt.Sprintf("/api/v1/pulses/%d", gen.PulseNumber()))
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	err = resp.Body.Close()
	require.NoError(t, err)
}

func TestPulse_Pulse_WrongFormat(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/v1/pulses/" + "wrong_type")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	err = resp.Body.Close()
	require.NoError(t, err)
}

func TestPulse_Pulse_GreaterThanMax(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/v1/pulses/" + string(math.MaxInt64) + "1")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	err = resp.Body.Close()
	require.NoError(t, err)
}

func TestPulses_HappyPath(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

	// insert pulses
	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	secondPulse, err := testutils.InitPulseDB()
	secondPulse.PulseNumber = pulse.PulseNumber + 10
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)

	// request pulses
	resp, err := http.Get("http://" + apihost + "/api/v1/pulses")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.PulsesResponse
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, *received.Result, 2)
	require.EqualValues(t, secondPulse.PulseNumber, *(*received.Result)[0].PulseNumber)
	require.EqualValues(t, pulse.PulseNumber, *(*received.Result)[1].PulseNumber)
	require.EqualValues(t, 2, *received.Total)
}

func TestPulses_OnePulse(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

	// insert pulse
	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)

	// request pulses
	resp, err := http.Get("http://" + apihost + "/api/v1/pulses")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.PulsesResponse
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, *received.Result, 1)
	require.EqualValues(t, pulse.PulseNumber, *(*received.Result)[0].PulseNumber)
	require.Nil(t, (*received.Result)[0].PrevPulseNumber)
	require.Nil(t, (*received.Result)[0].NextPulseNumber)
	require.EqualValues(t, 1, *received.Total)
}

func TestPulses_PulsesWithRecords(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

	s := storage.NewStorage(testDB)
	// insert data
	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)

	jetDrop1 := testutils.InitJetDropWithRecords(t, s, 6, pulse)
	jetDrop2 := testutils.InitJetDropWithRecords(t, s, 2, pulse)

	secondPulse, err := testutils.InitPulseDB()
	secondPulse.PulseNumber = pulse.PulseNumber + 10
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)
	jetDrop3 := testutils.InitJetDropWithRecords(t, s, 3, secondPulse)

	// request pulses
	resp, err := http.Get("http://" + apihost + "/api/v1/pulses")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.PulsesResponse
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, *received.Result, 2)
	require.EqualValues(t, secondPulse.PulseNumber, *(*received.Result)[0].PulseNumber)
	require.EqualValues(t, 1, *(*received.Result)[0].JetDropAmount)
	require.EqualValues(t, jetDrop3.RecordAmount, *(*received.Result)[0].RecordAmount)
	require.EqualValues(t, pulse.PulseNumber, *(*received.Result)[1].PulseNumber)
	require.EqualValues(t, 2, *(*received.Result)[1].JetDropAmount)
	require.EqualValues(t, jetDrop1.RecordAmount+jetDrop2.RecordAmount, *(*received.Result)[1].RecordAmount)
	require.EqualValues(t, 2, *received.Total)
}

func TestPulses_Empty(t *testing.T) {
	// request pulse from empty db
	resp, err := http.Get("http://" + apihost + "/api/v1/pulses")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.PulsesResponse
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Empty(t, received.Result)
	require.EqualValues(t, 0, *received.Total)
}

func TestPulses_Limit_Error(t *testing.T) {
	// request pulses with too big limit
	resp, err := http.Get("http://" + apihost + "/api/v1/pulses?limit=200000000")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.CodeValidationError
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	expected := server.CodeValidationError{
		Code:    NullableString(http.StatusText(http.StatusBadRequest)),
		Message: NullableString(InvalidParamsMessage),
		ValidationFailures: &[]server.CodeValidationFailures{{
			FailureReason: NullableString("should be in range [1, 1000]"),
			Property:      NullableString("limit"),
		}},
	}
	require.Equal(t, expected, received)
}

func TestPulses_Offset_Error(t *testing.T) {
	// request pulses with negative offset
	resp, err := http.Get("http://" + apihost + "/api/v1/pulses?offset=-10")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.CodeValidationError
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	expected := server.CodeValidationError{
		Code:    NullableString(http.StatusText(http.StatusBadRequest)),
		Message: NullableString(InvalidParamsMessage),
		ValidationFailures: &[]server.CodeValidationFailures{{
			FailureReason: NullableString("should not be negative"),
			Property:      NullableString("offset"),
		}},
	}
	require.Equal(t, expected, received)
}

func TestPulses_Several_Errors(t *testing.T) {
	// request pulses with negative offset
	resp, err := http.Get("http://" + apihost + "/api/v1/pulses?limit=200000000&offset=-10&from_pulse_number=0")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.CodeValidationError
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	expected := server.CodeValidationError{
		Code:    NullableString(http.StatusText(http.StatusBadRequest)),
		Message: NullableString(InvalidParamsMessage),
		ValidationFailures: &[]server.CodeValidationFailures{{
			FailureReason: NullableString("should be in range [1, 1000]"),
			Property:      NullableString("limit"),
		}, {
			FailureReason: NullableString("should not be negative"),
			Property:      NullableString("offset"),
		}, {
			FailureReason: NullableString("invalid"),
			Property:      NullableString("pulse"),
		}},
	}
	require.Equal(t, expected, received)
}

func TestPulses_FromPulseNumber(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

	// insert pulses
	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	secondPulse, err := testutils.InitPulseDB()
	secondPulse.PulseNumber = pulse.PulseNumber + 10
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)

	// request pulses
	resp, err := http.Get("http://" + apihost + fmt.Sprintf("/api/v1/pulses?from_pulse_number=%d", pulse.PulseNumber))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.PulsesResponse
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, *received.Result, 1)
	require.EqualValues(t, pulse.PulseNumber, *(*received.Result)[0].PulseNumber)
	require.EqualValues(t, 1, *received.Total)
}

func TestPulses_TimestampRange(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

	// insert pulses
	firstPulse := models.Pulse{
		PulseNumber: 66666666,
		IsComplete:  false,
		Timestamp:   66666666,
	}
	err := testutils.CreatePulse(testDB, firstPulse)
	require.NoError(t, err)

	secondPulse := models.Pulse{
		PulseNumber: 66666667,
		IsComplete:  false,
		Timestamp:   66666667,
	}
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)

	thirdPulse := models.Pulse{
		PulseNumber: 66666668,
		IsComplete:  false,
		Timestamp:   66666668,
	}
	err = testutils.CreatePulse(testDB, thirdPulse)
	require.NoError(t, err)

	fourthPulse := models.Pulse{
		PulseNumber: 66666669,
		IsComplete:  false,
		Timestamp:   66666669,
	}
	err = testutils.CreatePulse(testDB, fourthPulse)
	require.NoError(t, err)

	// request pulses
	resp, err := http.Get("http://" + apihost +
		fmt.Sprintf("/api/v1/pulses?timestamp_lte=%d&timestamp_gte=%d", thirdPulse.PulseNumber, secondPulse.PulseNumber),
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.PulsesResponse
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, *received.Result, 2)
	require.EqualValues(t, thirdPulse.PulseNumber, *(*received.Result)[0].PulseNumber)
	require.EqualValues(t, secondPulse.PulseNumber, *(*received.Result)[1].PulseNumber)
	require.EqualValues(t, 2, *received.Total)
}

func TestPulses_PulseNumberFilters(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

	p1 := int64(66666666)
	p2 := p1 + 1
	p3 := p2 + 1
	p4 := p3 + 1
	p5 := p4 + 1
	// insert pulses
	firstPulse := models.Pulse{
		PulseNumber:     p1,
		IsComplete:      false,
		Timestamp:       66666666,
		NextPulseNumber: p2,
	}
	err := testutils.CreatePulse(testDB, firstPulse)
	require.NoError(t, err)

	secondPulse := models.Pulse{
		PulseNumber:     p2,
		IsComplete:      false,
		Timestamp:       66666667,
		PrevPulseNumber: p1,
		NextPulseNumber: p3,
	}
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)

	thirdPulse := models.Pulse{
		PulseNumber:     p3,
		IsComplete:      false,
		Timestamp:       66666668,
		PrevPulseNumber: p2,
		NextPulseNumber: p4,
	}
	err = testutils.CreatePulse(testDB, thirdPulse)
	require.NoError(t, err)

	fourthPulse := models.Pulse{
		PulseNumber:     66666669,
		IsComplete:      false,
		Timestamp:       66666669,
		PrevPulseNumber: p3,
		NextPulseNumber: p5,
	}
	err = testutils.CreatePulse(testDB, fourthPulse)
	require.NoError(t, err)

	t.Run("pulse_number_lte", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost +
			fmt.Sprintf("/api/v1/pulses?pulse_number_lte=%d", thirdPulse.PulseNumber),
		)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.PulsesResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Len(t, *received.Result, 3)
		require.EqualValues(t, 3, *received.Total)
		require.EqualValues(t, thirdPulse.PulseNumber, *(*received.Result)[0].PulseNumber)
		require.EqualValues(t, secondPulse.PulseNumber, *(*received.Result)[1].PulseNumber)
		require.EqualValues(t, firstPulse.PulseNumber, *(*received.Result)[2].PulseNumber)

		require.Nil(t, (*received.Result)[2].PrevPulseNumber)
		require.Equal(t, firstPulse.NextPulseNumber, *(*received.Result)[2].NextPulseNumber)
		require.Equal(t, secondPulse.PrevPulseNumber, *(*received.Result)[1].PrevPulseNumber)
		require.Equal(t, secondPulse.NextPulseNumber, *(*received.Result)[1].NextPulseNumber)
		require.Equal(t, thirdPulse.PrevPulseNumber, *(*received.Result)[0].PrevPulseNumber)
		require.Equal(t, thirdPulse.NextPulseNumber, *(*received.Result)[0].NextPulseNumber)
	})

	t.Run("pulse_number_lt", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost +
			fmt.Sprintf("/api/v1/pulses?pulse_number_lt=%d", thirdPulse.PulseNumber),
		)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.PulsesResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Len(t, *received.Result, 2)
		require.EqualValues(t, 2, *received.Total)
		require.EqualValues(t, secondPulse.PulseNumber, *(*received.Result)[0].PulseNumber)
		require.EqualValues(t, firstPulse.PulseNumber, *(*received.Result)[1].PulseNumber)

		require.Nil(t, (*received.Result)[1].PrevPulseNumber)
		require.Equal(t, firstPulse.NextPulseNumber, *(*received.Result)[1].NextPulseNumber)
		require.Equal(t, secondPulse.PrevPulseNumber, *(*received.Result)[0].PrevPulseNumber)
		require.Equal(t, secondPulse.NextPulseNumber, *(*received.Result)[0].NextPulseNumber)
	})

	t.Run("pulse_number_gte", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost +
			fmt.Sprintf("/api/v1/pulses?pulse_number_gte=%d", thirdPulse.PulseNumber),
		)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.PulsesResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Len(t, *received.Result, 2)
		require.EqualValues(t, 2, *received.Total)
		require.EqualValues(t, fourthPulse.PulseNumber, *(*received.Result)[0].PulseNumber)
		require.EqualValues(t, thirdPulse.PulseNumber, *(*received.Result)[1].PulseNumber)

		require.Equal(t, thirdPulse.PrevPulseNumber, *(*received.Result)[1].PrevPulseNumber)
		require.Equal(t, thirdPulse.NextPulseNumber, *(*received.Result)[1].NextPulseNumber)
		require.Equal(t, fourthPulse.PrevPulseNumber, *(*received.Result)[0].PrevPulseNumber)
		require.Nil(t, (*received.Result)[0].NextPulseNumber)
	})

	t.Run("pulse_number_gt", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost +
			fmt.Sprintf("/api/v1/pulses?pulse_number_gt=%d", thirdPulse.PulseNumber),
		)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.PulsesResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Len(t, *received.Result, 1)
		require.EqualValues(t, 1, *received.Total)
		require.EqualValues(t, fourthPulse.PulseNumber, *(*received.Result)[0].PulseNumber)

		require.Equal(t, fourthPulse.PrevPulseNumber, *(*received.Result)[0].PrevPulseNumber)
		require.Nil(t, (*received.Result)[0].NextPulseNumber)
	})

	t.Run("sort_by asc", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost +
			fmt.Sprintf("/api/v1/pulses?sort_by=%s", server.SortByPulseNumber_pulse_number_asc),
		)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.PulsesResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Len(t, *received.Result, 4)
		require.EqualValues(t, 4, *received.Total)
		require.EqualValues(t, firstPulse.PulseNumber, *(*received.Result)[0].PulseNumber)
		require.EqualValues(t, secondPulse.PulseNumber, *(*received.Result)[1].PulseNumber)
		require.EqualValues(t, thirdPulse.PulseNumber, *(*received.Result)[2].PulseNumber)
		require.EqualValues(t, fourthPulse.PulseNumber, *(*received.Result)[3].PulseNumber)

		require.Nil(t, (*received.Result)[0].PrevPulseNumber)
		require.Equal(t, firstPulse.NextPulseNumber, *(*received.Result)[0].NextPulseNumber)
		require.Equal(t, secondPulse.PrevPulseNumber, *(*received.Result)[1].PrevPulseNumber)
		require.Equal(t, secondPulse.NextPulseNumber, *(*received.Result)[1].NextPulseNumber)
		require.Equal(t, thirdPulse.PrevPulseNumber, *(*received.Result)[2].PrevPulseNumber)
		require.Equal(t, thirdPulse.NextPulseNumber, *(*received.Result)[2].NextPulseNumber)
		require.Equal(t, fourthPulse.PrevPulseNumber, *(*received.Result)[3].PrevPulseNumber)
		require.Nil(t, (*received.Result)[3].NextPulseNumber)
	})

	t.Run("sort_by desc", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost +
			fmt.Sprintf("/api/v1/pulses?sort_by=%s", server.SortByPulseNumber_pulse_number_desc),
		)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.PulsesResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Len(t, *received.Result, 4)
		require.EqualValues(t, 4, *received.Total)
		require.EqualValues(t, firstPulse.PulseNumber, *(*received.Result)[3].PulseNumber)
		require.EqualValues(t, secondPulse.PulseNumber, *(*received.Result)[2].PulseNumber)
		require.EqualValues(t, thirdPulse.PulseNumber, *(*received.Result)[1].PulseNumber)
		require.EqualValues(t, fourthPulse.PulseNumber, *(*received.Result)[0].PulseNumber)

		require.Nil(t, (*received.Result)[3].PrevPulseNumber)
		require.Equal(t, firstPulse.NextPulseNumber, *(*received.Result)[3].NextPulseNumber)
		require.Equal(t, secondPulse.PrevPulseNumber, *(*received.Result)[2].PrevPulseNumber)
		require.Equal(t, secondPulse.NextPulseNumber, *(*received.Result)[2].NextPulseNumber)
		require.Equal(t, thirdPulse.PrevPulseNumber, *(*received.Result)[1].PrevPulseNumber)
		require.Equal(t, thirdPulse.NextPulseNumber, *(*received.Result)[1].NextPulseNumber)
		require.Equal(t, fourthPulse.PrevPulseNumber, *(*received.Result)[0].PrevPulseNumber)
		require.Nil(t, (*received.Result)[0].NextPulseNumber)
	})
}

func TestServer_JetDropsByPulseNumber(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

		// insert records
		pulse, err := testutils.InitPulseDB()
		require.NoError(t, err)
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(t, err)

		jetDrop2 := testutils.InitJetDropDB(pulse)
		jetID2 := jet.NewIDFromString("001")
		jetDrop2.JetID = converter.JetIDToString(jetID2)
		err = testutils.CreateJetDrop(testDB, jetDrop2)
		require.NoError(t, err)

		jetDrop1 := testutils.InitJetDropDB(pulse)
		jetID1 := jet.NewIDFromString("000")
		jetDrop1.JetID = converter.JetIDToString(jetID1)
		err = testutils.CreateJetDrop(testDB, jetDrop1)
		require.NoError(t, err)

		jetDrop3 := testutils.InitJetDropDB(pulse)
		jetID3 := jet.NewIDFromString("010")
		jetDrop3.JetID = converter.JetIDToString(jetID3)
		err = testutils.CreateJetDrop(testDB, jetDrop3)
		require.NoError(t, err)

		resp, err := http.Get("http://" + apihost + "/api/v1/pulses/" + strconv.FormatInt(pulse.PulseNumber, 10) + "/jet-drops")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.JetDropsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.EqualValues(t, 3, int(*received.Total))
		require.Len(t, *received.Result, 3)
		// check asc order by default
		require.Equal(t, models.NewJetDropID(jetDrop1.JetID, pulse.PulseNumber).ToString(), *(*received.Result)[0].JetDropId)
		require.Equal(t, models.NewJetDropID(jetDrop2.JetID, pulse.PulseNumber).ToString(), *(*received.Result)[1].JetDropId)
		require.Equal(t, models.NewJetDropID(jetDrop3.JetID, pulse.PulseNumber).ToString(), *(*received.Result)[2].JetDropId)
	})

	t.Run("happy with limit", func(t *testing.T) {
		defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

		// insert records
		pulse, err := testutils.InitPulseDB()
		require.NoError(t, err)
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(t, err)
		jetDrop1 := testutils.InitJetDropDB(pulse)
		jetID1 := jet.NewIDFromString("000")
		jetDrop1.JetID = converter.JetIDToString(jetID1)
		err = testutils.CreateJetDrop(testDB, jetDrop1)
		require.NoError(t, err)

		jetDrop2 := testutils.InitJetDropDB(pulse)
		jetID2 := jet.NewIDFromString("001")
		jetDrop2.JetID = converter.JetIDToString(jetID2)
		err = testutils.CreateJetDrop(testDB, jetDrop2)
		require.NoError(t, err)

		jetDrop3 := testutils.InitJetDropDB(pulse)
		jetID3 := jet.NewIDFromString("010")
		jetDrop3.JetID = converter.JetIDToString(jetID3)
		err = testutils.CreateJetDrop(testDB, jetDrop3)
		require.NoError(t, err)

		resp, err := http.Get("http://" + apihost + "/api/v1/pulses/" + strconv.FormatInt(pulse.PulseNumber, 10) + "/jet-drops?limit=2")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.JetDropsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.EqualValues(t, 3, int(*received.Total))
		require.Len(t, *received.Result, 2)
		// check asc order by default
		require.Equal(t, models.NewJetDropID(jetDrop1.JetID, int64(pulse.PulseNumber)).ToString(), *(*received.Result)[0].JetDropId)
		require.Equal(t, models.NewJetDropID(jetDrop2.JetID, int64(pulse.PulseNumber)).ToString(), *(*received.Result)[1].JetDropId)
	})

	t.Run("happy with all params", func(t *testing.T) {
		defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

		// insert records
		pulse, err := testutils.InitPulseDB()
		require.NoError(t, err)
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(t, err)
		jetDrop1 := testutils.InitJetDropDB(pulse)
		jetID1 := jet.NewIDFromString("000")
		jetDrop1.JetID = converter.JetIDToString(jetID1)
		err = testutils.CreateJetDrop(testDB, jetDrop1)
		require.NoError(t, err)

		jetDrop2 := testutils.InitJetDropDB(pulse)
		jetID2 := jet.NewIDFromString("001")
		jetDrop2.JetID = converter.JetIDToString(jetID2)
		err = testutils.CreateJetDrop(testDB, jetDrop2)
		require.NoError(t, err)

		jetDrop3 := testutils.InitJetDropDB(pulse)
		jetID3 := jet.NewIDFromString("010")
		jetDrop3.JetID = converter.JetIDToString(jetID3)
		err = testutils.CreateJetDrop(testDB, jetDrop3)
		require.NoError(t, err)
		jetDropID3 := models.NewJetDropID(jetDrop3.JetID, int64(pulse.PulseNumber)).ToString()

		jetDrop4 := testutils.InitJetDropDB(pulse)
		jetID4 := jet.NewIDFromString("011")
		jetDrop4.JetID = converter.JetIDToString(jetID4)
		err = testutils.CreateJetDrop(testDB, jetDrop4)
		require.NoError(t, err)

		jetDrop5 := testutils.InitJetDropDB(pulse)
		jetID5 := jet.NewIDFromString("100")
		jetDrop5.JetID = converter.JetIDToString(jetID5)
		err = testutils.CreateJetDrop(testDB, jetDrop5)
		require.NoError(t, err)

		jetDrop6 := testutils.InitJetDropDB(pulse)
		jetID6 := jet.NewIDFromString("101")
		jetDrop6.JetID = converter.JetIDToString(jetID6)
		err = testutils.CreateJetDrop(testDB, jetDrop6)
		require.NoError(t, err)

		resp, err := http.Get(
			"http://" + apihost + "/api/v1/pulses/" +
				strconv.FormatInt(pulse.PulseNumber, 10) +
				"/jet-drops?limit=2&offset=2&from_jet_drop_id=" +
				jetDropID3,
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.JetDropsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.EqualValues(t, 4, int(*received.Total))
		require.Len(t, *received.Result, 2)
		// check asc order by default
		require.Equal(t, models.NewJetDropID(jetDrop5.JetID, int64(pulse.PulseNumber)).ToString(), *(*received.Result)[0].JetDropId)
		require.Equal(t, models.NewJetDropID(jetDrop6.JetID, int64(pulse.PulseNumber)).ToString(), *(*received.Result)[1].JetDropId)
	})

	t.Run("error wrong jetdropid", func(t *testing.T) {
		pulse, err := testutils.InitPulseDB()
		require.NoError(t, err)
		resp, err := http.Get(
			"http://" + apihost + "/api/v1/pulses/" +
				strconv.FormatInt(pulse.PulseNumber, 10) +
				"/jet-drops?from_jet_drop_id=" +
				"test",
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		var received server.CodeValidationError
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		expected := server.CodeValidationFailures{
			FailureReason: NullableString("invalid"),
			Property:      NullableString("jet drop id"),
		}
		e := *received.ValidationFailures
		require.Equal(t, expected, e[0])

		resp, err = http.Get(
			"http://" + apihost + "/api/v1/pulses/" +
				strconv.FormatInt(pulse.PulseNumber, 10) +
				"/jet-drops?from_jet_drop_id=" +
				"10076767676",
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		bodyBytes, err = ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		expected = server.CodeValidationFailures{
			FailureReason: NullableString("invalid"),
			Property:      NullableString("jet drop id"),
		}
		e = *received.ValidationFailures
		require.Equal(t, expected, e[0])

		resp, err = http.Get(
			"http://" + apihost + "/api/v1/pulses/" +
				strconv.FormatInt(pulse.PulseNumber, 10) +
				"/jet-drops?from_jet_drop_id=" +
				"76767676:1000",
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		bodyBytes, err = ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		expected = server.CodeValidationFailures{
			FailureReason: NullableString("invalid"),
			Property:      NullableString("jet drop id"),
		}
		e = *received.ValidationFailures
		require.Equal(t, expected, e[0])
	})

	t.Run("error wrong jetdropid, pulse, limit", func(t *testing.T) {
		resp, err := http.Get(
			"http://" + apihost + "/api/v1/pulses/" +
				"100" +
				"/jet-drops?from_jet_drop_id=23423:90000&limit=2000",
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		var received server.CodeValidationError
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		expected := []server.CodeValidationFailures{
			{
				FailureReason: NullableString("should be in range [1, 1000]"),
				Property:      NullableString("limit"),
			},
			{
				FailureReason: NullableString("should not be negative"),
				Property:      NullableString("offset"),
			},
			{
				FailureReason: NullableString("invalid"),
				Property:      NullableString("pulse"),
			},
			{
				FailureReason: NullableString("invalid"),
				Property:      NullableString("jet drop id"),
			},
		}
		e := *received.ValidationFailures
		require.Contains(t, expected, e[0])
		require.Contains(t, expected, e[1])
		require.Contains(t, expected, e[2])

	})

	t.Run("error wrong pulse", func(t *testing.T) {
		resp, err := http.Get(
			"http://" + apihost + "/api/v1/pulses/" +
				"wrong-pulse" +
				"/jet-drops",
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		var received server.CodeValidationError
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Contains(t, *received.Message, "wrong-pulse")
	})

	t.Run("error wrong limit", func(t *testing.T) {
		pulse, err := testutils.InitPulseDB()
		require.NoError(t, err)
		resp, err := http.Get(
			"http://" + apihost + "/api/v1/pulses/" +
				strconv.FormatInt(pulse.PulseNumber, 10) +
				"/jet-drops?limit=" + "we248934h9h'`;",
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		var received server.CodeValidationError
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Contains(t, *received.Message, "we248934h9h")
	})

	t.Run("ok empty pulse", func(t *testing.T) {
		resp, err := http.Get(
			"http://" + apihost + "/api/v1/pulses/" +
				"383615209" +
				"/jet-drops",
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.JetDropsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.EqualValues(t, 0, int(*received.Total))
		require.Len(t, *received.Result, 0)
	})
}

func TestSearch_Pulse(t *testing.T) {
	pulseNumber := gen.PulseNumber()
	// search by pulse
	resp, err := http.Get("http://" + apihost + "/api/v1/search?value=" + pulseNumber.String())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.SearchPulse
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.EqualValues(t, "pulse", *received.Type)
	require.EqualValues(t, pulseNumber, *received.Meta.PulseNumber)
}

func TestSearch_Pulse_WrongValue(t *testing.T) {
	resp, err := http.Get("http://" + apihost + fmt.Sprintf("/api/v1/search?value=%d", pulse.MinTimePulse-1))
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.CodeValidationError
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, *received.ValidationFailures, 1)
	require.EqualValues(t, "not valid pulse number", *(*received.ValidationFailures)[0].FailureReason)
	require.EqualValues(t, "value", *(*received.ValidationFailures)[0].Property)
}

func TestSearch_JetDrop(t *testing.T) {
	pulseNumber := gen.PulseNumber()
	jetDropID := fmt.Sprintf("101010:%s", pulseNumber.String())
	// search by jetDrop
	resp, err := http.Get("http://" + apihost + "/api/v1/search?value=" + jetDropID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.SearchJetDrop
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.EqualValues(t, "jet-drop", *received.Type)
	require.EqualValues(t, jetDropID, *received.Meta.JetDropId)
}

func TestSearch_Object(t *testing.T) {
	objRef := gen.Reference().String()
	// search by object reference
	resp, err := http.Get("http://" + apihost + "/api/v1/search?value=" + objRef)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.SearchLifeline
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.EqualValues(t, "lifeline", *received.Type)
	require.EqualValues(t, objRef, *received.Meta.ObjectReference)
}

func TestSearch_Record(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

	// insert records
	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	jetDrop := testutils.InitJetDropDB(pulse)
	err = testutils.CreateJetDrop(testDB, jetDrop)
	require.NoError(t, err)

	objRef := gen.Reference()

	genRecords := testutils.OrderedRecords(t, testDB, jetDrop, *objRef.GetLocal(), 3)
	testutils.OrderedRecords(t, testDB, jetDrop, gen.ID(), 3)

	recRef := genRecords[1]
	// search by record reference
	resp, err := http.Get("http://" + apihost + "/api/v1/search?value=" + insolar.NewRecordReference(*insolar.NewIDFromBytes(recRef.Reference)).String())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.SearchState
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.EqualValues(t, "record", *received.Type)
	require.EqualValues(t, objRef.String(), *received.Meta.ObjectReference)
	require.EqualValues(t, fmt.Sprintf("%d:%d", recRef.PulseNumber, recRef.Order), *received.Meta.Index)
}

func TestSearch_Record_NotExist(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/v1/search?value=" + gen.RecordReference().String())
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.CodeValidationError
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, *received.ValidationFailures, 1)
	require.EqualValues(t, "record reference not found", *(*received.ValidationFailures)[0].FailureReason)
	require.EqualValues(t, "value", *(*received.ValidationFailures)[0].Property)
}

func TestSearch_NoValue(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/v1/search")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	err = resp.Body.Close()
	require.NoError(t, err)
}

func TestSearch_InvalidValue(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/v1/search?value=not_valid_value")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.CodeValidationError
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, *received.ValidationFailures, 1)
	require.EqualValues(t, "is neither pulse number, jet drop id nor reference", *(*received.ValidationFailures)[0].FailureReason)
	require.EqualValues(t, "value", *(*received.ValidationFailures)[0].Property)
}

func TestServer_JetDropByID(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

		// insert records
		pulse, err := testutils.InitPulseDB()
		require.NoError(t, err)
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(t, err)

		jetDrop2 := testutils.InitJetDropDB(pulse)
		jetID2 := jet.NewIDFromString("001")
		jetDrop2.JetID = converter.JetIDToString(jetID2)
		err = testutils.CreateJetDrop(testDB, jetDrop2)
		require.NoError(t, err)

		jetDrop1 := testutils.InitJetDropDB(pulse)
		jetID1 := jet.NewIDFromString("000")
		jetDrop1.JetID = converter.JetIDToString(jetID1)
		err = testutils.CreateJetDrop(testDB, jetDrop1)
		require.NoError(t, err)
		jetDropID1 := models.NewJetDropID(jetDrop1.JetID, int64(pulse.PulseNumber))

		jetDrop3 := testutils.InitJetDropDB(pulse)
		jetID3 := jet.NewIDFromString("010")
		jetDrop3.JetID = converter.JetIDToString(jetID3)
		err = testutils.CreateJetDrop(testDB, jetDrop3)
		require.NoError(t, err)

		resp, err := http.Get("http://" + apihost + "/api/v1/jet-drops/" + jetDropID1.ToString())
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.JetDropResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, jetDropID1.ToString(), *received.JetDropId)
		require.Equal(t, jetDropID1.JetIDToString(), *received.JetId)
		require.Equal(t, base64.StdEncoding.EncodeToString(jetDrop1.Hash), *received.Hash)
	})

	t.Run("happy first jet", func(t *testing.T) {
		defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

		// insert records
		pulse, err := testutils.InitPulseDB()
		require.NoError(t, err)
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(t, err)

		jetDrop1 := testutils.InitJetDropDB(pulse)
		jetID1 := jet.NewIDFromString("")
		jetDrop1.JetID = converter.JetIDToString(jetID1)
		err = testutils.CreateJetDrop(testDB, jetDrop1)
		require.NoError(t, err)
		jetDropID1 := models.NewJetDropID(jetDrop1.JetID, int64(pulse.PulseNumber))

		resp, err := http.Get("http://" + apihost + "/api/v1/jet-drops/" + jetDropID1.ToString())
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.JetDropResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, jetDropID1.ToString(), *received.JetDropId)
		require.Equal(t, jetDropID1.JetIDToString(), *received.JetId)
		require.Equal(t, base64.StdEncoding.EncodeToString(jetDrop1.Hash), *received.Hash)
	})

	t.Run("happy with tree", func(t *testing.T) {
		t.Skip("uncomment after tree will be ready")
		defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

		// insert records
		pulse, err := testutils.InitPulseDB()
		require.NoError(t, err)
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(t, err)

		jetDrop2 := testutils.InitJetDropDB(pulse)
		jetID2 := jet.NewIDFromString("001")
		jetDrop2.JetID = converter.JetIDToString(jetID2)
		err = testutils.CreateJetDrop(testDB, jetDrop2)
		require.NoError(t, err)
		jetDropID2 := models.NewJetDropID(jetDrop2.JetID, int64(pulse.PulseNumber)).ToString()

		jetDrop1 := testutils.InitJetDropDB(pulse)
		jetID1 := jet.NewIDFromString("000")
		jetDrop1.JetID = converter.JetIDToString(jetID1)
		err = testutils.CreateJetDrop(testDB, jetDrop1)
		require.NoError(t, err)
		jetDropID1 := models.NewJetDropID(jetDrop1.JetID, int64(pulse.PulseNumber)).ToString()

		jetDrop3 := testutils.InitJetDropDB(pulse)
		jetID3 := jet.NewIDFromString("010")
		jetDrop3.JetID = converter.JetIDToString(jetID3)
		err = testutils.CreateJetDrop(testDB, jetDrop3)
		require.NoError(t, err)
		jetDropID3 := models.NewJetDropID(jetDrop3.JetID, int64(pulse.PulseNumber)).ToString()

		// create next pulse and jet drops
		pulse.PulseNumber = pulse.PulseNumber + 10
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(t, err)

		jetDrop4 := testutils.InitJetDropDB(pulse)
		jetID4 := jet.NewIDFromString("001")
		jetDrop4.JetID = converter.JetIDToString(jetID4)
		err = testutils.CreateJetDrop(testDB, jetDrop4)
		require.NoError(t, err)
		jetDropID4 := models.NewJetDropID(jetDrop4.JetID, int64(pulse.PulseNumber)).ToString()

		jetDrop5 := testutils.InitJetDropDB(pulse)
		jetID5 := jet.NewIDFromString("000")
		jetDrop5.JetID = converter.JetIDToString(jetID5)
		err = testutils.CreateJetDrop(testDB, jetDrop5)
		require.NoError(t, err)
		jetDropID5 := models.NewJetDropID(jetDrop5.JetID, int64(pulse.PulseNumber)).ToString()

		jetDrop6 := testutils.InitJetDropDB(pulse)
		jetID6 := jet.NewIDFromString("010")
		jetDrop6.JetID = converter.JetIDToString(jetID6)
		err = testutils.CreateJetDrop(testDB, jetDrop6)
		require.NoError(t, err)
		jetDropID6 := models.NewJetDropID(jetDrop6.JetID, int64(pulse.PulseNumber)).ToString()

		// create next pulse and jet drops
		pulse.PulseNumber = pulse.PulseNumber + 10
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(t, err)

		jetDrop7 := testutils.InitJetDropDB(pulse)
		jetID7 := jet.NewIDFromString("001")
		jetDrop7.JetID = converter.JetIDToString(jetID7)
		err = testutils.CreateJetDrop(testDB, jetDrop7)
		require.NoError(t, err)
		jetDropID7 := models.NewJetDropID(jetDrop7.JetID, int64(pulse.PulseNumber)).ToString()

		jetDrop8 := testutils.InitJetDropDB(pulse)
		jetID8 := jet.NewIDFromString("000")
		jetDrop8.JetID = converter.JetIDToString(jetID8)
		err = testutils.CreateJetDrop(testDB, jetDrop8)
		require.NoError(t, err)
		jetDropID8 := models.NewJetDropID(jetDrop8.JetID, int64(pulse.PulseNumber)).ToString()

		jetDrop9 := testutils.InitJetDropDB(pulse)
		jetID9 := jet.NewIDFromString("010")
		jetDrop9.JetID = converter.JetIDToString(jetID9)
		err = testutils.CreateJetDrop(testDB, jetDrop9)
		require.NoError(t, err)
		jetDropID9 := models.NewJetDropID(jetDrop9.JetID, int64(pulse.PulseNumber)).ToString()

		resp, err := http.Get("http://" + apihost + "/api/v1/jet-drops/" + jetDropID1)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.JetDropResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, jetDropID1, *received.JetDropId)
		require.Equal(t, string(jetDrop1.Hash), *received.Hash)
		expectedPrev := []string{
			jetDropID1, jetDropID2, jetDropID3, jetDropID4,
		}
		expectedNext := []string{
			jetDropID5, jetDropID6, jetDropID7, jetDropID8, jetDropID9,
		}

		require.Contains(t, expectedPrev, *received.PrevJetDropId)
		require.Contains(t, expectedNext, *received.NextJetDropId)
	})

	t.Run("url encoded", func(t *testing.T) {
		defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

		// insert records
		pulse, err := testutils.InitPulseDB()
		require.NoError(t, err)
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(t, err)

		jetDrop1 := testutils.InitJetDropDB(pulse)
		jetID1 := jet.NewIDFromString("000100")
		jetDrop1.JetID = converter.JetIDToString(jetID1)
		err = testutils.CreateJetDrop(testDB, jetDrop1)
		require.NoError(t, err)
		jetDropID1 := models.NewJetDropID(jetDrop1.JetID, int64(pulse.PulseNumber))

		urlencoded := url.QueryEscape(jetDropID1.ToString())
		resp, err := http.Get("http://" + apihost + "/api/v1/jet-drops/" + urlencoded)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.JetDropResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, jetDropID1.ToString(), *received.JetDropId)
		require.Equal(t, jetDropID1.JetIDToString(), *received.JetId)
		require.Equal(t, base64.StdEncoding.EncodeToString(jetDrop1.Hash), *received.Hash)
	})

	t.Run("error wrong id", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/v1/jet-drops/1000:dfg")
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.CodeValidationError
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		expected := server.CodeValidationFailures{
			FailureReason: NullableString("invalid: wrong jet drop id format"),
			Property:      NullableString("jet drop id"),
		}
		e := *received.ValidationFailures
		require.Equal(t, expected, e[0])
	})

	t.Run("error not existent id", func(t *testing.T) {
		defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

		// insert records
		pulse, err := testutils.InitPulseDB()
		require.NoError(t, err)
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(t, err)

		jetDrop1 := testutils.InitJetDropDB(pulse)
		jetDropID1 := models.NewJetDropID(jetDrop1.JetID, int64(pulse.PulseNumber)).ToString()

		resp, err := http.Get("http://" + apihost + "/api/v1/jet-drops/" + jetDropID1)
		require.NoError(t, err)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode, string(bodyBytes))
	})
}

func TestServer_JetDropsByJetID_NextPrevTests(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	totalCount := 7
	jetID, preparedJetDrops, preparedPulses := testutils.GenerateJetDropsWithSomeJetID(t, totalCount)
	for i := 1; i <= len(preparedJetDrops)-1; i++ {
		preparedJetDrops[i].FirstPrevHash = preparedJetDrops[i-1].Hash
	}
	err := testutils.CreatePulses(testDB, preparedPulses)
	require.NoError(t, err)
	err = testutils.CreateJetDrops(testDB, preparedJetDrops)
	require.NoError(t, err)

	checkOkReturningResponse := func(t *testing.T, resp *http.Response, respErr error) server.JetDropsResponse {
		require.NoError(t, err)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(bodyBytes))
		var received server.JetDropsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		return received
	}

	// default sort by pulses desc
	t.Run("pulseNumberGte, pulseNumberLte", func(t *testing.T) {
		expectedCount := totalCount - 2
		pulseNumberGte := preparedPulses[1].PulseNumber
		pulseNumberLte := preparedPulses[totalCount-2].PulseNumber
		query := fmt.Sprintf("pulse_number_gte=%d&pulse_number_lte=%d", pulseNumberGte, pulseNumberLte)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		var expected []server.JetDrop
		for i := len(preparedJetDrops) - 2; i >= 1; i-- {
			expected = append(
				expected,
				JetDropToAPI(
					preparedJetDrops[i],
					[]server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops[i-1])},
					[]server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops[i+1])},
				),
			)
		}

		require.EqualValues(t, expected, *response.Result)
	})

	t.Run("pulseNumberGte, pulseNumberLte, sort_by=pulse_number_asc,jet_id_desc", func(t *testing.T) {
		expectedCount := totalCount - 2
		pulseNumberGte := preparedPulses[1].PulseNumber
		pulseNumberLte := preparedPulses[totalCount-2].PulseNumber
		query := fmt.Sprintf("sort_by=pulse_number_asc,jet_id_desc&pulse_number_gte=%d&pulse_number_lte=%d", pulseNumberGte, pulseNumberLte)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		var expected []server.JetDrop
		for i := 1; i <= len(preparedJetDrops)-2; i++ {
			expected = append(
				expected,
				JetDropToAPI(
					preparedJetDrops[i],
					[]server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops[i-1])},
					[]server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops[i+1])},
				),
			)
		}

		require.EqualValues(t, expected, *response.Result)
	})

	t.Run("pulseNumberGte", func(t *testing.T) {
		expectedCount := totalCount - 1
		pulseNumberGte := preparedPulses[1].PulseNumber
		query := fmt.Sprintf("pulse_number_gte=%d", pulseNumberGte)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		var expected []server.JetDrop
		for i := len(preparedJetDrops) - 1; i >= 1; i-- {
			next := []server.NextPrevJetDrop{}
			if i+1 < len(preparedJetDrops) {
				next = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops[i+1])}
			}
			expected = append(
				expected,
				JetDropToAPI(
					preparedJetDrops[i],
					[]server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops[i-1])},
					next,
				),
			)
		}

		require.EqualValues(t, expected, *response.Result)
	})

	t.Run("pulseNumberGte, sort_by=pulse_number_asc,jet_id_desc", func(t *testing.T) {
		expectedCount := totalCount - 1
		pulseNumberGte := preparedPulses[1].PulseNumber
		query := fmt.Sprintf("sort_by=pulse_number_asc,jet_id_desc&pulse_number_gte=%d", pulseNumberGte)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		var expected []server.JetDrop
		for i := 1; i <= len(preparedJetDrops)-1; i++ {
			next := []server.NextPrevJetDrop{}
			if i+1 < len(preparedJetDrops) {
				next = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops[i+1])}
			}
			expected = append(
				expected,
				JetDropToAPI(
					preparedJetDrops[i],
					[]server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops[i-1])},
					next,
				),
			)
		}

		require.EqualValues(t, expected, *response.Result)
	})

	t.Run("pulseNumberLte", func(t *testing.T) {
		expectedCount := totalCount - 1
		pulseNumberLte := preparedPulses[totalCount-2].PulseNumber
		query := fmt.Sprintf("pulse_number_lte=%d", pulseNumberLte)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		var expected []server.JetDrop
		for i := len(preparedJetDrops) - 2; i >= 0; i-- {
			next := []server.NextPrevJetDrop{}
			if i+1 < len(preparedJetDrops) {
				next = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops[i+1])}
			}
			prev := []server.NextPrevJetDrop{}
			if i-1 >= 0 {
				prev = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops[i-1])}
			}
			expected = append(
				expected,
				JetDropToAPI(
					preparedJetDrops[i],
					prev,
					next,
				),
			)
		}

		require.EqualValues(t, expected, *response.Result)
	})
}

func TestServer_JetDropsByJetID_NextPrevTests_Siblings(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	totalCount := 7
	// making different siblings in same pulses
	_, preparedJetDrops1, preparedPulses := testutils.GenerateJetDropsWithSomeJetID(t, totalCount)
	_, preparedJetDrops2, _ := testutils.GenerateJetDropsWithSomeJetID(t, totalCount)
	_, preparedJetDrops3, _ := testutils.GenerateJetDropsWithSomeJetID(t, totalCount)
	_, preparedJetDrops4, _ := testutils.GenerateJetDropsWithSomeJetID(t, totalCount)
	for i := 0; i <= len(preparedJetDrops1)-1; i++ {
		preparedJetDrops1[i].PulseNumber = preparedPulses[i].PulseNumber
		preparedJetDrops2[i].PulseNumber = preparedPulses[i].PulseNumber
		preparedJetDrops3[i].PulseNumber = preparedPulses[i].PulseNumber
		preparedJetDrops4[i].PulseNumber = preparedPulses[i].PulseNumber

		if i-1 >= 0 {
			preparedJetDrops1[i].FirstPrevHash = preparedJetDrops1[i-1].Hash
			preparedJetDrops2[i].FirstPrevHash = preparedJetDrops2[i-1].Hash
			preparedJetDrops3[i].FirstPrevHash = preparedJetDrops3[i-1].Hash
			preparedJetDrops4[i].FirstPrevHash = preparedJetDrops4[i-1].Hash
		}
		preparedJetDrops1[i].SecondPrevHash = []byte{}
		preparedJetDrops2[i].SecondPrevHash = []byte{}
		preparedJetDrops3[i].SecondPrevHash = []byte{}
		preparedJetDrops4[i].SecondPrevHash = []byte{}
	}
	err := testutils.CreatePulses(testDB, preparedPulses)
	require.NoError(t, err)
	err = testutils.CreateJetDrops(testDB, preparedJetDrops1)
	require.NoError(t, err)
	err = testutils.CreateJetDrops(testDB, preparedJetDrops2)
	require.NoError(t, err)
	err = testutils.CreateJetDrops(testDB, preparedJetDrops3)
	require.NoError(t, err)
	err = testutils.CreateJetDrops(testDB, preparedJetDrops4)
	require.NoError(t, err)

	checkOkReturningResponse := func(t *testing.T, resp *http.Response, respErr error) server.JetDropsResponse {
		require.NoError(t, err)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(bodyBytes))
		var received server.JetDropsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		return received
	}

	t.Run("pulseNumberLte, pulseNumberGte, siblings only, has prev/next pulse in db", func(t *testing.T) {
		expectedCount := (totalCount - 2) * 4
		pulseNumberLte := preparedPulses[totalCount-2].PulseNumber
		pulseNumberGte := preparedPulses[1].PulseNumber
		query := fmt.Sprintf("pulse_number_lte=%d&pulse_number_gte=%d", pulseNumberLte, pulseNumberGte)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/*/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i := len(preparedJetDrops1) - 2; i > 0; i-- {
			next1 := []server.NextPrevJetDrop{}
			next2 := []server.NextPrevJetDrop{}
			next3 := []server.NextPrevJetDrop{}
			next4 := []server.NextPrevJetDrop{}
			if i+1 < len(preparedJetDrops1) {
				next1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i+1])}
				next2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i+1])}
				next3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i+1])}
				next4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i+1])}
			}
			prev1 := []server.NextPrevJetDrop{}
			prev2 := []server.NextPrevJetDrop{}
			prev3 := []server.NextPrevJetDrop{}
			prev4 := []server.NextPrevJetDrop{}
			if i-1 >= 0 {
				prev1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i-1])}
				prev2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i-1])}
				prev3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i-1])}
				prev4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i-1])}
			}

			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops1[i], prev1, next1), "preparedJetDrops1[%d] must be contained, jid %s", i, preparedJetDrops1[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops2[i], prev2, next2), "preparedJetDrops2[%d] must be contained, jid %s", i, preparedJetDrops2[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops3[i], prev3, next3), "preparedJetDrops3[%d] must be contained, jid %s", i, preparedJetDrops3[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops4[i], prev4, next4), "preparedJetDrops4[%d] must be contained, jid %s", i, preparedJetDrops4[i].JetID)
		}
	})

	t.Run("pulseNumberLte, pulseNumberGte, siblings only, has prev/next pulse in db, sort_by=pulse_number_asc,jet_id_desc", func(t *testing.T) {
		expectedCount := (totalCount - 2) * 4
		pulseNumberLte := preparedPulses[totalCount-2].PulseNumber
		pulseNumberGte := preparedPulses[1].PulseNumber
		query := fmt.Sprintf("sort_by=pulse_number_asc,jet_id_desc&pulse_number_lte=%d&pulse_number_gte=%d", pulseNumberLte, pulseNumberGte)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/*/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i := 1; i <= len(preparedJetDrops1)-2; i++ {
			next1 := []server.NextPrevJetDrop{}
			next2 := []server.NextPrevJetDrop{}
			next3 := []server.NextPrevJetDrop{}
			next4 := []server.NextPrevJetDrop{}
			if i+1 < len(preparedJetDrops1) {
				next1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i+1])}
				next2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i+1])}
				next3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i+1])}
				next4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i+1])}
			}
			prev1 := []server.NextPrevJetDrop{}
			prev2 := []server.NextPrevJetDrop{}
			prev3 := []server.NextPrevJetDrop{}
			prev4 := []server.NextPrevJetDrop{}
			if i-1 >= 0 {
				prev1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i-1])}
				prev2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i-1])}
				prev3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i-1])}
				prev4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i-1])}
			}

			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops1[i], prev1, next1), "preparedJetDrops1[%d] must be contained, jid %s", i, preparedJetDrops1[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops2[i], prev2, next2), "preparedJetDrops2[%d] must be contained, jid %s", i, preparedJetDrops2[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops3[i], prev3, next3), "preparedJetDrops3[%d] must be contained, jid %s", i, preparedJetDrops3[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops4[i], prev4, next4), "preparedJetDrops4[%d] must be contained, jid %s", i, preparedJetDrops4[i].JetID)
		}
	})

	t.Run("pulseNumberLte, pulseNumberGte, siblings only, no prev/next pulse in db", func(t *testing.T) {
		expectedCount := totalCount * 4
		pulseNumberLte := preparedPulses[totalCount-1].PulseNumber
		pulseNumberGte := preparedPulses[0].PulseNumber
		query := fmt.Sprintf("pulse_number_lte=%d&pulse_number_gte=%d", pulseNumberLte, pulseNumberGte)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/*/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i := len(preparedJetDrops1) - 1; i >= 0; i-- {
			next1 := []server.NextPrevJetDrop{}
			next2 := []server.NextPrevJetDrop{}
			next3 := []server.NextPrevJetDrop{}
			next4 := []server.NextPrevJetDrop{}
			if i+1 < len(preparedJetDrops1) {
				next1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i+1])}
				next2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i+1])}
				next3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i+1])}
				next4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i+1])}
			}
			prev1 := []server.NextPrevJetDrop{}
			prev2 := []server.NextPrevJetDrop{}
			prev3 := []server.NextPrevJetDrop{}
			prev4 := []server.NextPrevJetDrop{}
			if i-1 >= 0 {
				prev1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i-1])}
				prev2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i-1])}
				prev3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i-1])}
				prev4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i-1])}
			}

			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops1[i], prev1, next1), "preparedJetDrops1[%d] must be contained, jid %s", i, preparedJetDrops1[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops2[i], prev2, next2), "preparedJetDrops2[%d] must be contained, jid %s", i, preparedJetDrops2[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops3[i], prev3, next3), "preparedJetDrops3[%d] must be contained, jid %s", i, preparedJetDrops3[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops4[i], prev4, next4), "preparedJetDrops4[%d] must be contained, jid %s", i, preparedJetDrops4[i].JetID)
		}
	})

	t.Run("pulseNumberLt, pulseNumberGt, siblings only, no prev/next pulse in db", func(t *testing.T) {
		expectedCount := totalCount * 4
		pulseNumberLt := preparedPulses[totalCount-1].PulseNumber + 10
		pulseNumberGt := preparedPulses[0].PulseNumber - 10
		query := fmt.Sprintf("pulse_number_lt=%d&pulse_number_gt=%d", pulseNumberLt, pulseNumberGt)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/*/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i := len(preparedJetDrops1) - 1; i >= 0; i-- {
			next1 := []server.NextPrevJetDrop{}
			next2 := []server.NextPrevJetDrop{}
			next3 := []server.NextPrevJetDrop{}
			next4 := []server.NextPrevJetDrop{}
			if i+1 <= len(preparedJetDrops1)-1 {
				next1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i+1])}
				next2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i+1])}
				next3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i+1])}
				next4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i+1])}
			}
			prev1 := []server.NextPrevJetDrop{}
			prev2 := []server.NextPrevJetDrop{}
			prev3 := []server.NextPrevJetDrop{}
			prev4 := []server.NextPrevJetDrop{}
			if i-1 >= 0 {
				prev1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i-1])}
				prev2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i-1])}
				prev3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i-1])}
				prev4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i-1])}
			}

			jd1 := JetDropToAPI(preparedJetDrops1[i], prev1, next1)
			require.Contains(t, *response.Result, jd1, "preparedJetDrops1[%d] must be contained, jid %s", i, preparedJetDrops1[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops2[i], prev2, next2), "preparedJetDrops2[%d] must be contained, jid %s", i, preparedJetDrops2[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops3[i], prev3, next3), "preparedJetDrops3[%d] must be contained, jid %s", i, preparedJetDrops3[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops4[i], prev4, next4), "preparedJetDrops4[%d] must be contained, jid %s", i, preparedJetDrops4[i].JetID)
		}
	})

	t.Run("pulseNumberLt, pulseNumberGt, siblings only, has prev/next pulse in db", func(t *testing.T) {
		expectedCount := (totalCount - 4) * 4
		pulseNumberLt := preparedPulses[totalCount-2].PulseNumber
		pulseNumberGt := preparedPulses[1].PulseNumber
		query := fmt.Sprintf("pulse_number_lt=%d&pulse_number_gt=%d", pulseNumberLt, pulseNumberGt)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/*/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i := len(preparedJetDrops1) - 3; i > 1; i-- {
			next1 := []server.NextPrevJetDrop{}
			next2 := []server.NextPrevJetDrop{}
			next3 := []server.NextPrevJetDrop{}
			next4 := []server.NextPrevJetDrop{}
			if i+1 < len(preparedJetDrops1) {
				next1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i+1])}
				next2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i+1])}
				next3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i+1])}
				next4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i+1])}
			}
			prev1 := []server.NextPrevJetDrop{}
			prev2 := []server.NextPrevJetDrop{}
			prev3 := []server.NextPrevJetDrop{}
			prev4 := []server.NextPrevJetDrop{}
			if i-1 >= 0 {
				prev1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i-1])}
				prev2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i-1])}
				prev3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i-1])}
				prev4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i-1])}
			}

			jd1 := JetDropToAPI(preparedJetDrops1[i], prev1, next1)
			require.Contains(t, *response.Result, jd1, "preparedJetDrops1[%d] must be contained, jid %s", i, preparedJetDrops1[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops2[i], prev2, next2), "preparedJetDrops2[%d] must be contained, jid %s", i, preparedJetDrops2[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops3[i], prev3, next3), "preparedJetDrops3[%d] must be contained, jid %s", i, preparedJetDrops3[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops4[i], prev4, next4), "preparedJetDrops4[%d] must be contained, jid %s", i, preparedJetDrops4[i].JetID)
		}
	})

	t.Run("pulseNumberLt, pulseNumberGt, siblings only, has prev/next pulse in db, sort_by=pulse_number_asc,jet_id_desc", func(t *testing.T) {
		expectedCount := (totalCount - 4) * 4
		pulseNumberLt := preparedPulses[totalCount-2].PulseNumber
		pulseNumberGt := preparedPulses[1].PulseNumber
		query := fmt.Sprintf("sort_by=pulse_number_asc,jet_id_desc&pulse_number_lt=%d&pulse_number_gt=%d", pulseNumberLt, pulseNumberGt)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/*/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i := 2; i <= len(preparedJetDrops1)-3; i++ {
			next1 := []server.NextPrevJetDrop{}
			next2 := []server.NextPrevJetDrop{}
			next3 := []server.NextPrevJetDrop{}
			next4 := []server.NextPrevJetDrop{}
			if i+1 < len(preparedJetDrops1) {
				next1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i+1])}
				next2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i+1])}
				next3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i+1])}
				next4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i+1])}
			}
			prev1 := []server.NextPrevJetDrop{}
			prev2 := []server.NextPrevJetDrop{}
			prev3 := []server.NextPrevJetDrop{}
			prev4 := []server.NextPrevJetDrop{}
			if i-1 >= 0 {
				prev1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i-1])}
				prev2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i-1])}
				prev3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i-1])}
				prev4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i-1])}
			}

			jd1 := JetDropToAPI(preparedJetDrops1[i], prev1, next1)
			require.Contains(t, *response.Result, jd1, "preparedJetDrops1[%d] must be contained, jid %s", i, preparedJetDrops1[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops2[i], prev2, next2), "preparedJetDrops2[%d] must be contained, jid %s", i, preparedJetDrops2[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops3[i], prev3, next3), "preparedJetDrops3[%d] must be contained, jid %s", i, preparedJetDrops3[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops4[i], prev4, next4), "preparedJetDrops4[%d] must be contained, jid %s", i, preparedJetDrops4[i].JetID)
		}
	})

	t.Run("pulseNumberLt, pulseNumberGt, siblings only, has prev/next pulse in db, limit 8", func(t *testing.T) {
		expectedCount := 12
		pulseNumberLt := preparedPulses[totalCount-2].PulseNumber
		pulseNumberGt := preparedPulses[1].PulseNumber
		query := fmt.Sprintf("pulse_number_lt=%d&pulse_number_gt=%d&limit=8", pulseNumberLt, pulseNumberGt)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/*/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, 8)
		for i := 4; i > 2; i-- {
			next1 := []server.NextPrevJetDrop{}
			next2 := []server.NextPrevJetDrop{}
			next3 := []server.NextPrevJetDrop{}
			next4 := []server.NextPrevJetDrop{}
			if i+1 < len(preparedJetDrops1) {
				next1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i+1])}
				next2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i+1])}
				next3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i+1])}
				next4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i+1])}
			}
			prev1 := []server.NextPrevJetDrop{}
			prev2 := []server.NextPrevJetDrop{}
			prev3 := []server.NextPrevJetDrop{}
			prev4 := []server.NextPrevJetDrop{}
			if i-1 >= 0 {
				prev1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i-1])}
				prev2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i-1])}
				prev3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i-1])}
				prev4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i-1])}
			}

			jd1 := JetDropToAPI(preparedJetDrops1[i], prev1, next1)
			require.Contains(t, *response.Result, jd1, "preparedJetDrops1[%d] must be contained, jid %s", i, preparedJetDrops1[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops2[i], prev2, next2), "preparedJetDrops2[%d] must be contained, jid %s", i, preparedJetDrops2[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops3[i], prev3, next3), "preparedJetDrops3[%d] must be contained, jid %s", i, preparedJetDrops3[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops4[i], prev4, next4), "preparedJetDrops4[%d] must be contained, jid %s", i, preparedJetDrops4[i].JetID)
		}
	})

	t.Run("pulseNumberLt, pulseNumberGt, siblings only, has prev/next pulse in db, limit 8", func(t *testing.T) {
		expectedCount := 12
		pulseNumberLt := preparedPulses[totalCount-2].PulseNumber
		pulseNumberGt := preparedPulses[1].PulseNumber
		query := fmt.Sprintf("pulse_number_lt=%d&pulse_number_gt=%d&limit=8", pulseNumberLt, pulseNumberGt)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/*/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, 8)
		for i := 4; i > 2; i-- {
			next1 := []server.NextPrevJetDrop{}
			next2 := []server.NextPrevJetDrop{}
			next3 := []server.NextPrevJetDrop{}
			next4 := []server.NextPrevJetDrop{}
			if i+1 < len(preparedJetDrops1) {
				next1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i+1])}
				next2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i+1])}
				next3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i+1])}
				next4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i+1])}
			}
			prev1 := []server.NextPrevJetDrop{}
			prev2 := []server.NextPrevJetDrop{}
			prev3 := []server.NextPrevJetDrop{}
			prev4 := []server.NextPrevJetDrop{}
			if i-1 >= 0 {
				prev1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i-1])}
				prev2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i-1])}
				prev3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i-1])}
				prev4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i-1])}
			}

			jd1 := JetDropToAPI(preparedJetDrops1[i], prev1, next1)
			require.Contains(t, *response.Result, jd1, "preparedJetDrops1[%d] must be contained, jid %s", i, preparedJetDrops1[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops2[i], prev2, next2), "preparedJetDrops2[%d] must be contained, jid %s", i, preparedJetDrops2[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops3[i], prev3, next3), "preparedJetDrops3[%d] must be contained, jid %s", i, preparedJetDrops3[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops4[i], prev4, next4), "preparedJetDrops4[%d] must be contained, jid %s", i, preparedJetDrops4[i].JetID)
		}
	})

	t.Run("pulseNumberLte, pulseNumberGte, siblings only, has prev/next pulse in db, limit 8, sort_by=pulse_number_asc,jet_id_desc", func(t *testing.T) {
		expectedCount := (totalCount - 2) * 4
		pulseNumberLte := preparedPulses[totalCount-2].PulseNumber
		pulseNumberGte := preparedPulses[1].PulseNumber
		query := fmt.Sprintf("sort_by=pulse_number_asc,jet_id_desc&pulse_number_lte=%d&pulse_number_gte=%d&limit=8", pulseNumberLte, pulseNumberGte)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/*/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, 8)
		for i := 1; i < 3; i++ {
			next1 := []server.NextPrevJetDrop{}
			next2 := []server.NextPrevJetDrop{}
			next3 := []server.NextPrevJetDrop{}
			next4 := []server.NextPrevJetDrop{}
			if i+1 < len(preparedJetDrops1) {
				next1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i+1])}
				next2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i+1])}
				next3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i+1])}
				next4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i+1])}
			}
			prev1 := []server.NextPrevJetDrop{}
			prev2 := []server.NextPrevJetDrop{}
			prev3 := []server.NextPrevJetDrop{}
			prev4 := []server.NextPrevJetDrop{}
			if i-1 >= 0 {
				prev1 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops1[i-1])}
				prev2 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops2[i-1])}
				prev3 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops3[i-1])}
				prev4 = []server.NextPrevJetDrop{transformPrevNextResp(preparedJetDrops4[i-1])}
			}

			jd1 := JetDropToAPI(preparedJetDrops1[i], prev1, next1)
			require.Contains(t, *response.Result, jd1, "preparedJetDrops1[%d] must be contained, jid %s", i, preparedJetDrops1[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops2[i], prev2, next2), "preparedJetDrops2[%d] must be contained, jid %s", i, preparedJetDrops2[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops3[i], prev3, next3), "preparedJetDrops3[%d] must be contained, jid %s", i, preparedJetDrops3[i].JetID)
			require.Contains(t, *response.Result, JetDropToAPI(preparedJetDrops4[i], prev4, next4), "preparedJetDrops4[%d] must be contained, jid %s", i, preparedJetDrops4[i].JetID)
		}
	})
}

func TestServer_JetDropsByJetID(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	totalCount := 5
	jetID, preparedJetDrops, preparedPulses := testutils.GenerateJetDropsWithSomeJetID(t, totalCount)
	err := testutils.CreatePulses(testDB, preparedPulses)
	require.NoError(t, err)
	err = testutils.CreateJetDrops(testDB, preparedJetDrops)
	require.NoError(t, err)
	checkOkReturningResponse := func(t *testing.T, resp *http.Response, respErr error) server.JetDropsResponse {
		require.NoError(t, err)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(bodyBytes))
		var received server.JetDropsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		return received
	}

	checkJetDrops := func(t *testing.T, expected, received server.JetDrop) {
		require.Equal(t, *expected.Hash, *received.Hash)
		require.Equal(t, *expected.PulseNumber, *received.PulseNumber)
		require.Equal(t, *expected.JetId, *received.JetId)
		require.Equal(t, *expected.JetDropId, *received.JetDropId)
		require.Equal(t, *expected.NextJetDropId, *received.NextJetDropId)
		require.Equal(t, *expected.PrevJetDropId, *received.PrevJetDropId)
		require.Equal(t, *expected.RecordAmount, *received.RecordAmount)
		require.Equal(t, *expected.Timestamp, *received.Timestamp)
	}

	t.Run("happy_no_query_params", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops")
		response := checkOkReturningResponse(t, resp, err)
		require.EqualValues(t, totalCount, int(*response.Total))
		require.Len(t, *response.Result, totalCount)
		for _, drop := range preparedJetDrops {
			require.Contains(t, *response.Result, JetDropToAPI(drop, []server.NextPrevJetDrop{}, []server.NextPrevJetDrop{}))
		}
	})

	t.Run("jetIDNotFound", func(t *testing.T) {
		wrongJetID := "00000000000000000000001"
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + wrongJetID + "/jet-drops")
		response := checkOkReturningResponse(t, resp, err)
		require.Equal(t, 0, int(*response.Total))
		require.Len(t, *response.Result, 0)
	})

	t.Run("jetIDIsNotCorrect", func(t *testing.T) {
		wrongJetID := "1010102"
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + wrongJetID + "/jet-drops")
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var response server.CodeValidationError
		err = json.Unmarshal(bodyBytes, &response)
		require.NoError(t, err)
		falures := *response.ValidationFailures
		require.Len(t, falures, 1)
		require.Contains(t, *falures[0].FailureReason, "parameter does not match with jetID valid value")
	})

	t.Run("limit", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?limit=1")
		received := checkOkReturningResponse(t, resp, err)
		limit := 1
		require.Equal(t, totalCount, int(*received.Total))
		require.Len(t, *received.Result, limit)
	})

	t.Run("pulseNumberLte", func(t *testing.T) {
		expectedCount := 2
		pulseNumberLte := strconv.FormatInt(preparedPulses[1].PulseNumber, 10)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?pulse_number_lte=" + pulseNumberLte)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i := 0; i < expectedCount; i++ {
			expected := JetDropToAPI(preparedJetDrops[expectedCount-i-1], []server.NextPrevJetDrop{}, []server.NextPrevJetDrop{})
			received := (*response.Result)[i]
			checkJetDrops(t, expected, received)
		}
	})
	t.Run("pulseNumberLte-all", func(t *testing.T) {
		expectedCount := totalCount
		pulseNumberLte := strconv.FormatInt(preparedPulses[totalCount-1].PulseNumber, 10)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?pulse_number_lte=" + pulseNumberLte)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i := 0; i < expectedCount; i++ {
			expected := JetDropToAPI(preparedJetDrops[expectedCount-i-1], []server.NextPrevJetDrop{}, []server.NextPrevJetDrop{})
			received := (*response.Result)[i]
			checkJetDrops(t, expected, received)
		}
	})
	t.Run("pulseNumberLt", func(t *testing.T) {
		expectedCount := 1
		pulseNumberLte := strconv.FormatInt(preparedPulses[1].PulseNumber, 10)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?pulse_number_lt=" + pulseNumberLte)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i := 0; i < expectedCount; i++ {
			expected := JetDropToAPI(preparedJetDrops[expectedCount-i-1], []server.NextPrevJetDrop{}, []server.NextPrevJetDrop{})
			received := (*response.Result)[i]
			checkJetDrops(t, expected, received)
		}
	})
	t.Run("pulseNumberLt-no-one", func(t *testing.T) {
		expectedCount := 0
		pulseNumberLt := strconv.FormatInt(preparedPulses[0].PulseNumber, 10)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?pulse_number_lt=" + pulseNumberLt)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
	})
	t.Run("pulseNumberGte", func(t *testing.T) {
		expectedCount := totalCount - 1
		pulseNumberGte := strconv.FormatInt(preparedPulses[1].PulseNumber, 10)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?pulse_number_gte=" + pulseNumberGte)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i, j := 1, 0; i < expectedCount; i, j = i+1, j+1 {
			expected := JetDropToAPI(preparedJetDrops[totalCount-j-1], []server.NextPrevJetDrop{}, []server.NextPrevJetDrop{})
			received := (*response.Result)[j]
			checkJetDrops(t, expected, received)
		}
	})
	t.Run("pulseNumberGte-all", func(t *testing.T) {
		expectedCount := totalCount
		pulseNumberGte := strconv.FormatInt(preparedPulses[0].PulseNumber, 10)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?pulse_number_gte=" + pulseNumberGte)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i := 0; i < expectedCount; i++ {
			expected := JetDropToAPI(preparedJetDrops[expectedCount-i-1], []server.NextPrevJetDrop{}, []server.NextPrevJetDrop{})
			received := (*response.Result)[i]
			checkJetDrops(t, expected, received)
		}
	})
	t.Run("pulseNumberGt", func(t *testing.T) {
		expectedCount := totalCount - 2
		pulseNumberGt := strconv.FormatInt(preparedPulses[1].PulseNumber, 10)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?pulse_number_gt=" + pulseNumberGt)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i := 0; i < expectedCount; i++ {
			expected := JetDropToAPI(preparedJetDrops[totalCount-i-1], []server.NextPrevJetDrop{}, []server.NextPrevJetDrop{})
			received := (*response.Result)[i]
			checkJetDrops(t, expected, received)
		}
	})
	t.Run("pulseNumberGt-no-one", func(t *testing.T) {
		expectedCount := 0
		pulseNumberGt := strconv.FormatInt(preparedPulses[totalCount-1].PulseNumber, 10)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?pulse_number_gt=" + pulseNumberGt)
		received := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*received.Total))
		require.Len(t, *received.Result, expectedCount)
	})
	t.Run("pulseNumberGte and pulseNumberLte", func(t *testing.T) {
		expectedCount := totalCount - 2
		pulseNumberGte := preparedPulses[1].PulseNumber
		pulseNumberLte := preparedPulses[totalCount-2].PulseNumber
		query := fmt.Sprintf("pulse_number_gte=%d&pulse_number_lte=%d", pulseNumberGte, pulseNumberLte)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i, j := 1, 0; i < expectedCount; i, j = i+1, j+1 {
			expected := JetDropToAPI(preparedJetDrops[totalCount-i-1], []server.NextPrevJetDrop{}, []server.NextPrevJetDrop{})
			received := (*response.Result)[j]
			checkJetDrops(t, expected, received)
		}
	})
	t.Run("pulseNumberGte and pulseNumberLt", func(t *testing.T) {
		expectedCount := totalCount - 3
		pulseNumberGte := preparedPulses[1].PulseNumber
		pulseNumberLt := preparedPulses[totalCount-2].PulseNumber
		query := fmt.Sprintf("pulse_number_gte=%d&pulse_number_lt=%d", pulseNumberGte, pulseNumberLt)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i, j := 1, 0; i < expectedCount; i, j = i+1, j+1 {
			expected := JetDropToAPI(preparedJetDrops[i], []server.NextPrevJetDrop{}, []server.NextPrevJetDrop{})
			received := (*response.Result)[expectedCount-j-1]
			checkJetDrops(t, expected, received)
		}
	})
	t.Run("pulseNumberGt and pulseNumberLt", func(t *testing.T) {
		expectedCount := totalCount - 4
		pulseNumberGt := preparedPulses[1].PulseNumber
		pulseNumberLt := preparedPulses[totalCount-2].PulseNumber
		query := fmt.Sprintf("pulse_number_gt=%d&pulse_number_lt=%d", pulseNumberGt, pulseNumberLt)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i, j := 1, 0; i < expectedCount; i, j = i+1, j+1 {
			expected := JetDropToAPI(preparedJetDrops[i], []server.NextPrevJetDrop{}, []server.NextPrevJetDrop{})
			received := (*response.Result)[expectedCount-j-1]
			checkJetDrops(t, expected, received)
		}
	})
	t.Run("pulseNumberGt and pulseNumberLte", func(t *testing.T) {
		expectedCount := totalCount - 3
		pulseNumberGt := preparedPulses[1].PulseNumber
		pulseNumberLte := preparedPulses[totalCount-2].PulseNumber
		query := fmt.Sprintf("pulse_number_gt=%d&pulse_number_lte=%d", pulseNumberGt, pulseNumberLte)
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?" + query)
		response := checkOkReturningResponse(t, resp, err)

		require.Equal(t, expectedCount, int(*response.Total))
		require.Len(t, *response.Result, expectedCount)
		for i, j := 2, 0; i < expectedCount; i, j = i+1, j+1 {
			expected := JetDropToAPI(preparedJetDrops[i], []server.NextPrevJetDrop{}, []server.NextPrevJetDrop{})
			received := (*response.Result)[expectedCount-j-1]
			checkJetDrops(t, expected, received)
		}
	})
	t.Run("sort_by_asc_and_desc", func(t *testing.T) {
		pnAsc := url.QueryEscape(string(server.SortByPulse_pulse_number_asc_jet_id_desc))
		pnDesc := url.QueryEscape(string(server.SortByPulse_pulse_number_desc_jet_id_asc))

		doReqFn := func(t *testing.T, jetID string, sortBy string, totalCount int) server.JetDropsResponse {
			resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?sort_by=" + sortBy)
			received := checkOkReturningResponse(t, resp, err)
			require.Equal(t, totalCount, int(*received.Total))
			require.Len(t, *received.Result, totalCount)
			return received
		}

		receivedAsc := *doReqFn(t, jetID, pnAsc, totalCount).Result
		receivedDesc := *doReqFn(t, jetID, pnDesc, totalCount).Result

		for i := 0; i < totalCount; i++ {
			dropAsc := receivedAsc[i]
			dropDesc := receivedDesc[totalCount-1-i]
			checkJetDrops(t, dropAsc, dropDesc)
		}
	})

}

func TestJetDropRecords(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	jetDrop1 := testutils.InitJetDropDB(pulse)
	jetDrop1.JetID = "10101"
	err = testutils.CreateJetDrop(testDB, jetDrop1)
	require.NoError(t, err)
	recordResult := testutils.InitRecordDB(jetDrop1)
	recordResult.Type = models.ResultRecord
	recordResult.Order = 1
	err = testutils.CreateRecord(testDB, recordResult)
	require.NoError(t, err)
	recordState1 := testutils.InitRecordDB(jetDrop1)
	recordState1.Order = 2
	err = testutils.CreateRecord(testDB, recordState1)
	require.NoError(t, err)
	recordState2 := testutils.InitRecordDB(jetDrop1)
	recordState2.Order = 3
	err = testutils.CreateRecord(testDB, recordState2)
	require.NoError(t, err)

	jetDrop2 := testutils.InitJetDropDB(pulse)
	jetDrop2.JetID = "11111"
	err = testutils.CreateJetDrop(testDB, jetDrop2)
	require.NoError(t, err)
	err = testutils.CreateRecord(testDB, testutils.InitRecordDB(jetDrop2))
	require.NoError(t, err)

	jetDropID := *models.NewJetDropID(jetDrop1.JetID, int64(pulse.PulseNumber))
	t.Run("happy", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/v1/jet-drops/" + jetDropID.ToString() + "/records")
		require.NoError(t, err)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.RecordsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.EqualValues(t, 3, *received.Total)
		require.Len(t, *received.Result, 3)
		require.Equal(t, insolar.NewIDFromBytes(recordResult.Reference).String(), *(*received.Result)[0].Reference)
		require.Equal(t, insolar.NewIDFromBytes(recordState1.Reference).String(), *(*received.Result)[1].Reference)
		require.Equal(t, insolar.NewIDFromBytes(recordState2.Reference).String(), *(*received.Result)[2].Reference)
	})

	t.Run("type", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/v1/jet-drops/" + jetDropID.ToString() + "/records?type=result")
		require.NoError(t, err)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.RecordsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.EqualValues(t, 1, *received.Total)
		require.Len(t, *received.Result, 1)
		require.Equal(t, insolar.NewIDFromBytes(recordResult.Reference).String(), *(*received.Result)[0].Reference)
	})

	t.Run("limit", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/v1/jet-drops/" + jetDropID.ToString() + "/records?limit=2")
		require.NoError(t, err)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.RecordsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.EqualValues(t, 3, *received.Total)
		require.Len(t, *received.Result, 2)
		require.Equal(t, insolar.NewIDFromBytes(recordResult.Reference).String(), *(*received.Result)[0].Reference)
		require.Equal(t, insolar.NewIDFromBytes(recordState1.Reference).String(), *(*received.Result)[1].Reference)
	})

	t.Run("offset", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/v1/jet-drops/" + jetDropID.ToString() + "/records?offset=1")
		require.NoError(t, err)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.RecordsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.EqualValues(t, 3, *received.Total)
		require.Len(t, *received.Result, 2)
		require.Equal(t, insolar.NewIDFromBytes(recordState1.Reference).String(), *(*received.Result)[0].Reference)
		require.Equal(t, insolar.NewIDFromBytes(recordState2.Reference).String(), *(*received.Result)[1].Reference)
	})

	t.Run("from_index", func(t *testing.T) {
		index := fmt.Sprintf("%d:%d", pulse.PulseNumber, recordState1.Order)
		resp, err := http.Get("http://" + apihost + "/api/v1/jet-drops/" + jetDropID.ToString() + "/records?from_index=" + index)
		require.NoError(t, err)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.RecordsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.EqualValues(t, 2, *received.Total)
		require.Len(t, *received.Result, 2)
		require.Equal(t, insolar.NewIDFromBytes(recordState1.Reference).String(), *(*received.Result)[0].Reference)
		require.Equal(t, insolar.NewIDFromBytes(recordState2.Reference).String(), *(*received.Result)[1].Reference)
	})

	t.Run("empty", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/v1/jet-drops/" + "00000:12121212" + "/records")
		require.NoError(t, err)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.RecordsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.EqualValues(t, 0, *received.Total)
		require.Len(t, *received.Result, 0)
	})
}

func TestJetDropRecords_Several_Errors(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/v1/jet-drops/not_valid:value/records?limit=200000000&offset=-10&type=not_valid_type&from_index=not_valid_index")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.CodeValidationError
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	expected := server.CodeValidationError{
		Code:    NullableString(http.StatusText(http.StatusBadRequest)),
		Message: NullableString(InvalidParamsMessage),
		ValidationFailures: &[]server.CodeValidationFailures{{
			FailureReason: NullableString("should be in range [1, 1000]"),
			Property:      NullableString("limit"),
		}, {
			FailureReason: NullableString("should not be negative"),
			Property:      NullableString("offset"),
		}, {
			FailureReason: NullableString("invalid"),
			Property:      NullableString("jet_drop_id"),
		}, {
			FailureReason: NullableString("invalid"),
			Property:      NullableString("from_index"),
		}, {
			FailureReason: NullableString("should be 'request', 'state' or 'result'"),
			Property:      NullableString("type"),
		}},
	}
	require.Equal(t, expected, received)
}
