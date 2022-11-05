package main

import "fmt"

func cli() {

	var command string
	for {
		fmt.Println("Hello")

		fmt.Print("=")
		fmt.Scanln(&command)

		switch command {
		case "info":
			fmt.Println("Peer Info")
		default:
			fmt.Printf("Unknown command %s\n", command)
		}
	}
}
