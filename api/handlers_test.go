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
	"net/http"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/jet"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/server"
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/etl/storage"
	"github.com/insolar/block-explorer/instrumentation/belogger"
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
	go func() {
		err := e.Start(apihost)
		dbCleaner()
		e.Logger.Fatal(err)
	}()
	// TODO: wait until API started
	time.Sleep(5 * time.Second)

	retCode := t.Run()

	dbCleaner()
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
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + objRef.String() + "/records?limit=20")
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
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + objRef.String() + "/records?sort_by=" + url.QueryEscape("+index"))
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
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + gen.Reference().String() + "/records?limit=200000000")
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
			FailureReason: NullableString("should be in range [1, 100]"),
			Property:      NullableString("limit"),
		}},
	}
	require.Equal(t, expected, received)
}

func TestObjectLifeline_Offset_Error(t *testing.T) {
	// request records with negative offset
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + gen.Reference().String() + "/records?offset=-10")
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
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + gen.Reference().String() + "/records?sort_by=not_supported_sort")
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
			FailureReason: NullableString("should be '-index' or '+index'"),
			Property:      NullableString("sort_by"),
		}},
	}
	require.Equal(t, expected, received)
}

func TestObjectLifeline_NoRecords(t *testing.T) {
	// request records for object without records
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + gen.Reference().String() + "/records")
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
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + "not_valid_reference" + "/records")
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
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + "  " + "/records")
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
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + gen.Reference().String() + "/records?from_index=not_valid_index")
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
	jetDrop1 := testutils.InitJetDropDB(pulse)
	jetDrop1.RecordAmount = 10
	err = testutils.CreateJetDrop(testDB, jetDrop1)
	jetDrop2 := testutils.InitJetDropDB(pulse)
	jetDrop2.RecordAmount = 25
	err = testutils.CreateJetDrop(testDB, jetDrop2)
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
	require.EqualValues(t, 2, *received.JetDropAmount)
	require.EqualValues(t, jetDrop1.RecordAmount+jetDrop2.RecordAmount, *received.RecordAmount)
}

func TestPulse_Pulse_NotExist(t *testing.T) {
	// request pulse for not existed pulse number
	resp, err := http.Get("http://" + apihost + fmt.Sprintf("/api/v1/pulses/%d", gen.PulseNumber()))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received server.PulseResponse
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Empty(t, received)
}

func TestPulse_Pulse_WrongFormat(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/v1/pulses/" + "wrong_type")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
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

func TestPulses_PulsesWithRecords(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

	// insert data
	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	jetDrop1 := testutils.InitJetDropDB(pulse)
	jetDrop1.RecordAmount = 10
	err = testutils.CreateJetDrop(testDB, jetDrop1)
	jetDrop2 := testutils.InitJetDropDB(pulse)
	jetDrop2.RecordAmount = 25
	err = testutils.CreateJetDrop(testDB, jetDrop2)
	require.NoError(t, err)

	secondPulse, err := testutils.InitPulseDB()
	secondPulse.PulseNumber = pulse.PulseNumber + 10
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)
	jetDrop3 := testutils.InitJetDropDB(secondPulse)
	jetDrop3.RecordAmount = 6
	err = testutils.CreateJetDrop(testDB, jetDrop3)

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
			FailureReason: NullableString("should be in range [1, 100]"),
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
			FailureReason: NullableString("should be in range [1, 100]"),
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
		jetDrop2.JetID = jetID2.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop2)
		require.NoError(t, err)

		jetDrop1 := testutils.InitJetDropDB(pulse)
		jetID1 := jet.NewIDFromString("000")
		jetDrop1.JetID = jetID1.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop1)
		require.NoError(t, err)

		jetDrop3 := testutils.InitJetDropDB(pulse)
		jetID3 := jet.NewIDFromString("010")
		jetDrop3.JetID = jetID3.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop3)
		require.NoError(t, err)

		resp, err := http.Get("http://" + apihost + "/api/v1/pulses/" + strconv.Itoa(pulse.PulseNumber) + "/jet-drops")
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
		require.Equal(t, models.NewJetDropID(jetDrop1.JetID, int64(pulse.PulseNumber)).ToString(), *(*received.Result)[0].JetDropId)
		require.Equal(t, models.NewJetDropID(jetDrop2.JetID, int64(pulse.PulseNumber)).ToString(), *(*received.Result)[1].JetDropId)
		require.Equal(t, models.NewJetDropID(jetDrop3.JetID, int64(pulse.PulseNumber)).ToString(), *(*received.Result)[2].JetDropId)
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
		jetDrop1.JetID = jetID1.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop1)
		require.NoError(t, err)

		jetDrop2 := testutils.InitJetDropDB(pulse)
		jetID2 := jet.NewIDFromString("001")
		jetDrop2.JetID = jetID2.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop2)
		require.NoError(t, err)

		jetDrop3 := testutils.InitJetDropDB(pulse)
		jetID3 := jet.NewIDFromString("010")
		jetDrop3.JetID = jetID3.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop3)
		require.NoError(t, err)

		resp, err := http.Get("http://" + apihost + "/api/v1/pulses/" + strconv.Itoa(pulse.PulseNumber) + "/jet-drops?limit=2")
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
		jetDrop1.JetID = jetID1.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop1)
		require.NoError(t, err)

		jetDrop2 := testutils.InitJetDropDB(pulse)
		jetID2 := jet.NewIDFromString("001")
		jetDrop2.JetID = jetID2.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop2)
		require.NoError(t, err)

		jetDrop3 := testutils.InitJetDropDB(pulse)
		jetID3 := jet.NewIDFromString("010")
		jetDrop3.JetID = jetID3.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop3)
		require.NoError(t, err)
		jetDropID3 := models.NewJetDropID(jetDrop3.JetID, int64(pulse.PulseNumber)).ToString()

		jetDrop4 := testutils.InitJetDropDB(pulse)
		jetID4 := jet.NewIDFromString("011")
		jetDrop4.JetID = jetID4.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop4)
		require.NoError(t, err)

		jetDrop5 := testutils.InitJetDropDB(pulse)
		jetID5 := jet.NewIDFromString("100")
		jetDrop5.JetID = jetID5.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop5)
		require.NoError(t, err)

		jetDrop6 := testutils.InitJetDropDB(pulse)
		jetID6 := jet.NewIDFromString("101")
		jetDrop6.JetID = jetID6.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop6)
		require.NoError(t, err)

		resp, err := http.Get(
			"http://" + apihost + "/api/v1/pulses/" +
				strconv.Itoa(pulse.PulseNumber) +
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
				strconv.Itoa(pulse.PulseNumber) +
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
				strconv.Itoa(pulse.PulseNumber) +
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
				strconv.Itoa(pulse.PulseNumber) +
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
				FailureReason: NullableString("should be in range [1, 100]"),
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
				strconv.Itoa(pulse.PulseNumber) +
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

func TestServer_JetDropsByID(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

		// insert records
		pulse, err := testutils.InitPulseDB()
		require.NoError(t, err)
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(t, err)

		jetDrop2 := testutils.InitJetDropDB(pulse)
		jetID2 := jet.NewIDFromString("001")
		jetDrop2.JetID = jetID2.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop2)
		require.NoError(t, err)

		jetDrop1 := testutils.InitJetDropDB(pulse)
		jetID1 := jet.NewIDFromString("000")
		jetDrop1.JetID = jetID1.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop1)
		require.NoError(t, err)
		jetDropID1 := models.NewJetDropID(jetDrop1.JetID, int64(pulse.PulseNumber)).ToString()

		jetDrop3 := testutils.InitJetDropDB(pulse)
		jetID3 := jet.NewIDFromString("010")
		jetDrop3.JetID = jetID3.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop3)
		require.NoError(t, err)

		resp, err := http.Get("http://" + apihost + "/api/v1/jet-drops/" + jetDropID1)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.JetDropResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, jetDropID1, *received.JetDropId)
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
		jetDrop2.JetID = jetID2.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop2)
		require.NoError(t, err)
		jetDropID2 := models.NewJetDropID(jetDrop2.JetID, int64(pulse.PulseNumber)).ToString()

		jetDrop1 := testutils.InitJetDropDB(pulse)
		jetID1 := jet.NewIDFromString("000")
		jetDrop1.JetID = jetID1.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop1)
		require.NoError(t, err)
		jetDropID1 := models.NewJetDropID(jetDrop1.JetID, int64(pulse.PulseNumber)).ToString()

		jetDrop3 := testutils.InitJetDropDB(pulse)
		jetID3 := jet.NewIDFromString("010")
		jetDrop3.JetID = jetID3.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop3)
		require.NoError(t, err)
		jetDropID3 := models.NewJetDropID(jetDrop3.JetID, int64(pulse.PulseNumber)).ToString()

		// create next pulse and jet drops
		pulse.PulseNumber = pulse.PulseNumber + 10
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(t, err)

		jetDrop4 := testutils.InitJetDropDB(pulse)
		jetID4 := jet.NewIDFromString("001")
		jetDrop4.JetID = jetID4.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop4)
		require.NoError(t, err)
		jetDropID4 := models.NewJetDropID(jetDrop4.JetID, int64(pulse.PulseNumber)).ToString()

		jetDrop5 := testutils.InitJetDropDB(pulse)
		jetID5 := jet.NewIDFromString("000")
		jetDrop5.JetID = jetID5.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop5)
		require.NoError(t, err)
		jetDropID5 := models.NewJetDropID(jetDrop5.JetID, int64(pulse.PulseNumber)).ToString()

		jetDrop6 := testutils.InitJetDropDB(pulse)
		jetID6 := jet.NewIDFromString("010")
		jetDrop6.JetID = jetID6.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop6)
		require.NoError(t, err)
		jetDropID6 := models.NewJetDropID(jetDrop6.JetID, int64(pulse.PulseNumber)).ToString()

		// create next pulse and jet drops
		pulse.PulseNumber = pulse.PulseNumber + 10
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(t, err)

		jetDrop7 := testutils.InitJetDropDB(pulse)
		jetID7 := jet.NewIDFromString("001")
		jetDrop7.JetID = jetID7.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop7)
		require.NoError(t, err)
		jetDropID7 := models.NewJetDropID(jetDrop7.JetID, int64(pulse.PulseNumber)).ToString()

		jetDrop8 := testutils.InitJetDropDB(pulse)
		jetID8 := jet.NewIDFromString("000")
		jetDrop8.JetID = jetID8.Prefix()
		err = testutils.CreateJetDrop(testDB, jetDrop8)
		require.NoError(t, err)
		jetDropID8 := models.NewJetDropID(jetDrop8.JetID, int64(pulse.PulseNumber)).ToString()

		jetDrop9 := testutils.InitJetDropDB(pulse)
		jetID9 := jet.NewIDFromString("010")
		jetDrop9.JetID = jetID9.Prefix()
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
			FailureReason: NullableString("invalid"),
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
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestServer_JetDropsByJetID(t *testing.T) {
	totalCount := 5
	someJetId, preparedJetDrops, preparedPulses := testutils.GenerateJetDropsWithSomeJetID(t, totalCount)
	err := testutils.CreatePulses(testDB, preparedPulses)
	require.NoError(t, err)
	err = testutils.CreateJetDrops(testDB, preparedJetDrops)
	require.NoError(t, err)
	jetID := models.ExporterJetIDToString(someJetId)

	t.Run("happy_no_query_params", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.JetDropsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.EqualValues(t, totalCount, int(*received.Total))
		require.Len(t, *received.Result, totalCount)
		for _, drop := range preparedJetDrops {
			require.Contains(t, *received.Result, JetDropToAPI(drop))
		}
	})

	t.Run("jetIDNotFound", func(t *testing.T) {
		wrongJetID := jetID + "1010100010"
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + wrongJetID + "/jet-drops")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received server.JetDropsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, 0, int(*received.Total))
		require.Len(t, *received.Result, 0)
	})

	t.Run("limit", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?limit=1")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		limit := 1
		var received server.JetDropsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, totalCount, int(*received.Total))
		require.Len(t, *received.Result, limit)
	})

	t.Run("offset and limit", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/v1/jets/" + jetID + "/jet-drops?limit=1&offset=4")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		expectedCount := 1
		var received server.JetDropsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, totalCount, int(*received.Total))
		require.Len(t, *received.Result, expectedCount)
	})

	t.Run("jetDropIDGte", func(t *testing.T) {
		t.Skip("wait for jet_drop_id_gt will be string not int")
		jetDropIDGte := models.NewJetDropID(
			preparedJetDrops[totalCount-1].JetID,
			int64(preparedJetDrops[totalCount-1].PulseNumber))
		resp, err := http.Get(
			fmt.Sprintf("http://%s/api/v1/jets/%s/jet-drops?jet_drop_id_gt=%s",
				apihost, jetID, jetDropIDGte.ToString()))
		require.NoError(t, err)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		expectedCount := 1
		var received server.JetDropsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, expectedCount, int(*received.Total))
		require.Len(t, *received.Result, expectedCount)
	})

	t.Run("jetDropIDLte", func(t *testing.T) {
		t.Skip("wait for jet_drop_id_lt will be string not int")
		jetDropIDLte := models.NewJetDropID(
			preparedJetDrops[2].JetID,
			int64(preparedJetDrops[2].PulseNumber))
		resp, err := http.Get(
			fmt.Sprintf("http://%s/api/v1/jets/%s/jet-drops?jet_drop_id_lt=%s",
				apihost, jetID, jetDropIDLte.ToString()))
		require.NoError(t, err)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		expectedCount := 3
		var received server.JetDropsResponse
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, expectedCount, int(*received.Total))
		require.Len(t, *received.Result, expectedCount)
	})

}

func TestJetDropRecords(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	jetDrop1 := testutils.InitJetDropDB(pulse)
	jetDrop1.JetID = jet.NewIDFromString("10101").Prefix()
	err = testutils.CreateJetDrop(testDB, jetDrop1)
	require.NoError(t, err)
	recordResult := testutils.InitRecordDB(jetDrop1)
	recordResult.Type = models.Result
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
	jetDrop2.JetID = jet.NewIDFromString("11111").Prefix()
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
		jetDropEmpty := testutils.InitJetDropDB(pulse)
		jetDropIDEmpty := *models.NewJetDropID(jetDropEmpty.JetID, int64(pulse.PulseNumber+1000))
		resp, err := http.Get("http://" + apihost + "/api/v1/jet-drops/" + jetDropIDEmpty.ToString() + "/records")
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
			FailureReason: NullableString("should be in range [1, 100]"),
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
