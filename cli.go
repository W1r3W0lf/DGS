package main

import (
	"bufio"
	"strings"
)

func getCommand(reader *bufio.Reader) ([]string, string) {

	rawCommand, err := reader.ReadString('\n')
	handleError(err, "Error Reading command")

	rawCommand = strings.TrimSuffix(rawCommand, "\n")

	// Parse command
	command := strings.Split(rawCommand, " ")

	return command, rawCommand
}
