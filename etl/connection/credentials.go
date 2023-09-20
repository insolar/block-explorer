package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type Token struct {
	AccessToken string    `json:"access_token"`
	ExpiresIn   int64     `json:"expires_in"`
	ReceivedAt  time.Time `json:"-"`
}

type tokenCredentials struct {
	authURL       string
	login         string
	pass          string
	refreshOffset int64
	insecureTLS   bool
	token         Token

	tokenMx    sync.RWMutex
	httpClient *http.Client
}

func newTokenCredentials(client *http.Client, authURL, login, pass string, refreshOffset int64, insecureTLS bool) *tokenCredentials {
	return &tokenCredentials{
		authURL:       authURL,
		login:         login,
		pass:          pass,
		refreshOffset: refreshOffset,
		insecureTLS:   insecureTLS,
		httpClient:    client,
	}
}

// GetRequestMetadata gets the current request metadata, refreshing tokens if required.
func (c *tokenCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	if c.needNewToken() {
		err := c.receiveAccessToken(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "can't get access_token")
		}
	}

	return map[string]string{"Authorization": fmt.Sprintf("Bearer %s", c.token.AccessToken)}, nil
}

// RequireTransportSecurity indicates whether the credentials requires transport security.
func (c *tokenCredentials) RequireTransportSecurity() bool {
	return !c.insecureTLS
}

func (c *tokenCredentials) needNewToken() bool {
	c.tokenMx.RLock()
	defer c.tokenMx.RUnlock()

	if c.token.AccessToken == "" {
		return true
	}

	timeToRefresh := c.token.ReceivedAt.Add(time.Duration(c.token.ExpiresIn-c.refreshOffset) * time.Second)
	return timeToRefresh.Before(time.Now())
}

func (c *tokenCredentials) receiveAccessToken(ctx context.Context) error {
	c.tokenMx.Lock()
	defer c.tokenMx.Unlock()

	request, err := http.NewRequest("GET", c.authURL, nil)
	if err != nil {
		return err
	}
	request.SetBasicAuth(c.login, c.pass)
	request = request.WithContext(ctx)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return errors.Wrap(err, "failed to do http request")
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	statusCode := response.StatusCode
	if statusCode != http.StatusOK {
		return fmt.Errorf("API path='%s' return %v status code", request.URL, statusCode)
	}

	token := Token{}
	err = json.Unmarshal(body, &token)
	if err != nil {
		return fmt.Errorf("failed to parse authorization response body %v", err)
	}

	token.ReceivedAt = time.Now()
	c.token = token
	return nil
}
