// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/pulse"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/server"
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/instrumentation/belogger"
)

const InvalidParamsMessage = "Invalid query or path parameters"

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
	pulse, jetDropAmount, recordAmount, err := s.storage.GetPulse(int(pulseNumber))
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return ctx.JSON(http.StatusOK, struct{}{})
		}
		err = errors.Wrapf(err, "error while select pulse from db by pulse number %d", pulseNumber)
		s.logger.Error(err)
		apiErr := server.CodeError{
			Code:        NullableString(http.StatusText(http.StatusInternalServerError)),
			Description: NullableString(err.Error()),
		}
		return ctx.JSON(http.StatusInternalServerError, apiErr)
	}

	pulseResponse := PulseToAPI(pulse, jetDropAmount, recordAmount)
	return ctx.JSON(http.StatusOK, pulseResponse)
}

func (s *Server) JetDropsByPulseNumber(ctx echo.Context, pulseNumber server.PulseNumberPathParam, params server.JetDropsByPulseNumberParams) error {
	var failures []server.CodeValidationFailures
	if !pulse.IsValidAsPulseNumber(int(pulseNumber)) {
		failures = append(failures, server.CodeValidationFailures{
			FailureReason: NullableString("invalid"),
			Property:      NullableString("pulse"),
		})
	}
	limit, offset, err := checkLimitOffset(params.Limit, params.Offset)
	if err != nil {
		failures = append(failures, server.CodeValidationFailures{
			FailureReason: NullableString("invalid"),
			Property:      NullableString("limit or offset"),
		})
	}

	jetDropID, err := checkJetDropID(params.FromJetDropId)
	if err != nil {
		failures = append(failures, server.CodeValidationFailures{
			FailureReason: NullableString("invalid"),
			Property:      NullableString("jet drop id"),
		})
	}

	if failures != nil {
		response := server.CodeValidationError{
			Code:               NullableString(strconv.Itoa(http.StatusBadRequest)),
			Description:        nil,
			Link:               nil,
			Message:            NullableString(InvalidParamsMessage),
			ValidationFailures: &failures,
		}
		return ctx.JSON(http.StatusBadRequest, response)
	}

	jetDrops, total, err := s.storage.GetJetDropsWithParams(
		models.Pulse{PulseNumber: int(pulseNumber)},
		jetDropID,
		limit,
		offset,
	)
	if err != nil {
		s.logger.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	var result []server.JetDrop
	for _, jetDrop := range jetDrops {
		result = append(result, JetDropToAPI(jetDrop))

	}
	cnt := int64(total)
	return ctx.JSON(http.StatusOK, server.JetDropsResponse{
		Total:  &cnt,
		Result: &result,
	})
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

func checkJetDropID(jetDropID *server.FromJetDropId) (*string, error) {
	if jetDropID == nil {
		return nil, nil
	}
	str := string(*jetDropID)
	s := strings.Split(str, ":")
	if len(s) != 2 {
		return nil, errors.New("wrong jet drop id format")
	}
	if _, err := strconv.ParseInt(s[0], 2, 64); err != nil {
		return nil, errors.New("wrong jet drop id format")
	}
	if _, err := strconv.ParseInt(s[1], 10, 64); err != nil {
		return nil, errors.New("wrong jet drop id format")
	}
	return &str, nil
}
