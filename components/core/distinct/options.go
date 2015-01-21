package main

import (
	"fmt"
	"os"
)

type options struct {
	DefaultExpiration int    `json:"duration"`
	CleanupInterval   int    `json:"cleanup"`
	File              string `json:"file"`
}

func (o *options) IsPersistent() bool {
	return o.File != ""
}

func (o *options) Validate() error {
	if o.DefaultExpiration < 0 {
		o.DefaultExpiration = 0
	}
	if o.CleanupInterval < 0 {
		o.CleanupInterval = 0
	}
	if o.CleanupInterval < o.DefaultExpiration {
		o.CleanupInterval = o.DefaultExpiration + 10
	}
	if o.IsPersistent() {
		info, err := os.Stat(o.File)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if info.IsDir() {
			return fmt.Errorf("Received directory instead of a file: %s", o.File)
		}
	}
	return nil
}
