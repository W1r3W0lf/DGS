package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/network"
)

var logger = log.Logger("DGS")

func handleStream(stream network.Stream) {
	logger.Info("Got a new stream!")

	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go readData(rw)
	go writeData(rw)

	// 'stream' will stay open until you close it (or the other side closes it).
}

func handleError(err error, message string) {
	if err != nil {
		fmt.Fprintln(os.Stderr, message)
		panic(err)
	}
}

func main() {

	inputReader := bufio.NewReader(os.Stdin)
	user := startUser(inputReader)
	fmt.Println("Welcome back", user.Name)

	repoChan := make(chan string)

	var repo Repository
	repo.Initilised = false

	for {
		//Take user input
		fmt.Print(">")

		command, rawCommand := getCommand(inputReader)

		switch command[0] {
		case "new":
			if repo.Initilised {
				fmt.Println("Repo alredy initilised")
			} else {
				repo = newRepository(command[1], user)

				user.Repos[repo.Name] = repo
				writeConfig(user)

				go repo.Run(repoChan)
			}
		case "open":
			if repo.Initilised {
				fmt.Println("Repo alredy initilised")
			} else {
				repo = user.Repos[command[1]]
				go repo.Run(repoChan)
			}
		case "clone":
			if repo.Initilised {
				fmt.Println("Repo alredy initilised")
			} else {
				repo = cloneRepository(command[1], user)

				user.Repos[repo.Name] = repo
				writeConfig(user)

				go repo.Run(repoChan)
			}
		case "close":
			if repo.Initilised {
				fmt.Println("No Repository to close")
			} else {
				repoChan <- "terminate"
				// Save Repository to config
				user.Repos[repo.Name] = repo
				// Uninitilise repository
				writeConfig(user)
			}

		case "exit":
			if repo.Initilised {
				repoChan <- "terminate"
				// Close Repository
				user.Repos[repo.Name] = repo
			}
			writeConfig(user)

			// Exit Program
			os.Exit(0)
		case "terminate":
			fmt.Println("Unknown command")
		case "help":
			fmt.Println("new PATH\nopen NAME\nconnect ip:port\naccept :port")
		default:
			if repo.Initilised {
				repoChan <- rawCommand
			} else {
				fmt.Println("Unknown command\nRepo Not started")
			}
		}
	}
}

func p2pStart() {

	log.SetAllLoggers(log.LevelWarn)
	//log.SetLogLevel("DGS", "info")
	log.SetLogLevel("DGS", "debug")
	help := flag.Bool("h", false, "Display Help")
	config, err := ParseFlags()
	if err != nil {
		panic(err)
	}

	if *help {
		fmt.Println("This program demonstrates a simple p2p chat application using libp2p")
		fmt.Println()
		fmt.Println("Usage: Run './chat in two different terminals. Let them connect to the bootstrap nodes, announce themselves and connect to the peers")
		flag.PrintDefaults()
		return
	}

	host := newP2PNode(config)

	connectToNetwork(host, config)
}
