// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md
//

package generator

import (
	"math/big"
	"net/http"

	"github.com/insolar/common-test/apierrors"
	"github.com/insolar/x-crypto/ecdsa"
)

// signature

type UserSignature struct {
	PublicKey     ecdsa.PublicKey
	PrivateKey    *ecdsa.PrivateKey
	X509PublicKey []byte
	PemPublicKey  []byte
}

type EcdsaSignature struct {
	R, S *big.Int
}

type KeysFormat struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

// response struct

type APIResult struct {
	Http         *http.Response
	RequestBody  interface{}
	ResponseBody interface{}
	Error        error
}

// for data table tests

type GeneralTestCase struct {
	TestName          string
	PreconditionValue interface{}
	ExpectedValue     interface{}
	CaseDescription   string
}

type NegativeTestCase struct {
	TestName        string
	Value           interface{}
	CaseDescription string
	Error           apierrors.ApiError
	ErrorTrace      string
	IssueLink       string
}
