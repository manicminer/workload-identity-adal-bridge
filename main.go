package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/manicminer/workload-identity-adal-bridge/cmd"
	"github.com/manicminer/workload-identity-adal-bridge/internal/logger"
)

const (
	serviceName = "workload-identity-adal-bridge"
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
	logger.Log = hclog.New(&hclog.LoggerOptions{
		Name:   serviceName,
		Level:  hclog.LevelFromString(os.Getenv("LOG_LEVEL")),
		TimeFn: time.Now,
	})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "stop", stop)
	defer cancel()

	cmd.Execute(ctx)
	os.Exit(0)
}
