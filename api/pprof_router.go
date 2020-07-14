// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/pkg/errors"
)

func NewRouter() *Router {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "OK")
	})

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	hs := &http.Server{Addr: ":8000", Handler: mux}

	r := &Router{
		hs: hs,
	}

	return r
}

type PromHTTPLoggerAdapter struct {
	log.Logger
}

func (o PromHTTPLoggerAdapter) Println(v ...interface{}) {
	o.Error(v)
}

type Router struct {
	hs *http.Server
}

func (r *Router) Start(ctx context.Context) error {
	logger := belogger.FromContext(ctx)
	go func() {
		logger.Debugf("starting http: %+v", r.hs)
		err := r.hs.ListenAndServe()
		if err != http.ErrServerClosed {
			logger.Error(errors.Wrapf(err, "http server ListenAndServe"))
		}
	}()

	return nil
}

func (r *Router) Stop(ctx context.Context) error {
	logger := belogger.FromContext(ctx)
	if err := r.hs.Shutdown(ctx); err != nil {
		logger.Error(errors.Wrapf(err, "http server shutdown"))
		return err
	}
	return nil
}
