package main

import (
	"fmt"
	"os"
)

// Motivation of this repository is to have TX Pool filled with insane numbers in geth.
// For now it will be just only a spike that makes the work, if possible it will be refactored and polished.
// It should be designed to work especially in docker and kubernetes environment, but tests at least in unit/component
// level should be runnable without containerisation.

var (
	IPC_ENDPOINT = "./geth.ipc"
)

func init() {
	ipcEndpoint := os.Getenv("IPC_ENDPOINT")

	if "" != ipcEndpoint {
		IPC_ENDPOINT = ipcEndpoint
	}
}

func main() {
	fmt.Printf("\n Running chaindriller on IPC: %s", IPC_ENDPOINT)
}
