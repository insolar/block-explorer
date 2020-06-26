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
	"github.com/insolar/block-explorer/etl/storage"
	"github.com/insolar/block-explorer/instrumentation/belogger"
)

const InvalidParamsMessage = "Invalid query or path parameters"

type Server struct {
	storage interfaces.StorageAPIFetcher
	logger  log.Logger
	config  configuration.API
}

// NewServer returns instance of API server
func NewServer(ctx context.Context, storage interfaces.StorageAPIFetcher, config configuration.API) *Server {
	logger := belogger.FromContext(ctx)
	return &Server{storage: storage, logger: logger, config: config}
}

func (s *Server) JetDropByID(ctx echo.Context, jetDropID server.JetDropIdPathParam) error {
	exporterJetDropID, err := models.NewJetDropIDFromString(string(jetDropID))
	if err != nil {
		response := server.CodeValidationError{
			Code:        NullableString(strconv.Itoa(http.StatusBadRequest)),
			Description: nil,
			Link:        nil,
			Message:     NullableString(InvalidParamsMessage),
			ValidationFailures: &[]server.CodeValidationFailures{{
				FailureReason: NullableString("invalid"),
				Property:      NullableString("jet drop id"),
			}},
		}
		return ctx.JSON(http.StatusBadRequest, response)
	}
	jetDrop, err := s.storage.GetJetDropByID(
		*exporterJetDropID,
	)
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return ctx.JSON(http.StatusNotFound, struct{}{})
		}
		s.logger.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	apiJetDrop := JetDropToAPI(jetDrop)
	return ctx.JSON(http.StatusOK, server.JetDropResponse(apiJetDrop))
}

func (s *Server) JetDropRecords(ctx echo.Context, jetDropID server.JetDropIdPathParam, params server.JetDropRecordsParams) error {
	limit, offset, failures := checkLimitOffset(params.Limit, params.Offset)

	jetDrop, err := models.NewJetDropIDFromString(string(jetDropID))
	if err != nil {
		failures = append(failures, server.CodeValidationFailures{
			FailureReason: NullableString("invalid"),
			Property:      NullableString("jet_drop_id"),
		})
	}

	var fromIndexString *string
	var recordType *string

	if params.FromIndex != nil {
		str := string(*params.FromIndex)
		fromIndexString = &str
		_, _, err := storage.CheckIndex(str)
		if err != nil {
			failures = append(failures, server.CodeValidationFailures{
				FailureReason: NullableString("invalid"),
				Property:      NullableString("from_index"),
			})
		}
	}
	if params.Type != nil {
		str := string(*params.Type)
		if str != "request" && str != "result" && str != "state" {
			failures = append(failures, server.CodeValidationFailures{
				FailureReason: NullableString("should be 'request', 'state' or 'result'"),
				Property:      NullableString("type"),
			})
		}
		recordType = &str
	}

	if failures != nil {
		apiErr := server.CodeValidationError{
			Code:               NullableString(http.StatusText(http.StatusBadRequest)),
			Message:            NullableString(InvalidParamsMessage),
			ValidationFailures: &failures,
		}
		return ctx.JSON(http.StatusBadRequest, apiErr)
	}

	records, count, err := s.storage.GetRecordsByJetDrop(
		*jetDrop,
		fromIndexString,
		recordType,
		limit, offset,
	)
	if err != nil {
		s.logger.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	result := []server.Record{}
	for _, r := range records {
		result = append(result, RecordToAPI(r))
	}
	cnt := int64(count)
	return ctx.JSON(http.StatusOK, server.RecordsResponse{
		Total:  &cnt,
		Result: &result,
	})
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

	result := []server.Pulse{}
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
			return ctx.JSON(http.StatusNotFound, struct{}{})
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
	var exporterJetDropID *models.JetDropID
	var err error
	if params.FromJetDropId != nil {
		exporterJetDropID, err = models.NewJetDropIDFromString(string(*params.FromJetDropId))
		if err != nil {
			failures = append(failures, server.CodeValidationFailures{
				FailureReason: NullableString("invalid"),
				Property:      NullableString("jet drop id"),
			})
		}
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
		exporterJetDropID,
		limit,
		offset,
	)
	if err != nil {
		s.logger.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	drops := make([]server.JetDrop, len(jetDrops))
	for i, jetDrop := range jetDrops {
		drops[i] = JetDropToAPI(jetDrop)
	}
	cnt := int64(total)

	return ctx.JSON(http.StatusOK, server.JetDropsResponse{
		Total:  &cnt,
		Result: &drops,
	})
}

func (s *Server) Search(ctx echo.Context, params server.SearchParams) error {
	value := params.Value
	// check if value is pulse
	pulseNumber, err := strconv.ParseInt(value, 10, 64)
	if err == nil {
		return s.searchResponsePulse(ctx, pulseNumber)
	}
	// check if value is jetDropID
	_, err = models.NewJetDropIDFromString(value)
	if err == nil {
		return ctx.JSON(http.StatusOK, server.SearchJetDrop{
			Meta: &struct {
				JetDropId *string `json:"jet_drop_id,omitempty"` // nolint
			}{
				JetDropId: &value,
			},
			Type: NullableString("jet-drop"),
		})
	}
	// check if value is reference
	ref, err := checkReference(value)
	if err == nil {
		return s.searchReferencePulse(ctx, ref)
	}
	response := server.CodeValidationError{
		Code:        NullableString(http.StatusText(http.StatusBadRequest)),
		Description: NullableString(InvalidParamsMessage),
		ValidationFailures: &[]server.CodeValidationFailures{{
			FailureReason: NullableString("is neither pulse number, jet drop id nor reference"),
			Property:      NullableString("value"),
		}},
	}
	return ctx.JSON(http.StatusBadRequest, response)
}

func (s *Server) searchResponsePulse(ctx echo.Context, pulseNumber int64) error {
	if !pulse.IsValidAsPulseNumber(int(pulseNumber)) {
		response := server.CodeValidationError{
			Code:        NullableString(http.StatusText(http.StatusBadRequest)),
			Description: NullableString(InvalidParamsMessage),
			ValidationFailures: &[]server.CodeValidationFailures{{
				FailureReason: NullableString("not valid pulse number"),
				Property:      NullableString("value"),
			}},
		}
		return ctx.JSON(http.StatusBadRequest, response)
	}
	return ctx.JSON(http.StatusOK, server.SearchPulse{
		Meta: &struct {
			PulseNumber *int64 `json:"pulse_number,omitempty"`
		}{
			PulseNumber: &pulseNumber,
		},
		Type: NullableString("pulse"),
	})
}

func (s *Server) searchReferencePulse(ctx echo.Context, ref *insolar.Reference) error {
	if ref.IsObjectReference() {
		return ctx.JSON(http.StatusOK, server.SearchLifeline{
			Meta: &struct {
				ObjectReference *string `json:"object_reference,omitempty"`
			}{
				ObjectReference: NullableString(ref.String()),
			},
			Type: NullableString("lifeline"),
		})
	}
	// get record from db to provide information for response
	record, err := s.storage.GetRecord(ref.GetLocal().Bytes())
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			response := server.CodeValidationError{
				Code:        NullableString(http.StatusText(http.StatusBadRequest)),
				Description: NullableString(InvalidParamsMessage),
				ValidationFailures: &[]server.CodeValidationFailures{{
					FailureReason: NullableString("record reference not found"),
					Property:      NullableString("value"),
				}},
			}
			return ctx.JSON(http.StatusBadRequest, response)
		}
		s.logger.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}
	return ctx.JSON(http.StatusOK, server.SearchRecord{
		Meta: &struct {
			Index           *string `json:"index,omitempty"`
			ObjectReference *string `json:"object_reference,omitempty"`
		}{
			Index:           NullableString(fmt.Sprintf("%d:%d", record.PulseNumber, record.Order)),
			ObjectReference: NullableString(insolar.NewReference(*insolar.NewIDFromBytes(record.ObjectReference)).String()),
		},
		Type: NullableString("record"),
	})
}

func (s *Server) ObjectLifeline(ctx echo.Context, objectReference server.ObjectReferencePathParam, params server.ObjectLifelineParams) error {
	limit, offset, failures := checkLimitOffset(params.Limit, params.Offset)

	ref, err := checkReference(string(objectReference))
	if err != nil {
		failures = append(failures, server.CodeValidationFailures{
			FailureReason: NullableString(err.Error()),
			Property:      NullableString("object_reference"),
		})
	}

	sort := "-index"
	if params.SortBy != nil {
		s := string(*params.SortBy)
		if s != "-index" && s != "+index" {
			failures = append(failures, server.CodeValidationFailures{
				FailureReason: NullableString("should be '-index' or '+index'"),
				Property:      NullableString("sort_by"),
			})
		}
		sort = s
	}

	var fromIndexString *string
	var pulseNumberLtString *int
	var pulseNumberGtString *int
	var timestampLteString *int
	var timestampGteString *int

	if params.FromIndex != nil {
		str := string(*params.FromIndex)
		fromIndexString = &str
		_, _, err := storage.CheckIndex(str)
		if err != nil {
			failures = append(failures, server.CodeValidationFailures{
				FailureReason: NullableString("invalid"),
				Property:      NullableString("from_index"),
			})
		}
	}
	if params.PulseNumberLt != nil {
		str := int(*params.PulseNumberLt)
		pulseNumberLtString = &str
		if !pulse.IsValidAsPulseNumber(str) {
			failures = append(failures, server.CodeValidationFailures{
				FailureReason: NullableString("invalid"),
				Property:      NullableString("pulse_number_lt"),
			})
		}
	}
	if params.PulseNumberGt != nil {
		str := int(*params.PulseNumberGt)
		pulseNumberGtString = &str
		if !pulse.IsValidAsPulseNumber(str) {
			failures = append(failures, server.CodeValidationFailures{
				FailureReason: NullableString("invalid"),
				Property:      NullableString("pulse_number_gt"),
			})
		}
	}

	if failures != nil {
		apiErr := server.CodeValidationError{
			Code:               NullableString(http.StatusText(http.StatusBadRequest)),
			Message:            NullableString(InvalidParamsMessage),
			ValidationFailures: &failures,
		}
		return ctx.JSON(http.StatusBadRequest, apiErr)
	}

	if params.TimestampLte != nil {
		str := int(*params.TimestampLte)
		timestampLteString = &str
	}
	if params.TimestampGte != nil {
		str := int(*params.TimestampGte)
		timestampGteString = &str
	}

	records, count, err := s.storage.GetLifeline(
		ref.GetLocal().Bytes(),
		fromIndexString,
		pulseNumberLtString, pulseNumberGtString,
		timestampLteString, timestampGteString,
		limit, offset,
		sort,
	)
	if err != nil {
		s.logger.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	result := []server.Record{}
	for _, r := range records {
		result = append(result, RecordToAPI(r))
	}
	cnt := int64(count)
	return ctx.JSON(http.StatusOK, server.RecordsResponse{
		Total:  &cnt,
		Result: &result,
	})
}

func checkReference(referenceRow string) (*insolar.Reference, error) {
	referenceString := strings.TrimSpace(referenceRow)

	if len(referenceString) == 0 {
		return nil, errors.New("empty reference")
	}

	reference, err := url.QueryUnescape(referenceString)
	if err != nil {
		return nil, errors.New("error unescaping")
	}

	ref, err := insolar.NewReferenceFromString(reference)
	if err != nil {
		return nil, errors.New("wrong format")
	}

	return ref, nil
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
