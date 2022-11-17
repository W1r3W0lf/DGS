package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

type Repository struct {
	Name       string   // The name of the repository
	ActiveRepo string   // The name of the active repo
	RepoStore  string   // The location where all of the diffrent versions are stored
	Self       string   // The name of this node
	Peers      []Node   // All connected Peers
	AllPeers   []string // All connected and disconnected Peers
	//Initilised bool     // Has the repository been set up yet
}

func (repo *Repository) Initilised() bool {
	if repo.Name == "" {
		fmt.Fprintln(os.Stderr, "Repo has no name")
		return false
	}
	if repo.ActiveRepo == "" {
		fmt.Fprintln(os.Stderr, "Repo has no activeRepo")
		return false
	}
	if repo.RepoStore == "" {
		fmt.Fprintln(os.Stderr, "Repo has no Store")
		return false
	}
	if repo.Self == "" {
		fmt.Fprintln(os.Stderr, "Repo dosen't have self set")
		return false
	}

	return true
}

func (repo *Repository) SetRepoSymLink(peer string) {

	target, err := filepath.Abs(repo.RepoStore + repo.ActiveRepo)
	handleError(err, "Error getting an absolute path")

	destination, err := filepath.Abs(repo.RepoStore[:len(repo.RepoStore)-4])
	handleError(err, "Error getting an absolute path")

	err = os.Symlink(target, destination)
	handleError(err, "Error making symlink to repo")
}

func newRepository(path string, config UserConfig) Repository {

	var repo Repository

	repo.Name = filepath.Base(path)
	repo.Self = config.Name
	repo.ActiveRepo = repo.Self
	repo.RepoStore = config.RepoPath + repo.Name + "-vs/"
	repo.Peers = make([]Node, 0)
	repo.AllPeers = make([]string, 0)

	repo.Self = config.Name

	git.PlainClone(repo.RepoStore+config.Name, true, &git.CloneOptions{URL: path})

	repo.SetRepoSymLink(repo.Self)

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
	_, err = fmt.Fprintf(conn, "clone ")
	handleError(err, "Failed to send clone command")

	fmt.Println("Getting Server's name")
	// Get the Repository name, and the server's peer name
	_, err = fmt.Fscanf(conn, "%s", &node.Name)
	handleError(err, "Failed to get server's name")

	fmt.Println("Getting Repo's name")

	_, err = fmt.Fscanf(conn, "%s", &repo.Name)
	handleError(err, "Failed to get repo's name")

	// make the ./repos/NAME-vs/ direcotry
	repo.RepoStore = config.RepoPath + repo.Name + "-vs/"
	err = os.Mkdir(config.RepoPath+repo.Name+"-vs/", os.FileMode(0777))
	handleError(err, "Error Creating repo folder")

	fmt.Println("Getting repo")
	getRepo(repo.RepoStore+node.Name+".tar.gz", conn, reader)

	// Extract compressed Repository
	fmt.Println("Uncompressing file into ", repo.RepoStore)
	err = uncompressRepo(repo.RepoStore+node.Name, repo.RepoStore)
	handleError(err, "Error Extracting Repository")

	// Send my name to the server
	fmt.Fprintf(conn, config.Name+" ")

	// Add server to known peers
	repo.AllPeers = append(repo.AllPeers, node.Name)

	repo.SetRepoSymLink(repo.Self)

	// Finish setting up the repo
	repo.Self = config.Name
	repo.ActiveRepo = repo.Self

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
				for _, peer := range repo.Peers {
					if peer.Name == command[1] {
						pullRequest(repo, peer)
					}

				}
			case "accept":
				if len(command) == 2 {
					fmt.Println("Starting Server")
					repo.Peers = append(repo.Peers, newServerNode(command[1], repo))
					go repo.Peers[len(repo.Peers)-1].NodeDaemon()
				} else {
					fmt.Println("Incorrect number of arguments")
				}
			case "connect":
				if len(command) == 2 {
					fmt.Println("Connecting to Server")
					repo.Peers = append(repo.Peers, newClientNode(command[1], repo))
					go repo.Peers[len(repo.Peers)-1].NodeDaemon()
				} else {
					fmt.Println("Incorrect number of arguments")
				}
			case "terminate":
				// Kill all daemons
				for _, peer := range repo.Peers {
					peer.DaemonCMD <- "kill"
				}

			default:
				fmt.Println("Unknown command")
			}

		// If there is nothing to do, don't block
		default:
		}

		// Execute peer commands
		for _, peer := range repo.Peers {
			select {
			case command := <-peer.ReadChannel:
				switch command {
				case "pull":
					fmt.Println("Processing Pull Request")
					pullAccept(repo, peer)
				default:
				}
			default:
			}
		}
	}

}

func pullRequest(repo *Repository, peer Node) {
	peer.DaemonCMD <- "pause"
	getRepo(repo.RepoStore+peer.Name+".tar.gz", peer.Conn, peer.Reader)
	peer.DaemonCMD <- "resume"
}

func pullAccept(repo *Repository, peer Node) {
	sendRepo(repo.RepoStore+repo.Self+".tar.gz", peer.Conn)
}
