package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
)

const (
	defaultHttpPort     = 8080
	defaultHttpsPort    = 8443
	defaultTlsCert      = "server.pem"
	defaultTlsKey       = "server.key"
	serviceName         = "workload-identity-adal-bridge"
	serviceFriendlyName = "Workload Identity ADAL Bridge Service"
)

var (
	httpPort, httpsPort     int
	logger                  hclog.Logger
	tlsEnabled              bool
	tlsCertPath, tlsKeyPath string
	version                 = "development"
	wg                      sync.WaitGroup
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	ClientID     string `json:"client_id"`
	Resource     string `json:"resource"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	ExpiresOn    int64  `json:"expires_on"`
	ExtExpiresIn int64  `json:"ext_expires_in"`
}

func main() {
	logger = hclog.New(&hclog.LoggerOptions{
		Name:  serviceName,
		Level: hclog.LevelFromString(os.Getenv("LOG_LEVEL")),
	})

	fmt.Printf(fmt.Sprintf("%s %s\n", serviceName, version))

	flag.IntVar(&httpPort, "http-port", defaultHttpPort, fmt.Sprintf("HTTP port to listen on (default: %d)", defaultHttpPort))
	flag.IntVar(&httpsPort, "https-port", defaultHttpsPort, fmt.Sprintf("HTTPS port to listen on (default: %d)", defaultHttpsPort))
	flag.BoolVar(&tlsEnabled, "enable-tls", false, "enable TLS (default: false)")
	flag.StringVar(&tlsCertPath, "tls-cert", defaultTlsCert, fmt.Sprintf("path to PEM-encoded TLS certificate (default: %s)", defaultTlsCert))
	flag.StringVar(&tlsKeyPath, "tls-key", defaultTlsKey, fmt.Sprintf("path to PEM-encoded TLS key (default: %s)", defaultTlsKey))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	startServer(ctx, httpPort, false, handler(), stop)
	if tlsEnabled {
		startServer(ctx, httpsPort, true, handler(), stop)
	}

	wg.Wait()

	os.Exit(0)
}

func startServer(ctx context.Context, port int, tls bool, handler *http.ServeMux, stop chan os.Signal) {
	server := &http.Server{
		Addr: fmt.Sprintf("127.0.0.1:%d", port),
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
		Handler: handler,
	}

	logger.Info(fmt.Sprintf("%s listening", serviceFriendlyName), "port", port, "tls", strconv.FormatBool(tls))
	wg.Go(func() {
		if tls {
			if err := server.ListenAndServeTLS(tlsCertPath, tlsKeyPath); err != nil && err != http.ErrServerClosed {
				logger.Error(fmt.Sprintf("server.ListenAndServeTLS: %v", err), "port", port)
				os.Exit(1)
			}
		} else {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error(fmt.Sprintf("server.ListenAndServe: %v", err), "port", port)
				os.Exit(1)
			}
		}
	})

	go func() {
		<-stop
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Info(fmt.Sprintf("Forcibly shutting down server: %v", err))
		}
	}()
}

func handler() (ret *http.ServeMux) {
	ret = http.NewServeMux()

	ret.HandleFunc("/metadata/endpoints", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		apiVersion := q.Get("api-version")

		body, status, err := loadMetadataFromFile(apiVersion)
		if err != nil {
			w.WriteHeader(status)
			fmt.Fprintf(w, `{"error":"%s", "api-version": "%s"}`, err, apiVersion)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(body)
	})

	ret.HandleFunc("/metadata/identity/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		resource := q.Get("resource")

		body, status, err := acquireAccessToken(resource)
		if err != nil {
			w.WriteHeader(status)
			fmt.Fprintf(w, `{"error":"%s", "resource": "%s"}`, err, resource)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(body)
	})

	return
}

func acquireAccessToken(resource string) (body []byte, status int, err error) {
	azureTokenService := os.Getenv("AZURE_AUTHORITY_HOST")
	if azureTokenService == "" {
		return nil, http.StatusInternalServerError, fmt.Errorf("environment variable AZURE_AUTHORITY_HOST is not set")
	}

	clientId := os.Getenv("AZURE_CLIENT_ID")
	if clientId == "" {
		return nil, http.StatusInternalServerError, fmt.Errorf("environment variable AZURE_CLIENT_ID is not set")
	}

	idTokenFile := os.Getenv("AZURE_FEDERATED_TOKEN_FILE")
	if idTokenFile == "" {
		return nil, http.StatusInternalServerError, fmt.Errorf("environment variable AZURE_FEDERATED_TOKEN_FILE is not set")
	}

	tenantId := os.Getenv("AZURE_TENANT_ID")
	if tenantId == "" {
		return nil, http.StatusInternalServerError, fmt.Errorf("environment variable AZURE_TENANT_ID is not set")
	}

	idToken, err := os.ReadFile(idTokenFile)
	if err != nil {
		logger.Error(fmt.Sprintf("reading ID token: %v", err), "path", idTokenFile)
	}

	reqBody := url.Values{}
	reqBody.Set("audience", tenantId)
	reqBody.Set("client_assertion", string(idToken))
	reqBody.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	reqBody.Set("client_id", clientId)
	reqBody.Set("grant_type", "client_credentials")
	reqBody.Set("scope", fmt.Sprintf("%s/.default", resource))

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s/oauth2/v2.0/token", azureTokenService, tenantId), strings.NewReader(reqBody.Encode()))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	tokResp := tokenResponse{}
	if err = json.Unmarshal(body, &tokResp); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	tokResp.ClientID = clientId
	tokResp.Resource = resource
	tokResp.ExpiresOn = time.Now().Unix() + tokResp.ExpiresIn

	out, err := json.Marshal(tokResp)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return out, http.StatusOK, nil
}

func loadMetadataFromFile(apiVersion string) ([]byte, int, error) {
	filename := ""

	switch apiVersion {
	case "1.0", "2015-01-01":
		filename = "metadata20150101.json"
	case "2018-01-01":
		filename = "metadata20180101.json"
	case "2019-05-01", "2020-06-01":
		filename = "metadata20190501.json"
	case "2022-09-01":
		filename = "metadata20220901.json"
	default:
		return nil, http.StatusBadRequest, fmt.Errorf("unrecognized api-version")
	}

	metadata, err := os.ReadFile(path.Join("metadata", filename))
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("reading metadata from file: %v", err)
	}

	return metadata, http.StatusOK, nil
}
