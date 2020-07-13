// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package connection

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"google.golang.org/grpc/credentials"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type GRPCClientConnection struct {
	grpc *grpc.ClientConn
}

// NewGRPCClientConnection returns implementation
func NewGRPCClientConnection(ctx context.Context, cfg configuration.Replicator) (*GRPCClientConnection, error) {
	log := belogger.FromContext(ctx)
	c, e := func() (*grpc.ClientConn, error) {
		limits := grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(cfg.MaxTransportMsg),
			grpc.MaxCallSendMsgSize(cfg.MaxTransportMsg),
			// bug solving. "wait for ready" true block the connection until the server return data.
			// we are reading the data faster than the server can send
			grpc.WaitForReady(true),
		)
		log.Infof("trying connect to %s...", cfg.Addr)

		options := []grpc.DialOption{limits, grpc.WithInsecure()}
		if cfg.Auth.Required {
			log.Info("replicator auth is required, preparing auth options")
			cp, err := x509.SystemCertPool()
			if err != nil {
				return nil, errors.Wrapf(err, "failed get x509 SystemCertPool")
			}
			httpClient := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs: cp,
						// nolint:gosec
						InsecureSkipVerify: cfg.Auth.InsecureTLS,
					},
				},
				Timeout: cfg.Auth.Timeout,
			}
			perRPCCred := grpc.WithPerRPCCredentials(newTokenCredentials(httpClient, cfg.Auth.URL,
				cfg.Auth.Login, cfg.Auth.Password,
				cfg.Auth.RefreshOffset, cfg.Auth.InsecureTLS))

			tlsOption := grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(cp, ""))
			if cfg.Auth.InsecureTLS {
				tlsOption = grpc.WithInsecure()
			}

			options = []grpc.DialOption{limits, tlsOption, perRPCCred}
		}

		// We omit error here because connect happens in background.
		conn, err := grpc.Dial(cfg.Addr, options...)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to grpc.Dial")
		}
		return conn, err
	}()

	if e != nil {
		return &GRPCClientConnection{}, e
	}

	return &GRPCClientConnection{c}, nil
}

func (c *GRPCClientConnection) GetGRPCConn() *grpc.ClientConn {
	return c.grpc
}

func GetClientConfiguration(addr string) configuration.Replicator {
	return configuration.Replicator{
		Addr:            addr,
		MaxTransportMsg: 100500,
	}
}
