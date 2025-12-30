package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/manicminer/workload-identity-adal-bridge/identityclient"
	"github.com/manicminer/workload-identity-adal-bridge/internal/logger"
)

var wg sync.WaitGroup

type Server struct {
	TLSCertPath string
	TLSKeyPath  string

	ServiceName         string
	ServiceFriendlyName string

	Addr      *string
	HTTPPort  *int
	HTTPSPort *int
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	ClientID     string `json:"client_id"`
	Resource     string `json:"resource"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	ExpiresOn    int64  `json:"expires_on"`
	ExtExpiresIn int64  `json:"ext_expires_in"`
}

func StartServer(ctx context.Context, server Server) error {
	if server.HTTPPort == nil && server.HTTPSPort == nil {
		return fmt.Errorf("neither http port or https port was specified")
	}

	if server.HTTPPort != nil {
		server.run(ctx, *server.HTTPPort, false)
	}
	if server.HTTPSPort != nil {
		server.run(ctx, *server.HTTPSPort, true)
	}

	wg.Wait()

	return nil
}

func (s *Server) run(ctx context.Context, port int, tls bool) {
	if s.Addr == nil {
		addr := "127.0.0.1"
		s.Addr = &addr
	}
	server := &http.Server{
		Addr: fmt.Sprintf("%s:%d", *s.Addr, port),
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
		Handler: s.handler(),
	}

	logger.Info(fmt.Sprintf("%s listening", s.ServiceFriendlyName), "port", port, "tls", strconv.FormatBool(tls))
	wg.Go(func() {
		if tls {
			if err := server.ListenAndServeTLS(s.TLSCertPath, s.TLSKeyPath); err != nil && err != http.ErrServerClosed {
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
		<-ctx.Value("stop").(chan os.Signal)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Warn(fmt.Sprintf("Forcibly shutting down server: %v", err))
		}
	}()
}

func (s *Server) handler() (ret *http.ServeMux) {
	ret = http.NewServeMux()

	ret.HandleFunc("/metadata/endpoints", func(w http.ResponseWriter, r *http.Request) {
		s.logRequest(r)
		q := r.URL.Query()
		apiVersion := q.Get("api-version")

		body, status, err := s.loadMetadataFromFile(apiVersion)
		if err != nil {
			w.WriteHeader(status)
			fmt.Fprintf(w, `{"error":"%s", "api-version": "%s"}`, err, apiVersion)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(body)
	})

	ret.HandleFunc("/metadata/identity/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		s.logRequest(r)
		q := r.URL.Query()
		clientId := q.Get("client_id")
		resource := q.Get("resource")
		scope := q.Get("scope")

		body, status, err := s.acquireAccessToken(r.Context(), resource, scope, clientId)
		if err != nil {
			w.WriteHeader(status)
			fmt.Fprintf(w, `{"error":"%s", "resource": "%s", "scope: "%s", "clientId": "%s"}`, err, resource, scope, clientId)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(body)
	})

	return
}

func (s *Server) acquireAccessToken(ctx context.Context, resource, scope, clientId string) (body []byte, status int, err error) {
	tokResp, err := identityclient.AccessToken(ctx, resource, scope, clientId)
	if err != nil {
		logger.Error(fmt.Sprintf("acquiring access token: %v", err))
		return nil, http.StatusInternalServerError, err
	}

	if err = json.Unmarshal(body, &tokResp); err != nil {
		logger.Error(fmt.Sprintf("unmarshalling response from token service: %v", err), "response", string(body))
		return nil, http.StatusInternalServerError, err
	}

	out, err := json.Marshal(tokResp)
	if err != nil {
		logger.Error(fmt.Sprintf("marshalling response to client: %v", err), "token_response", fmt.Sprintf("%#v", tokResp))
		return nil, http.StatusInternalServerError, err
	}

	return out, http.StatusOK, nil
}

func (s *Server) loadMetadataFromFile(apiVersion string) ([]byte, int, error) {
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
		logger.Error("bad request: unrecognised api-version")
		return nil, http.StatusBadRequest, fmt.Errorf("unrecognized api-version")
	}

	metadata, err := os.ReadFile(path.Join("metadata", filename))
	if err != nil {
		logger.Error(fmt.Sprintf("reading metadata from file: %v", err))
		return nil, http.StatusInternalServerError, fmt.Errorf("reading metadata from file: %v", err)
	}

	return metadata, http.StatusOK, nil
}

func (s *Server) logRequest(r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(body))
	logger.Info(fmt.Sprintf("received request: %s %s", r.Method, r.RequestURI), "content_type", r.Header.Get("Content-Type"), "body", string(body))
}
