package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

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

func main() {

	/*
		log.SetDebugLogging()

		f, err := os.Create("profile")
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	*/

	//	git.PlainClone("/tmp/DGSgit", true, &git.CloneOptions{
	//		URL: "/tmp/test-git"})

	repoChan := make(chan string)

	running := true
	var rawCommand string

	var repo repository
	repoStarted := false
	for running {
		//Take user input
		fmt.Print(">")
		//fmt.Scanln(&rawCommand)
		inputReader := bufio.NewReader(os.Stdin)
		rawCommand, _ = inputReader.ReadString('\n')

		// Parse command
		command := strings.Split(rawCommand, " ")

		switch command[0] {
		case "new":
			if repoStarted {

			} else {
				repo = newRepository(command[1])
				go repo.Run(repoChan)
			}
		case "open":
			if repoStarted {

			} else {
				repo = openRepository(command[1])
				go repo.Run(repoChan)
			}
		default:
			repoChan <- rawCommand
		}

	}

	return

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

	host := newNode(config)

	connectToNetwork(host, config)

}
