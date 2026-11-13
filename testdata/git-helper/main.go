// This disposable Git shim lets subprocess tests cancel a known Git operation.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	if strings.Contains(strings.Join(os.Args[1:], " "), os.Getenv("SSB_GIT_BLOCK_MATCH")) {
		if err := os.WriteFile(os.Getenv("SSB_GIT_READY"), []byte("ready\n"), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		release := os.Getenv("SSB_GIT_RELEASE")
		if release == "" {
			time.Sleep(time.Minute)
			os.Exit(1)
		}
		deadline := time.Now().Add(30 * time.Second)
		for {
			if _, err := os.Stat(release); err == nil {
				break
			}
			if time.Now().After(deadline) {
				os.Exit(1)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	command := exec.Command(os.Getenv("SSB_REAL_GIT"), os.Args[1:]...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		if failure, ok := err.(*exec.ExitError); ok {
			os.Exit(failure.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
