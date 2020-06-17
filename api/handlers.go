// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/server"
	"github.com/labstack/echo/v4"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/instrumentation/belogger"
)

type Server struct {
	storage interfaces.StorageFetcher
	logger  log.Logger
	config  configuration.API
}

// NewServer returns instance of API server
func NewServer(ctx context.Context, storage interfaces.StorageFetcher, config configuration.API) *Server {
	logger := belogger.FromContext(ctx)
	return &Server{storage: storage, logger: logger, config: config}
}

func (s *Server) JetDropByID(ctx echo.Context, jetDropID server.JetDropIdPathParam) error {
	panic("implement me")
}

func (s *Server) JetDropRecords(ctx echo.Context, jetDropID server.JetDropIdPathParam, params server.JetDropRecordsParams) error {
	panic("implement me")
}

func (s *Server) JetDropsByJetID(ctx echo.Context, jetID server.JetIdPathParam, params server.JetDropsByJetIDParams) error {
	panic("implement me")
}

func (s *Server) Pulses(ctx echo.Context, params server.PulsesParams) error {
	panic("implement me")
}

func (s *Server) Pulse(ctx echo.Context, pulseNumber server.PulseNumberPathParam) error {
	panic("implement me")
}

func (s *Server) JetDropsByPulseNumber(ctx echo.Context, pulseNumber server.PulseNumberPathParam, params server.JetDropsByPulseNumberParams) error {
	panic("implement me")
}

func (s *Server) Search(ctx echo.Context, params server.SearchParams) error {
	panic("implement me")
}

func (s *Server) ObjectLifeline(ctx echo.Context, objectReference server.ObjectReferencePathParam, params server.ObjectLifelineParams) error {
	limit, offset, err := checkLimitOffset(params.Limit, params.Offset)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError(err.Error()))
	}

	ref, errMsg := checkReference(objectReference)
	if errMsg != nil {
		return ctx.JSON(http.StatusBadRequest, *errMsg)
	}

	sort := "desc"
	if params.SortBy != nil {
		s := string(*params.SortBy)
		if s != "desc" && s != "asc" {
			return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("query parameter 'sort' should be 'desc' or 'asc'"))
		}
		sort = s
	}

	var fromIndexString *string
	var pulseNumberLtString *int
	var pulseNumberGtString *int

	if params.FromIndex != nil {
		str := string(*params.FromIndex)
		fromIndexString = &str
	}
	if params.PulseNumberLt != nil {
		str := int(*params.PulseNumberLt)
		pulseNumberLtString = &str
	}
	if params.PulseNumberGt != nil {
		str := int(*params.PulseNumberGt)
		pulseNumberGtString = &str
	}

	records, count, err := s.storage.GetLifeline(
		ref.Bytes(),
		fromIndexString,
		pulseNumberLtString, pulseNumberGtString,
		limit, offset,
		sort,
	)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "query parameter") {
			return ctx.JSON(http.StatusBadRequest, NewSingleMessageError(errMsg))

		}
		s.logger.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	var result []server.Record
	for _, r := range records {
		result = append(result, RecordToAPI(r))
	}
	cnt := int64(count)
	return ctx.JSON(http.StatusOK, server.RecordsResponse{
		Total:  &cnt,
		Result: &result,
	})
}

func checkReference(referenceRow server.ObjectReferencePathParam) (*insolar.ID, *ErrorMessage) {
	referenceString := strings.TrimSpace(string(referenceRow))
	var errMsg ErrorMessage

	if len(referenceString) == 0 {
		errMsg = NewSingleMessageError("empty reference")
		return nil, &errMsg
	}

	reference, err := url.QueryUnescape(referenceString)
	if err != nil {
		errMsg = NewSingleMessageError("error unescaping reference parameter")
		return nil, &errMsg
	}

	ref, err := insolar.NewReferenceFromString(reference)
	if err != nil {
		errMsg = NewSingleMessageError("path parameter object reference wrong format")
		return nil, &errMsg
	}

	return ref.GetLocal(), nil
}

func checkLimitOffset(l *server.LimitParam, o *server.OffsetParam) (int, int, error) {
	limit := 20
	if l != nil {
		limit = int(*l)
	}
	if limit <= 0 || limit > 100 {
		return 0, 0, errors.New("query parameter 'limit' should be in range [1, 100]")
	}

	offset := 0
	if o != nil {
		offset = int(*o)
	}
	if offset < 0 {
		return 0, 0, errors.New("query parameter 'offset' should not be negative")
	}

	return limit, offset, nil
}
