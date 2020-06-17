// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
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
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + objRef.String() + "/records?sort_by=asc")
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

	var received ErrorMessage
	err = json.Unmarshal(bodyBytes, &received)
	fmt.Println(string(bodyBytes))
	require.NoError(t, err)

	expected := ErrorMessage{Error: []string{"query parameter 'limit' should be in range [1, 100]"}}
	require.Equal(t, expected, received)
}

func TestObjectLifeline_Offset_Error(t *testing.T) {
	// request records with negative offset
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + gen.Reference().String() + "/records?offset=-10")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received ErrorMessage
	err = json.Unmarshal(bodyBytes, &received)
	fmt.Println(string(bodyBytes))
	require.NoError(t, err)

	expected := ErrorMessage{Error: []string{"query parameter 'offset' should not be negative"}}
	require.Equal(t, expected, received)
}

func TestObjectLifeline_Sort_Error(t *testing.T) {
	// request records with wrong sort param
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + gen.Reference().String() + "/records?sort_by=not_supported_sort")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received ErrorMessage
	err = json.Unmarshal(bodyBytes, &received)
	fmt.Println(string(bodyBytes))
	require.NoError(t, err)

	expected := ErrorMessage{Error: []string{"query parameter 'sort' should be 'desc' or 'asc'"}}
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
	require.Nil(t, received.Result)
}

func TestObjectLifeline_ReferenceFormat_Error(t *testing.T) {
	// request records with wrong format object reference
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + "not_valid_reference" + "/records")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received ErrorMessage
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	expected := ErrorMessage{Error: []string{"path parameter object reference wrong format"}}
	require.Equal(t, expected, received)
}

func TestObjectLifeline_ReferenceEmpty_Error(t *testing.T) {
	// request records with empty object reference
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + "  " + "/records")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received ErrorMessage
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	expected := ErrorMessage{Error: []string{"empty reference"}}
	require.Equal(t, expected, received)
}

func TestObjectLifeline_Index_Error(t *testing.T) {
	// request records with wrong format from_index param
	resp, err := http.Get("http://" + apihost + "/api/v1/lifeline/" + gen.Reference().String() + "/records?from_index=not_valid_index")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received ErrorMessage
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	expected := ErrorMessage{Error: []string{"query parameter 'index' should have the '<pulse_number>:<order>' format"}}
	require.Equal(t, expected, received)
}
