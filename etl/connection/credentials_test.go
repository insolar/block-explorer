// +build unit

package connection

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/stretchr/testify/require"
)

func TestTokenCredentials_GetRequestMetadata(t *testing.T) {
	getMethod := "GET"
	login := "test_login"
	password := "test_pass"
	expectedToken := "test-access-token"
	expectedExpiresIn := time.Now().Add(time.Hour).Unix()
	expectedResponse := fmt.Sprintf(`{"access_token": "%s", "expires_in": %d}`, expectedToken, expectedExpiresIn)
	refreshOffset := int64(60)

	t.Run("fresh_token", func(t *testing.T) {
		cred := newTokenCredentials(nil, "", "", "", refreshOffset, true)
		cred.token = Token{
			AccessToken: expectedToken,
			ExpiresIn:   int64(61),
			ReceivedAt:  time.Now(),
		}
		expected := fmt.Sprintf("Bearer %s", expectedToken)

		metadata, err := cred.GetRequestMetadata(belogger.TestContext(t), "")
		require.NoError(t, err)
		require.Equal(t, expected, metadata["Authorization"])
	})

	t.Run("empty_token", func(t *testing.T) {
		expected := fmt.Sprintf("Bearer %s", expectedToken)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, getMethod, r.Method)
			u, p, ok := r.BasicAuth()
			require.True(t, ok)
			require.Equal(t, login, u)
			require.Equal(t, password, p)

			w.WriteHeader(200)
			_, err := w.Write([]byte(expectedResponse))
			require.NoError(t, err)
		}))
		defer server.Close()

		cred := newTokenCredentials(server.Client(), server.URL, login, password, refreshOffset, true)
		metadata, err := cred.GetRequestMetadata(belogger.TestContext(t), "")
		require.NoError(t, err)
		require.Equal(t, expected, metadata["Authorization"])
	})

	t.Run("expired_token", func(t *testing.T) {
		expected := fmt.Sprintf("Bearer %s", expectedToken)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, getMethod, r.Method)
			u, p, ok := r.BasicAuth()
			require.True(t, ok)
			require.Equal(t, login, u)
			require.Equal(t, password, p)

			w.WriteHeader(200)
			_, err := w.Write([]byte(expectedResponse))
			require.NoError(t, err)
		}))
		defer server.Close()

		cred := newTokenCredentials(server.Client(), server.URL, login, password, refreshOffset, true)
		cred.token = Token{
			AccessToken: "expired-token",
			ExpiresIn:   int64(60),
			ReceivedAt:  time.Now(),
		}
		metadata, err := cred.GetRequestMetadata(belogger.TestContext(t), "")
		require.NoError(t, err)
		require.Equal(t, expected, metadata["Authorization"])
	})

	t.Run("auth_response_error", func(t *testing.T) {
		expectedErrorCode := 500
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, getMethod, r.Method)
			u, p, ok := r.BasicAuth()
			require.True(t, ok)
			require.Equal(t, login, u)
			require.Equal(t, password, p)

			w.WriteHeader(expectedErrorCode)
		}))
		defer server.Close()

		cred := newTokenCredentials(server.Client(), server.URL, login, password, refreshOffset, true)
		metadata, err := cred.GetRequestMetadata(belogger.TestContext(t), "")
		require.Error(t, err)
		require.Contains(t, err.Error(), fmt.Sprintf("%d status code", expectedErrorCode))
		require.Empty(t, metadata["Authorization"])
	})

	t.Run("malformed_response_body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, getMethod, r.Method)
			u, p, ok := r.BasicAuth()
			require.True(t, ok)
			require.Equal(t, login, u)
			require.Equal(t, password, p)

			w.WriteHeader(200)
			_, err := w.Write([]byte("not a json string"))
			require.NoError(t, err)
		}))
		defer server.Close()

		cred := newTokenCredentials(server.Client(), server.URL, login, password, refreshOffset, true)
		metadata, err := cred.GetRequestMetadata(belogger.TestContext(t), "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse authorization response body")
		require.Empty(t, metadata["Authorization"])
	})
}
