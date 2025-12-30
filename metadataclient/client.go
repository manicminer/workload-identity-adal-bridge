package metadataclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/manicminer/workload-identity-adal-bridge/internal/logger"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	ClientID     string `json:"client_id"`
	Resource     string `json:"resource"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	ExpiresOn    int64  `json:"expires_on"`
	ExtExpiresIn int64  `json:"ext_expires_in"`
}

func AccessToken(ctx context.Context, metadataUrl, resource, scope, clientId string) (*TokenResponse, error) {
	if clientId == "" {
		return nil, fmt.Errorf("clientId was not specified")
	}

	if resource == "" {
		if scope != "" {
			u, err := url.Parse(scope)
			if err != nil {
				return nil, fmt.Errorf("parsing scope: %v", err)
			}
			if u.Host == "" {
				return nil, fmt.Errorf("invalid scope specified")
			}
			resource = fmt.Sprintf("https://%s", u.Host)
		} else {
			return nil, fmt.Errorf("`scope` or `resource` must be specified")
		}
	}

	query := url.Values{}
	query.Set("api-version", "2022-09-01")
	query.Set("client_id", clientId)
	query.Set("resource", resource)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/metadata/identity/oauth2/token?%s", metadataUrl, query.Encode()), nil)
	req.Header.Add("Accept", "application/json")
	req.URL.Query()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error(fmt.Sprintf("requesting access token: %v", err), "client_id", clientId, "resource", resource)
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("reading response body: %v", err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d received with body: %v", resp.StatusCode, string(body))
	}

	out := TokenResponse{}
	if err = json.Unmarshal(body, &out); err != nil {
		logger.Error(fmt.Sprintf("unmarshalling response from instance metadata service: %v", err), "response_body", string(body))
		return nil, err
	}

	return &out, nil
}
