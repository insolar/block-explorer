// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

import (
	"context"
	"fmt"
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
	limit, offset, failures := checkLimitOffset(params.Limit, params.Offset)

	var fromPulseString *int64
	var timestampLte *int
	var timestampGte *int

	if params.FromPulseNumber != nil {
		i := int64(*params.FromPulseNumber)
		fromPulseString = &i
		if !pulse.IsValidAsPulseNumber(int(i)) {
			failures = append(failures, server.CodeValidationFailures{
				FailureReason: NullableString("invalid"),
				Property:      NullableString("pulse"),
			})
		}
	}

	if failures != nil {
		response := server.CodeValidationError{
			Code:               NullableString(http.StatusText(http.StatusBadRequest)),
			Message:            NullableString(InvalidParamsMessage),
			ValidationFailures: &failures,
		}
		return ctx.JSON(http.StatusBadRequest, response)
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
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	var result []server.Pulse
	for _, p := range pulses {
		jetDrops, records, err := s.storage.GetAmounts(p.PulseNumber)
		if err != nil {
			s.logger.Error(err)
			return ctx.JSON(http.StatusInternalServerError, struct{}{})
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
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	pulseResponse := PulseToAPI(pulse, jetDropAmount, recordAmount)
	return ctx.JSON(http.StatusOK, pulseResponse)
}

func (s *Server) JetDropsByPulseNumber(ctx echo.Context, pulseNumber server.PulseNumberPathParam, params server.JetDropsByPulseNumberParams) error {
	limit, offset, failures := checkLimitOffset(params.Limit, params.Offset)

	if !pulse.IsValidAsPulseNumber(int(pulseNumber)) {
		failures = append(failures, server.CodeValidationFailures{
			FailureReason: NullableString("invalid"),
			Property:      NullableString("pulse"),
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
	limit, offset, failures := checkLimitOffset(params.Limit, params.Offset)
	if len(failures) != 0 {
		apiErr := server.CodeValidationError{
			Code:               NullableString(http.StatusText(http.StatusBadRequest)),
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

func checkJetDropID(jetDropID *server.FromJetDropId) (*string, error) {
	if jetDropID == nil {
		return nil, nil
	}
	str := string(*jetDropID)
	s := strings.Split(str, ":")
	if len(s) != 2 {
		return nil, fmt.Errorf("wrong jet drop id format")
	}
	if _, err := strconv.ParseInt(s[0], 2, 64); err != nil {
		return nil, fmt.Errorf("wrong jet drop id format")
	}
	if _, err := strconv.ParseInt(s[1], 10, 64); err != nil {
		return nil, fmt.Errorf("wrong jet drop id format")
	}
	return &str, nil
}
