package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
)

type Repository struct {
	Name          string   // The name of the repository
	ActiveVersion string   //The name of the user who's repository is being used
	LinkPath      string   // The path to where the User is linked to
	RepoStore     string   // The location where all of the diffrent versions are stored
	Self          string   // The name of this node
	Peers         []Node   // All connected Peers
	AllPeers      []string // All connected and disconnected Peers
	Initilised    bool     // Has the repository been set up yet
}

func newRepository(path string, config UserConfig) Repository {

	var repo Repository

	repo.Name = filepath.Base(path)
	repo.ActiveVersion = config.Name
	repo.LinkPath = config.RepoPath + filepath.Base(path)
	repo.RepoStore = config.RepoPath + filepath.Base(path) + "-vs/"
	repo.Peers = make([]Node, 0)
	repo.AllPeers = make([]string, 0)

	repo.Self = config.Name

	git.PlainClone(repo.RepoStore+repo.Name, true, &git.CloneOptions{URL: path})

	abs, err := filepath.Abs(repo.RepoStore + repo.Name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error abs path", err.Error())
		panic(err)
	}

	err = os.Symlink(abs, repo.LinkPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error makeing symlink to repo", err.Error())
		panic(err)
	}

	repo.Initilised = true

	return repo
}

func openRepository(name string, config UserConfig) (Repository, error) {
	var repo Repository

	for _, rp := range config.Repos {
		if rp.Name == name {
			return rp, nil
		}
	}

	return repo, errors.New("No such repository")
}

func cloneRepository(address string, config UserConfig) Repository {

	var repo Repository
	var node Node

	node.Address = address

	// Make a TCP connection to the server
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error connecting ", err.Error())
		panic(err)
	}
	reader := bufio.NewReader(conn)

	// Send the clone command
	fmt.Fprintf(conn, "clone\n")

	// Get the Repository name, and the server's peer name
	node.Name, err = reader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error Getting Peer's name", err.Error())
		panic(err)
	}

	repo.Name, err = reader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error Getting Repository's name", err.Error())
		panic(err)
	}

	// Get the number of bytes that need to be accepted
	repoSizeString, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error Getting Peer's name", err.Error())
		panic(err)
	}
	repoSize, _ := strconv.Atoi(repoSizeString)
	buffer := make([]byte, repoSize)

	// Download the repository to ./repos/NAME-vs/USER.tar.gz
	repo.RepoStore = config.RepoPath + repo.Name + "-vs/"

	n, err := io.ReadFull(reader, buffer)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error Downloading Repository", err.Error())
		panic(err)
	}

	if n != repoSize {
		fmt.Println("Didn't recive enough bytes")
	}

	// TODO This dosen't match the Server
	ioutil.WriteFile(repo.RepoStore+node.Name+".tar.gz", buffer, 0644)

	// Extract compressed Repository

	// Open the Repository

	// Send my name to the server

	repo.Initilised = true

	return repo
}

func (repo *Repository) Run(commandChannel chan string) {
	fmt.Println("Strting", repo.Name)

	var cmd string

	for {

		// Execute user commands
		select {
		case cmd = <-commandChannel:
			command := strings.Split(cmd, " ")

			switch command[0] {
			case "pull":
				pullRequest(repo, command[1])
			case "accept":
				if len(command) == 2 {
					fmt.Println("Starting Server")
					repo.Peers = append(repo.Peers, newServerNode(command[1], repo))
				} else {
					fmt.Println("Incorrect number of arguments")
				}
			case "connect":
				if len(command) == 2 {
					fmt.Println("Connecting to Server")
					repo.Peers = append(repo.Peers, newClientNode(command[1]))
				} else {
					fmt.Println("Incorrect number of arguments")
				}
			case "terminate":

			default:
				fmt.Println("Unknown command")
			}

		// If there is nothing to do, don't block
		default:
		}

		// Execute peer commands
		for _, peer := range repo.Peers {
			rawMessage, _ := peer.Reader.ReadString('\n')

			message := strings.Split(rawMessage, "\n")

			switch message[0] {
			case "pull":
				pullAccept(repo, peer)
			default:
			}

		}
	}

}

func cloneAccept(repo *Repository) {

}

func pullRequest(repo *Repository, peer string) {

}

func pullAccept(repo *Repository, peer Node) {

}
