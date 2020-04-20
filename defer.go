package datadog

import (
	"io"
	"log"
)

// Close closes the given closer and logs an error if one occurred.
func Close(c io.Closer) {
	if err := c.Close(); err != nil {
		log.Println("failed to close:", err)
	}
}
