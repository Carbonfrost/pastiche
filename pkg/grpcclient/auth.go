// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grpcclient

import (
	"context"
	"encoding/base64"

	"github.com/Carbonfrost/pastiche/pkg/model"
	"google.golang.org/grpc"
)

type headerAuthCreds struct {
	headers map[string]string
}

func (b *headerAuthCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return b.headers, nil
}

func (b *headerAuthCreds) RequireTransportSecurity() bool {
	return false // Set to true if using TLS
}

func withAuth(a model.Auth) grpc.DialOption {
	if a == nil {
		return nil
	}
	switch auth := a.(type) {
	case *model.BasicAuth:
		return basicAuth(auth.User, auth.Password)
	}
	return nil
}

func basicAuth(username, password string) grpc.DialOption {
	auth := username + ":" + password
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	return grpc.WithPerRPCCredentials(&headerAuthCreds{headers: map[string]string{
		"authorization": "Basic " + encodedAuth,
	}})
}
