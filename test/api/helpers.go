// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/insolar/block-explorer/testutils/connectionmanager"
)

func LogHTTP(t *testing.T, http *http.Response, requestBody interface{}, responseBody interface{}) {
	var buf strings.Builder
	buf.WriteString("\n")
	buf.WriteString("Request:")
	buf.WriteString(fmt.Sprintf("%v %v://%v%v \n",
		http.Request.Method, http.Request.URL.Scheme, http.Request.URL.Host, http.Request.URL.Path))

	headers := http.Request.Header
	if len(headers) > 0 {
		for k, v := range headers {
			buf.WriteString(fmt.Sprintf("  -H %v: %v\n", k, v))
		}
	}

	if requestBody != nil {
		bytes, e := json.MarshalIndent(requestBody, "", "    ")
		if e != nil {
			t.Fatal(e)
		}
		buf.WriteString(fmt.Sprintf("request body:\n%v\n", string(bytes)))
	}

	buf.WriteString("Received response:\n")
	buf.WriteString(fmt.Sprintf("http status: %s\n", http.Status))
	bytes, e := json.MarshalIndent(responseBody, "", "    ")
	if e != nil {
		t.Fatal(e)
	}
	buf.WriteString(fmt.Sprintf("response body:\n%v", string(bytes)))

	t.Log(buf.String())
}

func GetHTTPClient() *BEApiClient {
	return NewBeAPIClient(fmt.Sprintf("http://localhost%v", connectionmanager.DefaultAPIPort))
}
