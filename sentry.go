package main

import (
	"log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

func init() {
	sentryDsn := os.Getenv("SENTRY_DSN")
	if sentryDsn == "" {
		log.Println("Warning: Sentry DSN not set. Sentry error tracking will be disabled")
		return
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              sentryDsn,
		Environment:      "production",
		Debug:            true,
		TracesSampleRate: 1.0,
	})

	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}

	// Flush buffered events before the program terminates
	defer sentry.Flush(2 * time.Second)
}
