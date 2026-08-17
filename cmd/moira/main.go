package main

import (
	"log/slog"
	"os"

	"github.com/harshalvk/moira/internal/version"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting moira", "version", version.Version)

	// Phase 0, Step 2 will replace this with the actual
	// watch -> decide -> bind scheduling loop.
	logger.Info("moira scaffold is alive — no scheduling logic yet")
}
