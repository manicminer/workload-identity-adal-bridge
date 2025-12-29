package identityclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

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

func AccessToken(resource, scope, clientId string) (*TokenResponse, error) {
	azureTokenService := os.Getenv("AZURE_AUTHORITY_HOST")
	if azureTokenService == "" {
		logger.Error("environment variable `AZURE_AUTHORITY_HOST` is not set")
		return nil, fmt.Errorf("environment variable AZURE_AUTHORITY_HOST is not set")
	}

	idTokenFile := os.Getenv("AZURE_FEDERATED_TOKEN_FILE")
	if idTokenFile == "" {
		logger.Error("environment variable `AZURE_FEDERATED_TOKEN_FILE` is not set")
		return nil, fmt.Errorf("environment variable AZURE_FEDERATED_TOKEN_FILE is not set")
	}

	tenantId := os.Getenv("AZURE_TENANT_ID")
	if tenantId == "" {
		logger.Error("environment variable `AZURE_TENANT_ID` is not set")
		return nil, fmt.Errorf("environment variable AZURE_TENANT_ID is not set")
	}

	idToken, err := os.ReadFile(idTokenFile)
	if err != nil {
		logger.Error(fmt.Sprintf("reading ID token: %v", err), "path", idTokenFile)
	}

	if clientId == "" {
		clientId = os.Getenv("AZURE_CLIENT_ID")
	}
	if clientId == "" {
		logger.Error("environment variable `AZURE_CLIENT_ID` is not set, or clientId was not specified by the calling client")
		return nil, fmt.Errorf("environment variable AZURE_CLIENT_ID is not set, or clientId was not specified by the calling client")
	}

	if scope == "" {
		if resource != "" {
			scope = fmt.Sprintf("%s/.default", resource)
		} else {
			return nil, fmt.Errorf("`scope` or `resource` must be specified by the calling client")
		}
	}

	reqBody := url.Values{}
	reqBody.Set("audience", tenantId)
	reqBody.Set("client_assertion", string(idToken))
	reqBody.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	reqBody.Set("client_id", clientId)
	reqBody.Set("grant_type", "client_credentials")
	reqBody.Set("scope", scope)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s/oauth2/v2.0/token", azureTokenService, tenantId), strings.NewReader(reqBody.Encode()))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error(fmt.Sprintf("requesting access token: %v", err), "values", reqBody.Encode())
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
		logger.Error(fmt.Sprintf("unmarshalling response from token service: %v", err), "response", string(body))
		return nil, err
	}

	out.ClientID = clientId
	out.Resource = resource
	out.ExpiresOn = time.Now().Unix() + out.ExpiresIn

	return &out, nil
}
