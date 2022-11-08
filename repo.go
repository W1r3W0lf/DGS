package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

type Repository struct {
	name        string // The name of the repository
	path        string // The path to where the User is linked to
	backingPath string // The location where all of the diffrent versions are stored
	initilised  bool   // Has the repository been set up yet
	peers       []Node // All connected Peers
	appPeers    []Node // All connected and disconnected Peers
}

func newRepository(path string) Repository {

	var repo Repository

	repo.name = filepath.Base(path)

	repo.backingPath = "./repos/" + filepath.Base(path) + "-vs/"

	repo.path = "./repos/" + filepath.Base(path)

	git.PlainClone(repo.path, true, &git.CloneOptions{URL: path})

	err := os.Symlink(repo.backingPath+repo.name, repo.path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error makeing symlink to repo", err.Error)
		panic(err)
	}

	repo.initilised = true

	return repo
}

func openRepository(path string) Repository {

	var repo Repository

	repo.name = filepath.Base(path)

	repo.path = "./repos/" + filepath.Base(path)

	repo.backingPath = "./repos/" + filepath.Base(path) + "-vs/"

	repo.initilised = true

	return repo
}

func cloneRepository(address string) Repository {

	var repo Repository
	var node Node

	node.address = address

	// Make a TCP connection to the server
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting", err.Error)
		panic(err)
	}
	reader := bufio.NewReader(conn)

	// Send the clone command

	// Get the Repository name, and the server's peer name
	node.name, err = reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Getting Peer's name", err.Error)
		panic(err)
	}

	repo.name, err = reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Getting Peer's name", err.Error)
		panic(err)
	}

	// Download the repository to ./repos/NAME
	repo.path = "./repos/" + repo.name

	// Get the number of bytes that need to be accepted

	// Open the Repository

	repo.initilised = true

	return repo
}

func (repo *Repository) Run(commandChannel chan string) {
	fmt.Println("Strting", repo.name)

	// At this point the only way of setting up a repo is by cloneing
	if !repo.initilised {
		cloneRepo(repo)
		repo.initilised = true
	}

	var cmd string
	// Execute user commands
	select {
	case cmd = <-commandChannel:
		command := strings.Split(cmd, " ")

		switch command[0] {
		case "pull":
			pullFromPeer(repo, command[1])
		case "accept":
			if len(command) == 3 {
				fmt.Println("Starting Server")
				repo.peers = append(repo.peers, newServerNode(command[1], command[2]))
			} else {
				fmt.Println("Incorrect number of arguments")
			}
		case "connect":
			if len(command) == 3 {
				fmt.Println("Connecting to Server")
				repo.peers = append(repo.peers, newClientNode(command[1], command[2]))
			} else {
				fmt.Println("Incorrect number of arguments")
			}
		default:
			fmt.Println("Unknown command", cmd)
		}

	// If there is nothing to do, don't block
	default:
	}

	// Execute peer commands
	for _, peer := range repo.peers {
		rawMessage, _ := peer.reader.ReadString('\n')

		message := strings.Split(rawMessage, "\n")

		switch message[0] {
		case "pull":
			pushToPeer(repo, peer)
		}

	}

}

func pullFromPeer(repo *Repository, peer string) {

}

func cloneRepo(repo *Repository) {

}

func pushToPeer(repo *Repository, peer Node) {

}
