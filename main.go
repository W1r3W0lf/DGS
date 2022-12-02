package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/ipfs/go-log"
)

var logger = log.Logger("DGS")

/*
func handleStreamOLD(stream network.Stream) {
	logger.Info("Got a new stream!")

	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go readData(rw)
	go writeData(rw)

	// 'stream' will stay open until you close it (or the other side closes it).
}
*/

func handleError(err error, message string) {
	if err != nil {
		fmt.Fprintln(os.Stderr, message)
		panic(err)
	}
}

func main() {
	log.SetAllLoggers(log.LevelWarn)
	//log.SetLogLevel("DGS", "info")
	log.SetLogLevel("DGS", "debug")

	inputReader := bufio.NewReader(os.Stdin)
	user := startUser(inputReader)
	fmt.Println("Welcome back", user.Name)

	fmt.Println("Starting P2P")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host := newP2PHost(user.Port, ctx)

	var repo Repository

	// If there is only one repo, then open it
	if len(user.Repos) == 1 {
		for _, rp := range user.Repos {
			fmt.Printf("Opening %s\n", rp.Name)
			repo, _ = openRepository(rp.Name, user, host)
		}
	}

	for {
		//Take user input
		fmt.Fprintf(os.Stdout, ">")

		command, _ := getCommand(inputReader)

		switch command[0] {
		case "new":
			if repo.Initilised() {
				fmt.Println("Repo alredy initilised")
			} else {
				repo = newRepository(command[1], user, host)

				user.Repos[repo.Name] = repo
				writeConfig(user)

			}
		case "open":
			if repo.Initilised() {
				fmt.Println("Repo alredy initilised")
			} else {
				repo = user.Repos[command[1]]
			}
		case "clone":
			if repo.Initilised() {
				fmt.Println("Repo alredy initilised")
			} else {
				repo = cloneRepository(command[1], user, host)

				user.Repos[repo.Name] = repo
				writeConfig(user)

			}
		case "close":
			if repo.Initilised() {
				//command = "terminate"
				// Save Repository to config
				user.Repos[repo.Name] = repo
				// Uninitilise repository
				writeConfig(user)
				repo.Name = ""
			} else {
				fmt.Println("No Repository to close")
			}

		case "exit":
			if repo.Initilised() {
				//command = "terminate"
				// Close Repository
				user.Repos[repo.Name] = repo
			}
			writeConfig(user)

			// Exit Program
			os.Exit(0)
		case "terminate":
			fmt.Println("Unknown command")
		case "help":
			fmt.Println("new PATH\nopen NAME\nclone ip:port\nclose\nexit\nconnect ip:port\naccept :port\nping")
		default:
			if repo.Initilised() {
				repo.Run(command, host)
			} else {
				fmt.Println("Unknown command\nRepo Not started")
			}
		}
	}
}

/*
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
*/
