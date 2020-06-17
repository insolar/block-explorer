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
	"github.com/insolar/spec-insolar-block-explorer-api/v1/server"
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

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
	limit, offset, failures := checkLimitOffset(params.Limit, params.Offset)
	if len(failures) != 0 {
		apiErr := server.CodeValidationError{
			Code:               NullableString(http.StatusText(http.StatusBadRequest)),
			ValidationFailures: &failures,
		}
		return ctx.JSON(http.StatusBadRequest, apiErr)
	}

	var fromPulseString *int64
	var timestampLte *int
	var timestampGte *int

	if params.FromPulseNumber != nil {
		i := int64(*params.FromPulseNumber)
		fromPulseString = &i
	}
	if params.TimestampLte != nil {
		str := int(*params.TimestampLte)
		timestampLte = &str
	}
	if params.TimestampGte != nil {
		str := int(*params.TimestampGte)
		timestampGte = &str
	}

	pulses, count, err := s.storage.GetPulses(
		fromPulseString,
		timestampLte, timestampGte,
		limit, offset,
	)
	if err != nil {
		s.logger.Error(err)
		apiErr := server.CodeError{
			Code:        NullableString(http.StatusText(http.StatusInternalServerError)),
			Description: NullableString(err.Error()),
		}
		return ctx.JSON(http.StatusInternalServerError, apiErr)
	}

	var result []server.Pulse
	for _, p := range pulses {
		jetDrops, records, err := s.storage.GetAmounts(p.PulseNumber)
		if err != nil {
			s.logger.Error(err)
			apiErr := server.CodeError{
				Code:        NullableString(http.StatusText(http.StatusInternalServerError)),
				Description: NullableString(errors.Wrapf(err, "error while select count of records from db for pulse number %d", p.PulseNumber).Error()),
			}
			return ctx.JSON(http.StatusInternalServerError, apiErr)
		}
		result = append(result, PulseToAPI(p, jetDrops, records))
	}
	cnt := int64(count)
	return ctx.JSON(http.StatusOK, server.PulsesResponse{
		Total:  &cnt,
		Result: &result,
	})
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
	panic("implement me")
}

func (s *Server) Search(ctx echo.Context, params server.SearchParams) error {
	panic("implement me")
}

func (s *Server) ObjectLifeline(ctx echo.Context, objectReference server.ObjectReferencePathParam, params server.ObjectLifelineParams) error {
	limit, offset, failures := checkLimitOffset(params.Limit, params.Offset)
	if len(failures) != 0 {
		apiErr := server.CodeValidationError{
			Code:               NullableString(http.StatusText(http.StatusInternalServerError)),
			ValidationFailures: &failures,
		}
		return ctx.JSON(http.StatusBadRequest, apiErr)
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

func checkLimitOffset(l *server.LimitParam, o *server.OffsetParam) (int, int, []server.CodeValidationFailures) {
	var failures []server.CodeValidationFailures
	limit := 20
	if l != nil {
		limit = int(*l)
	}
	if limit <= 0 || limit > 100 {
		failures = append(failures, server.CodeValidationFailures{
			FailureReason: NullableString("should be in range [1, 100]"),
			Property:      NullableString("limit"),
		})
	}

	offset := 0
	if o != nil {
		offset = int(*o)
	}
	if offset < 0 {
		failures = append(failures, server.CodeValidationFailures{
			FailureReason: NullableString("should not be negative"),
			Property:      NullableString("offset"),
		})
	}

	return limit, offset, failures
}
