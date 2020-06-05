// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/insolar/insolar"
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

func (s Server) ObjectLifeline(ctx echo.Context, objectReference string, params ObjectLifelineParams) error {
	limit := 20
	if params.Limit != nil {
		limit = *params.Limit
	}
	if limit <= 0 || limit > 100 {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("`limit` should be in range [1, 100]"))
	}

	offset := 0
	if params.Offset != nil {
		offset = *params.Offset
	}
	if offset < 0 {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("`offset` should not be negative"))
	}

	ref, errMsg := s.checkReference(objectReference)
	if errMsg != nil {
		return ctx.JSON(http.StatusBadRequest, *errMsg)
	}

	sort := "desc"
	if params.Sort != nil {
		s := *params.Sort
		if s != "desc" && s != "asc" {
			return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("'sort' should be 'desc' or 'asc'"))
		}
		sort = s
	}

	records, count, err := s.storage.GetLifeline(
		ref.Bytes(),
		params.FromIndex,
		params.PulseNumberLt, params.PulseNumberGt,
		limit, offset,
		sort,
	)
	if err != nil {
		s.logger.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	result := []ResponsesRecordYaml{}
	for _, r := range records {
		result = append(result, RecordToAPI(r))
	}
	return ctx.JSON(http.StatusOK, ResponsesLifelineYaml{
		Total:  int64(count),
		Result: &result,
	})
}

func (s *Server) checkReference(referenceRow string) (*insolar.ID, *ErrorMessage) {
	referenceRow = strings.TrimSpace(referenceRow)
	var errMsg ErrorMessage

	if len(referenceRow) == 0 {
		errMsg = NewSingleMessageError("empty reference")
		return nil, &errMsg
	}

	reference, err := url.QueryUnescape(referenceRow)
	if err != nil {
		errMsg = NewSingleMessageError("error unescaping reference parameter")
		return nil, &errMsg
	}

	ref, err := insolar.NewReferenceFromString(reference)
	if err != nil {
		errMsg = NewSingleMessageError("reference wrong format")
		return nil, &errMsg
	}

	return ref.GetLocal(), nil
}
