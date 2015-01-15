package utils

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//
// HandleInterruption makes handling Ctrl+C (Interrup & SIGTERM sinals) in a separate go-routine.
// When signal received exists with status code 0 with a slight delay.
// Returns a created signal (if you need to force the exit in a proper way)
//
func HandleInterruption() chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		for _ = range ch {
			// Give 0MQ time to deliver before stopping...
			time.Sleep(1e9)
			log.Println("Stopped")
			os.Exit(0)
		}
	}()
	return ch
}
