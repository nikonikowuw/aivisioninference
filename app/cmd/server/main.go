// Package main is the entry point for the niko-admin server.
//
//	@title           Niko Admin API
//	@version         1.0
//	@description     基于 Gin 的后台管理系统 API
//	@host            localhost:8080
//	@BasePath        /api/v1
//	@securityDefinitions.apikey BearerAuth
//	@in              header
//	@name            Authorization
package main

import (
	"log"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/buildinfo"
	"github.com/niko-admin/niko-admin/internal/server"
)

func main() {
	app, err := server.InitializeApp()
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	zap.L().Info("starting niko-admin",
		zap.String("version", buildinfo.Version),
		zap.String("build_time", buildinfo.BuildTime),
	)

	app.Run()
}
