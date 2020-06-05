// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

// import (
// 	"context"
// 	"math/big"
// 	"os"
// 	"testing"
// 	"time"
//
// 	"github.com/go-pg/pg"
// 	"github.com/insolar/insolar/pulse"
// 	apiconfiguration "github.com/insolar/observer/configuration/api"
// 	"github.com/jinzhu/gorm"
// 	"github.com/labstack/echo/v4"
// 	"github.com/stretchr/testify/require"
//
// 	"github.com/insolar/insolar/instrumentation/inslogger"
// 	"github.com/insolar/block-explorer/testutils"
// )
//
// const (
// 	apihost = ":14800"
// )
//
// var (
// 	db *gorm.DB
// )
//
// func TestMain(t *testing.M) {
//
// 	var dbCleaner func()
// 	var err error
// 	db, dbCleaner, err = testutils.SetupDB()
// 	require.NoError(t, err)
//
// 	e := echo.New()
//
// 	logger := inslogger.FromContext(context.Background())
//
// 	pStorage = postgres.NewPulseStorage(logger, db)
// 	nowPulse := 1575302444 - pulse.UnixTimeOfMinTimePulse + pulse.MinTimePulse
// 	_ = pStorage.Insert(&observer.Pulse{Number: pulse.Number(nowPulse)})
//
// 	observerAPI := NewObserverServer(db, logger, pStorage, apiconfiguration.Configuration{
// 		FeeAmount: testFee,
// 		Price:     testPrice,
// 	})
//
// 	RegisterHandlers(e, observerAPI)
// 	go func() {
// 		err := e.Start(apihost)
// 		dbCleaner()
// 		e.Logger.Fatal(err)
// 	}()
// 	// TODO: wait until API started
// 	time.Sleep(5 * time.Second)
//
// 	retCode := t.Run()
//
// 	dbCleaner()
// 	os.Exit(retCode)
// }
//
// func truncateDB(t *testing.T) {
// 	_, err := db.Model(&models.Transaction{}).Exec("TRUNCATE TABLE ?TableName CASCADE")
// 	require.NoError(t, err)
// 	_, err = db.Model(&models.Member{}).Exec("TRUNCATE TABLE ?TableName CASCADE")
// 	require.NoError(t, err)
// 	_, err = db.Model(&models.Deposit{}).Exec("TRUNCATE TABLE ?TableName CASCADE")
// 	require.NoError(t, err)
// 	_, err = db.Model(&models.MigrationAddress{}).Exec("TRUNCATE TABLE ?TableName CASCADE")
// 	require.NoError(t, err)
//
// 	_, err = db.Exec("TRUNCATE TABLE pulses CASCADE")
// 	require.NoError(t, err)
// 	nowPulse := 1575302444 - pulse.UnixTimeOfMinTimePulse + pulse.MinTimePulse
// 	_ = pStorage.Insert(&observer.Pulse{Number: pulse.Number(nowPulse)})
// }
