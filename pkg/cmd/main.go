package main

import (
	"context"
	"github.com/Cool-Andrey/Calculating/pkg/internal/application"
	"os"
)

func main() {
	app := application.New()
	ctx := context.Background()
	app.Run(ctx)
	os.Exit(0)
}
