package main

import (
	"context"
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/application"
	"os"
)

func main() {
	app := application.New()
	ctx := context.Background()
	os.Exit(app.Run(ctx))
}
