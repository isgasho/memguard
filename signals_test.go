// +build !windows

package memguard

import (
	"net"
	"os"
	"testing"
)

func TestCatchSignal(t *testing.T) {
	// Start a listener object
	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		SafePanic(err)
	}
	defer listener.Close()

	// Spawn a handler to catch interrupts
	CatchSignal(NewHandler(func(signals ...os.Signal) interface{} {
		// Close the listener
		listener.Close()

		// Return the signals we caught.
		var caught []string
		for _, signal := range signals {
			caught = append(caught, signal.String())
		}
		return caught
	}, true))

	//time.Sleep(8 * time.Second)

	// Grab a handle on the running process
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Error(nil)
	}

	// Send it an interrupt signal
	if err := process.Signal(os.Interrupt); err != nil {
		t.Error(err)
	}
}
