package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
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

// jetIDRegexp uses for a validation of the JetID
var jetIDRegexp = regexp.MustCompile(`^(\*|([0-1]{1,216}))$`)

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

func (s *Server) JetDropByID(ctx echo.Context, jetDropID server.JetDropIdPath) error {
	exporterJetDropID, err := models.NewJetDropIDFromString(string(jetDropID))
	if err != nil {
		response := server.CodeValidationError{
			Code:        NullableString(strconv.Itoa(http.StatusBadRequest)),
			Description: nil,
			Message:     NullableString(InvalidParamsMessage),
			ValidationFailures: &[]server.CodeValidationFailures{{
				FailureReason: NullableString(errors.Wrapf(err, "invalid").Error()),
				Property:      NullableString("jet drop id"),
			}},
		}
		return ctx.JSON(http.StatusBadRequest, response)
	}
	jetDrop, prevJetDrop, nextJetDrops, err := s.storage.GetJetDropByID(
		*exporterJetDropID,
	)
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return ctx.JSON(http.StatusNotFound, struct{}{})
		}
		s.logger.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	nextJetDropID := []server.NextPrevJetDrop{}
	prevJetDropID := []server.NextPrevJetDrop{}
	for _, jetDrop := range prevJetDrop {
		prevJetDropID = append(prevJetDropID, transformPrevNextResp(jetDrop))
	}
	for _, jetDrop := range nextJetDrops {
		nextJetDropID = append(nextJetDropID, transformPrevNextResp(jetDrop))
	}

	apiJetDrop := JetDropToAPI(jetDrop, prevJetDropID, nextJetDropID)
	return ctx.JSON(http.StatusOK, server.JetDropResponse(apiJetDrop))
}

func (s *Server) JetDropRecords(ctx echo.Context, jetDropID server.JetDropIdPath, params server.JetDropRecordsParams) error {
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

func (s *Server) JetDropsByJetID(ctx echo.Context, jetID server.JetIdPath, params server.JetDropsByJetIDParams) error {
	var failures []server.CodeValidationFailures
	limit, _, failures := checkLimitOffset(params.Limit, nil)
	// FIXME here PENV-505
	if params.Limit == nil {
		limit = 1000
	}
	id, validationError := checkJetID(jetID)
	if validationError != nil {
		failures = append(failures, validationError...)
	}

	sortByAsc, validationError := checkSortByPulseParameter(params.SortBy)
	if validationError != nil {
		failures = append(failures, validationError...)
	}

	var pulseNumberLte, pulseNumberLt, pulseNumberGte, pulseNumberGt *int64
	if params.PulseNumberGt != nil {
		pulseNumberGt, failures = getPulseNumberValue(int(*params.PulseNumberGt), "pulse_number_gt", failures)
	}

	if params.PulseNumberGte != nil {
		pulseNumberGte, failures = getPulseNumberValue(int(*params.PulseNumberGte), "pulse_number_gte", failures)
	}

	if params.PulseNumberLt != nil {
		pulseNumberLt, failures = getPulseNumberValue(int(*params.PulseNumberLt), "pulse_number_lt", failures)
	}

	if params.PulseNumberLte != nil {
		pulseNumberLte, failures = getPulseNumberValue(int(*params.PulseNumberLte), "pulse_number_lte", failures)
	}

	if len(failures) > 0 {
		apiErr := server.CodeValidationError{
			Code:               NullableString(http.StatusText(http.StatusBadRequest)),
			Message:            NullableString(InvalidParamsMessage),
			ValidationFailures: &failures,
		}
		return ctx.JSON(http.StatusBadRequest, apiErr)
	}

	jetDrops, total, err := s.storage.GetJetDropsByJetID(id, pulseNumberLte, pulseNumberLt, pulseNumberGte, pulseNumberGt, limit, sortByAsc)
	if gorm.IsRecordNotFoundError(err) || len(jetDrops) == 0 {
		cnt := int64(0)
		drops := []server.JetDrop{}
		return ctx.JSON(http.StatusOK, server.JetDropsResponse{
			Total:  &cnt,
			Result: &drops,
		})
	}
	if err != nil {
		s.logger.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	firstPN, lastPN := s.findEdgePNInJetDrops(jetDrops, sortByAsc)
	enrichedJetDrops, err := s.getEnrichingJetDrops(firstPN, lastPN)
	if err != nil {
		s.logger.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}
	enrichedJetDrops = append(enrichedJetDrops, jetDrops...)
	jetDropsByHash, jetDropsByPrevHash := s.createNextPrevMaps(enrichedJetDrops)

	cnt := int64(total)
	drops := make([]server.JetDrop, len(jetDrops))
	for i, jetDrop := range jetDrops {
		nextJetDrops := []server.NextPrevJetDrop{}
		prevJetDrops := []server.NextPrevJetDrop{}

		prev, ok := jetDropsByHash[base64.StdEncoding.EncodeToString(jetDrop.FirstPrevHash)]
		if ok {
			prevJetDrops = append(prevJetDrops, prev)
		}
		prev, ok = jetDropsByHash[base64.StdEncoding.EncodeToString(jetDrop.SecondPrevHash)]
		if ok {
			prevJetDrops = append(prevJetDrops, prev)
		}

		next, ok := jetDropsByPrevHash[base64.StdEncoding.EncodeToString(jetDrop.Hash)]
		if ok {
			nextJetDrops = append(nextJetDrops, next...)
		}

		drops[i] = JetDropToAPI(jetDrop, prevJetDrops, nextJetDrops)
	}
	return ctx.JSON(http.StatusOK, server.JetDropsResponse{
		Total:  &cnt,
		Result: &drops,
	})
}

func (s *Server) getEnrichingJetDrops(oldestPulse, newestPulse int64) ([]models.JetDrop, error) {
	var enrichedDrops []models.JetDrop
	emptyPulse := models.Pulse{}

	pulse, err := s.storage.GetPulse(oldestPulse)
	if err == nil && pulse.PrevPulseNumber > 0 {
		ejd, err := s.storage.GetJetDrops(models.Pulse{PulseNumber: pulse.PrevPulseNumber})
		if err != nil {
			return []models.JetDrop{}, errors.Wrapf(err, "can't enrich jetDrops for pulse %d, from oldestPulse %d", pulse.PrevPulseNumber, oldestPulse)
		}
		enrichedDrops = append(enrichedDrops, ejd...)
	}

	nextPulse, err := s.storage.GetNextSavedPulse(models.Pulse{PulseNumber: newestPulse}, false)
	if err == nil && nextPulse != emptyPulse {
		ejd, err := s.storage.GetJetDrops(models.Pulse{PulseNumber: nextPulse.PulseNumber})
		if err != nil {
			return []models.JetDrop{}, errors.Wrapf(err, "can't enrich jetDrops for pulse %d, from newestPulse %d", nextPulse.PulseNumber, newestPulse)
		}
		enrichedDrops = append(enrichedDrops, ejd...)
	}

	return enrichedDrops, nil
}

func (s *Server) createNextPrevMaps(jetDrops []models.JetDrop) (map[string]server.NextPrevJetDrop, map[string][]server.NextPrevJetDrop) {
	jetDropsByHash := map[string]server.NextPrevJetDrop{}
	for _, jetDrop := range jetDrops {
		key := base64.StdEncoding.EncodeToString(jetDrop.Hash)
		jetDropsByHash[key] = transformPrevNextResp(jetDrop)
	}
	jetDropsByPrevHash := map[string][]server.NextPrevJetDrop{}
	for _, jetDrop := range jetDrops {
		add := func(hash []byte) {
			key := base64.StdEncoding.EncodeToString(hash)
			savedDrops, ok := jetDropsByPrevHash[key]
			if !ok {
				jetDropsByPrevHash[key] = append(jetDropsByPrevHash[key], transformPrevNextResp(jetDrop)) // nolint
				return
			}

			apiNextPrevDrop := transformPrevNextResp(jetDrop) // nolint
			for _, d := range savedDrops {
				if *d.JetDropId == *apiNextPrevDrop.JetDropId {
					return
				}
			}
			jetDropsByPrevHash[key] = append(jetDropsByPrevHash[key], apiNextPrevDrop)
		}
		add(jetDrop.FirstPrevHash)
		// add(jetDrop.SecondPrevHash)
	}
	return jetDropsByHash, jetDropsByPrevHash
}

func getPulseNumberValue(unptr int, propertyName string, failures []server.CodeValidationFailures) (*int64, []server.CodeValidationFailures) {
	_int64 := int64(unptr)
	pulseNumber := &_int64
	if !pulse.IsValidAsPulseNumber(unptr) {
		failures = append(failures, server.CodeValidationFailures{
			FailureReason: NullableString("invalid value"),
			Property:      NullableString(propertyName),
		})
	}
	return pulseNumber, failures
}

func (s *Server) Pulses(ctx echo.Context, params server.PulsesParams) error {
	limit, offset, failures := checkLimitOffset(params.Limit, params.Offset)

	var fromPulseString *int64
	var timestampLte *int64
	var timestampGte *int64

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

	var pulseNumberLte, pulseNumberLt, pulseNumberGte, pulseNumberGt *int64
	if params.PulseNumberGt != nil {
		pulseNumberGt, failures = getPulseNumberValue(int(*params.PulseNumberGt), "pulse_number_gt", failures)
	}
	if params.PulseNumberGte != nil {
		pulseNumberGte, failures = getPulseNumberValue(int(*params.PulseNumberGte), "pulse_number_gte", failures)
	}
	if params.PulseNumberLt != nil {
		pulseNumberLt, failures = getPulseNumberValue(int(*params.PulseNumberLt), "pulse_number_lt", failures)
	}
	if params.PulseNumberLte != nil {
		pulseNumberLte, failures = getPulseNumberValue(int(*params.PulseNumberLte), "pulse_number_lte", failures)
	}

	sortByAsc, validationFailures := checkSortByPulseNumberParameter(params.SortBy)
	if validationFailures != nil {
		failures = append(failures, validationFailures...)
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
		str := int64(*params.TimestampLte)
		timestampLte = &str
	}
	if params.TimestampGte != nil {
		str := int64(*params.TimestampGte)
		timestampGte = &str
	}

	pulses, count, err := s.storage.GetPulses(
		fromPulseString,
		timestampLte, timestampGte,
		pulseNumberLte, pulseNumberLt, pulseNumberGte, pulseNumberGt,
		sortByAsc,
		limit, offset,
	)
	if err != nil {
		s.logger.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	result := []server.Pulse{}
	for _, p := range pulses {
		result = append(result, PulseToAPI(p))
	}
	cnt := int64(count)
	return ctx.JSON(http.StatusOK, server.PulsesResponse{
		Total:  &cnt,
		Result: &result,
	})
}

func (s *Server) Pulse(ctx echo.Context, pulseNumber server.PulseNumberPath) error {
	pulse, err := s.storage.GetPulse(int64(pulseNumber))
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return ctx.JSON(http.StatusNotFound, struct{}{})
		}
		err = errors.Wrapf(err, "error while select pulse from db by pulse number %d", pulseNumber)
		s.logger.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	pulseResponse := PulseToAPI(pulse)
	return ctx.JSON(http.StatusOK, pulseResponse)
}

func (s *Server) JetDropsByPulseNumber(ctx echo.Context, pulseNumber server.PulseNumberPath, params server.JetDropsByPulseNumberParams) error {
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
			Message:            NullableString(InvalidParamsMessage),
			ValidationFailures: &failures,
		}
		return ctx.JSON(http.StatusBadRequest, response)
	}

	jetDrops, total, err := s.storage.GetJetDropsWithParams(
		models.Pulse{PulseNumber: int64(pulseNumber)},
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
		drops[i] = JetDropToAPI(jetDrop, []server.NextPrevJetDrop{}, []server.NextPrevJetDrop{})
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

func (s *Server) ObjectLifeline(ctx echo.Context, objectReference server.ObjectReferencePath, params server.ObjectLifelineParams) error {
	limit, offset, failures := checkLimitOffset(params.Limit, params.Offset)

	ref, err := checkReference(string(objectReference))
	if err != nil {
		failures = append(failures, server.CodeValidationFailures{
			FailureReason: NullableString(err.Error()),
			Property:      NullableString("object_reference"),
		})
	}

	sortAsc := string(server.SortByIndex_index_asc)
	sortDesc := string(server.SortByIndex_index_desc)
	var sortByIndexAsc bool
	if params.SortBy != nil {
		s := string(*params.SortBy)
		if s != sortDesc && s != sortAsc {
			failures = append(failures, server.CodeValidationFailures{
				FailureReason: NullableString(fmt.Sprintf("should be '%s' or '%s'", sortDesc, sortAsc)),
				Property:      NullableString("sort_by"),
			})
		}
		if s == sortAsc {
			sortByIndexAsc = true
		}
	}

	var fromIndexString *string
	var pulseNumberLt *int64
	var pulseNumberGt *int64
	var timestampLte *int64
	var timestampGte *int64

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
	if params.PulseNumberGt != nil {
		pulseNumberGt, failures = getPulseNumberValue(int(*params.PulseNumberGt), "pulse_number_gt", failures)
	}
	if params.PulseNumberLt != nil {
		pulseNumberLt, failures = getPulseNumberValue(int(*params.PulseNumberLt), "pulse_number_lt", failures)
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
		unptr := int64(*params.TimestampLte)
		timestampLte = &unptr
	}
	if params.TimestampGte != nil {
		unptr := int64(*params.TimestampGte)
		timestampGte = &unptr
	}

	records, count, err := s.storage.GetLifeline(
		ref.GetLocal().Bytes(),
		fromIndexString,
		pulseNumberLt, pulseNumberGt,
		timestampLte, timestampGte,
		limit, offset,
		sortByIndexAsc,
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

func (s *Server) findEdgePNInJetDrops(jetDrops []models.JetDrop, sortByAsc bool) (int64, int64) {
	if len(jetDrops) == 0 {
		return 0, 0
	}
	if sortByAsc {
		return jetDrops[0].PulseNumber, jetDrops[len(jetDrops)-1].PulseNumber
	}
	return jetDrops[len(jetDrops)-1].PulseNumber, jetDrops[0].PulseNumber
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

func checkLimitOffset(l *server.Limit, o *server.OffsetParam) (int, int, []server.CodeValidationFailures) {
	var failures []server.CodeValidationFailures
	limit := 20
	if l != nil {
		limit = int(*l)
	}
	if limit <= 0 || limit > 1000 {
		failures = append(failures, server.CodeValidationFailures{
			FailureReason: NullableString("should be in range [1, 1000]"),
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

func checkSortByPulseParameter(sortBy *server.SortByPulse) (bool, []server.CodeValidationFailures) {
	pnAsc := string(server.SortByPulse_pulse_number_asc_jet_id_desc)
	pnDesc := string(server.SortByPulse_pulse_number_desc_jet_id_asc)
	var sortByPnAsc bool
	if sortBy != nil {
		s := string(*sortBy)
		if s != pnAsc && s != pnDesc {
			errResponse := []server.CodeValidationFailures{
				{
					Property:      NullableString("sort_by"),
					FailureReason: NullableString(fmt.Sprintf("query parameter 'sort_by' should be '%s' or '%s'", pnAsc, pnDesc)),
				},
			}
			return false, errResponse
		}
		if s == pnAsc {
			sortByPnAsc = true
		}
	}
	return sortByPnAsc, nil
}

func checkSortByPulseNumberParameter(sortBy *server.SortByPulseNumber) (bool, []server.CodeValidationFailures) {
	pnAsc := string(server.SortByPulseNumber_pulse_number_asc)
	pnDesc := string(server.SortByPulseNumber_pulse_number_desc)
	var sortByPnAsc bool
	if sortBy != nil {
		s := string(*sortBy)
		if s != pnAsc && s != pnDesc {
			errResponse := []server.CodeValidationFailures{
				{
					Property:      NullableString("sort_by"),
					FailureReason: NullableString(fmt.Sprintf("query parameter 'sort_by' should be '%s' or '%s'", pnAsc, pnDesc)),
				},
			}
			return false, errResponse
		}
		if s == pnAsc {
			sortByPnAsc = true
		}
	}
	return sortByPnAsc, nil
}

func checkJetID(jetID server.JetIdPath) (string, []server.CodeValidationFailures) {
	var failures []server.CodeValidationFailures

	value := strings.TrimSpace(string(jetID))

	if len(value) == 0 {
		failures = append(failures, server.CodeValidationFailures{
			Property:      NullableString("jet-id path parameter"),
			FailureReason: NullableString("empty value of path parameter"),
		})
	}

	id, err := url.QueryUnescape(value)
	if err != nil {
		failures = append(failures, server.CodeValidationFailures{
			Property:      NullableString("jet-id path parameter"),
			FailureReason: NullableString(errors.Wrapf(err, "cannot unescape path parameter jet-id").Error()),
		})
	}

	if !jetIDRegexp.MatchString(id) {
		failures = append(failures, server.CodeValidationFailures{
			Property:      NullableString("jet-id path parameter"),
			FailureReason: NullableString("parameter does not match with jetID valid value"),
		})
	}

	if failures != nil {
		return "", failures
	}
	if id == "*" {
		return "", nil
	}
	return id, nil
}

func transformPrevNextResp(jetDrop models.JetDrop) server.NextPrevJetDrop {
	jd := models.NewJetDropID(jetDrop.JetID, jetDrop.PulseNumber)
	return server.NextPrevJetDrop{
		JetId:       NullableString(jd.JetIDToString()),
		JetDropId:   NullableString(jd.ToString()),
		PulseNumber: &jd.PulseNumber,
	}
}
