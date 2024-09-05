package util

import (
	"log"
	"os/exec"
)

func ExecuteCommand(command string, args ...string) *exec.Cmd {
	log.Println("Executing:", command, args)
	return exec.Command(command, args...)
}
