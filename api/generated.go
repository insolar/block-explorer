// Package api provides primitives to interact the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen DO NOT EDIT.
package api

import (
	"fmt"
	"net/http"

	"github.com/deepmap/oapi-codegen/pkg/runtime"
	"github.com/labstack/echo/v4"
)

type ResponsesLifelineYaml struct {
	Total  int64                  `json:"total"`
	Result *[]ResponsesRecordYaml `json:"result,omitempty"`
}

// ResponsesRecordYaml defines model for responses-record-yaml.
type ResponsesRecordYaml struct {
	Hash                *string `json:"hash,omitempty"`
	JetDropId           *string `json:"jet_drop_id,omitempty"`
	JetId               *string `json:"jet_id,omitempty"`
	ObjectReference     *string `json:"object_reference,omitempty"`
	Index               *string `json:"index,omitempty"`
	Payload             *string `json:"payload,omitempty"`
	PrevRecordReference *string `json:"prev_record_reference,omitempty"`
	PrototypeReference  *string `json:"prototype_reference,omitempty"`
	PulseNumber         *int64  `json:"pulse_number,omitempty"`
	Reference           *string `json:"reference,omitempty"`
	Timestamp           *int64  `json:"timestamp,omitempty"`
	Type                *string `json:"type,omitempty"`
}

// ObjectLifelineParams defines parameters for ObjectLifeline.
type ObjectLifelineParams struct {
	// The numbers of items to return.
	Limit *int `json:"limit,omitempty"`

	// The number of items to skip before starting to collect the result set.
	Offset *int `json:"offset,omitempty"`

	PulseNumberLt *int `json:"pulse_number_lt,omitempty"`

	PulseNumberGt *int `json:"pulse_number_gt,omitempty"`

	Sort *string `json:"sort,omitempty"`

	FromIndex *string `json:"from_index,omitempty"`

	// The record type.
	Type *string `json:"type,omitempty"`
}

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Object Lifeline// (GET /api/v1/lifeline/{object_reference}/records)
	ObjectLifeline(ctx echo.Context, objectReference string, params ObjectLifelineParams) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// ObjectLifeline converts echo context to params.
func (w *ServerInterfaceWrapper) ObjectLifeline(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "object_reference" -------------
	var objectReference string

	err = runtime.BindStyledParameter("simple", false, "object_reference", ctx.Param("object_reference"), &objectReference)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter object_reference: %s", err))
	}

	// Parameter object where we will unmarshal all parameters from the context
	var params ObjectLifelineParams
	// ------------- Optional query parameter "limit" -------------

	err = runtime.BindQueryParameter("form", true, false, "limit", ctx.QueryParams(), &params.Limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter limit: %s", err))
	}

	// ------------- Optional query parameter "offset" -------------

	err = runtime.BindQueryParameter("form", true, false, "offset", ctx.QueryParams(), &params.Offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter offset: %s", err))
	}

	// ------------- Optional query parameter "pulse_number_lt" -------------

	err = runtime.BindQueryParameter("form", true, false, "pulse_number_lt", ctx.QueryParams(), &params.PulseNumberLt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter pulse_number_lt: %s", err))
	}

	// ------------- Optional query parameter "pulse_number_gt" -------------
	err = runtime.BindQueryParameter("form", true, false, "pulse_number_gt", ctx.QueryParams(), &params.PulseNumberGt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter pulse_number_gt: %s", err))
	}

	// ------------- Optional query parameter "from_index" -------------

	err = runtime.BindQueryParameter("form", true, false, "from_index", ctx.QueryParams(), &params.FromIndex)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter from_index: %s", err))
	}

	// ------------- Optional query parameter "sort" -------------

	err = runtime.BindQueryParameter("form", true, false, "sort", ctx.QueryParams(), &params.Sort)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter sort: %s", err))
	}

	// ------------- Optional query parameter "type" -------------

	err = runtime.BindQueryParameter("form", true, false, "type", ctx.QueryParams(), &params.Type)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter type: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.ObjectLifeline(ctx, objectReference, params)
	return err
}

// This is a simple interface which specifies echo.Route addition functions which
// are present on both echo.Echo and echo.Group, since we want to allow using
// either of them for path registration
type EchoRouter interface {
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router EchoRouter, si ServerInterface) {

	wrapper := ServerInterfaceWrapper{
		Handler: si,
	}

	router.GET("/api/v1/lifeline/:object_reference/records", wrapper.ObjectLifeline)
}
