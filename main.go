package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/manicminer/workload-identity-adal-bridge/cmd"
	"github.com/manicminer/workload-identity-adal-bridge/internal/logger"
	"github.com/spf13/viper"
)

const (
	serviceName = "workload-identity-adal-bridge"
)

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

	viper.SetEnvPrefix("BRIDGE")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	cmd.Execute(ctx)
	os.Exit(0)
}
