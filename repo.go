package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
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

	git.PlainClone(repo.RepoStore+config.Name, true, &git.CloneOptions{URL: path})

	abs, err := filepath.Abs(repo.RepoStore + repo.Name)
	handleError(err, "Error getting an absolute path")

	err = os.Symlink(abs, repo.LinkPath)
	handleError(err, "Error making symlink to repo")

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
	handleError(err, "Failed to connect")
	reader := bufio.NewReader(conn)

	fmt.Println("Sending Clone command")
	// Send the clone command
	_, err = fmt.Fprintf(conn, "clone\n")
	handleError(err, "Failed to send clone command")

	fmt.Println("Getting Server's name")
	// Get the Repository name, and the server's peer name
	_, err = fmt.Fscanf(conn, "%s", node.Name)
	handleError(err, "Failed to get server's name")

	fmt.Println("Getting Repo's name")

	_, err = fmt.Fscanf(conn, "%s", repo.Name)
	handleError(err, "Failed to get repo's name")

	fmt.Println("Getting Repo's size")
	// Get the number of bytes that need to be accepted
	var repoSizeString string
	_, err = fmt.Fscanf(conn, "%s", repoSizeString)
	handleError(err, "Failed to get repo's size")

	repoSize, _ := strconv.Atoi(repoSizeString)
	buffer := make([]byte, repoSize)

	// Download the repository to ./repos/NAME-vs/USER.tar.gz
	repo.RepoStore = config.RepoPath + repo.Name + "-vs/"
	err = os.Mkdir(config.RepoPath+repo.Name+"-vs/", os.FileMode(0777))
	handleError(err, "Error Creating repo folder")

	fmt.Println("Reading Bytes into buffer")
	n, err := io.ReadFull(reader, buffer)
	handleError(err, "Error Downloading repo")

	fmt.Println("Finishded Reading Bytes into buffer")

	if n != repoSize {
		fmt.Println("Didn't recive enough bytes")
	}

	fmt.Println("Writting file into", repo.RepoStore+node.Name+".tar.gz")
	//ioutil.WriteFile(repo.RepoStore+node.Name+".tar.gz", buffer, 0644)
	f, err := os.Create(repo.RepoStore + node.Name + ".tar.gz")
	handleError(err, "Error Creating Repository File")

	defer f.Close()
	f.Write(buffer)

	// Extract compressed Repository
	fmt.Println("Uncompressing file into ", repo.RepoStore)
	err = uncompressRepo(repo.RepoStore+node.Name, repo.RepoStore)
	handleError(err, "Error Extracting Repository")

	// Send my name to the server
	fmt.Fprintf(conn, config.Name)

	// Add server to known peers
	repo.AllPeers = append(repo.AllPeers, node.Name)

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
					repo.Peers = append(repo.Peers, newClientNode(command[1], repo))
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

func pullRequest(repo *Repository, peer string) {

}

func pullAccept(repo *Repository, peer Node) {

}
