package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func getCommand(reader *bufio.Reader) ([]string, string) {

	rawCommand, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error Reading command ", err.Error())
		panic(err)
	}

	rawCommand = strings.TrimSuffix(rawCommand, "\n")

	// Parse command
	command := strings.Split(rawCommand, " ")

	return command, rawCommand
}
