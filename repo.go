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
	"github.com/go-git/go-git/v5/config"
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

func newRepository(path string, uConfig UserConfig) Repository {

	var repo Repository

	repo.Name = filepath.Base(path)
	repo.Self = uConfig.Name
	repo.ActiveRepo = repo.Self
	repo.RepoStore = uConfig.RepoPath + repo.Name + "-vs/"
	repo.Peers = make([]Node, 0)
	repo.AllPeers = make([]string, 0)

	repo.Self = uConfig.Name

	git.PlainClone(repo.RepoStore+uConfig.Name, true, &git.CloneOptions{URL: path})

	repo.SetRepoSymLink(repo.Self)

	destination, err := filepath.Abs(repo.RepoStore[:len(repo.RepoStore)-4])
	handleError(err, "Error getting an absolute path")

	r, err := git.PlainOpen(path)
	handleError(err, "Error opening original repository")
	r.CreateRemote(&config.RemoteConfig{Name: "DGS", URLs: []string{destination}})

	handleError(err, "Error getting the absolute path of the symlink to the repository")

	fmt.Println("DGS has been added as remote DGS in your repository")

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
	var err error

	repo.Self = config.Name
	repo.ActiveRepo = repo.Self

	node.Address = address
	// Make a TCP connection to the server
	node.Conn, err = net.Dial("tcp", address)
	handleError(err, "Failed to connect")
	node.Reader = bufio.NewReader(node.Conn)
	node.Writer = bufio.NewWriter(node.Conn)

	fmt.Println("Sending Clone command")
	// Send the clone command
	_, err = fmt.Fprintf(node.Conn, "clone ")
	handleError(err, "Failed to send clone command")

	fmt.Println("Getting Server's name")
	// Get the Repository name, and the server's peer name
	_, err = fmt.Fscanf(node.Conn, "%s", &node.Name)
	handleError(err, "Failed to get server's name")

	fmt.Println("Getting Repo's name")

	_, err = fmt.Fscanf(node.Conn, "%s", &repo.Name)
	handleError(err, "Failed to get repo's name")

	// make the ./repos/NAME-vs/ direcotry
	repo.RepoStore = config.RepoPath + repo.Name + "-vs/"
	err = os.Mkdir(config.RepoPath+repo.Name+"-vs/", os.FileMode(0777))
	handleError(err, "Error Creating repo folder")

	fmt.Println("Getting repo")
	getRepo(repo.RepoStore+node.Name+".tar.gz", node.Conn)

	// Extract compressed Repository
	fmt.Println("Uncompressing file into ", repo.RepoStore)
	err = uncompressRepo(repo.RepoStore+node.Name, repo.RepoStore)
	handleError(err, "Error Extracting Repository")

	// Rename direcotry to make it mine
	err = os.Rename(repo.RepoStore+node.Name, repo.RepoStore+repo.Self)
	handleError(err, "Error renaming the repo into my directory")

	// Extract compressed Repository again to be the remote's repo
	fmt.Println("Uncompressing file into ", repo.RepoStore)
	err = uncompressRepo(repo.RepoStore+node.Name, repo.RepoStore)
	handleError(err, "Error Extracting Repository")

	// Send my name to the server
	fmt.Fprintf(node.Conn, config.Name+" ")

	// Add server to known peers
	repo.AllPeers = append(repo.AllPeers, node.Name)

	repo.Peers = append(repo.Peers, node)

	repo.SetRepoSymLink(repo.Self)

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
				if len(command) == 2 {
					fmt.Println("Pulling from " + command[1])
					for _, peer := range repo.Peers {
						if peer.Name == command[1] {
							pullRequest(repo, peer)
						}
					}
				} else {
					fmt.Println("Incorrect number of arguments")
				}
			case "accept":
				if len(command) == 2 {
					fmt.Println("Starting Server")
					repo.Peers = append(repo.Peers, newServerNode(command[1], repo))
					repo.Peers[len(repo.Peers)-1].Command = make(chan string, 0)
				} else {
					fmt.Println("Incorrect number of arguments")
				}
			case "connect":
				if len(command) == 2 {
					fmt.Println("Connecting to Server")
					repo.Peers = append(repo.Peers, newClientNode(command[1], repo))
					repo.Peers[len(repo.Peers)-1].Command = make(chan string, 0)
				} else {
					fmt.Println("Incorrect number of arguments")
				}
			case "terminate":
				// Kill all daemons
				return
			case "ping":
				fmt.Println("pong")

			case "peers":
				fmt.Println("Connected:")
				for _, peer := range repo.Peers {
					fmt.Println(peer.Name)
				}
				fmt.Println("\nAll:")
				for _, peer := range repo.AllPeers {
					fmt.Println(peer)
				}

			default:
				fmt.Println("Unknown command")
			}

		// If there is nothing to do, don't block
		default:
		}

		for n := range repo.Peers {

			if repo.Peers[n].Daemon == false {
				// Start new Daemon
				repo.Peers[n].Daemon = true
				go func(CMDPeer *Node) {
					fmt.Println("New Command Getter")
					var cmd string
					fmt.Fscanf(CMDPeer.Conn, "%s", &cmd)
					CMDPeer.Command <- cmd
					CMDPeer.Daemon = false
					return
				}(&repo.Peers[n])
			}

			select {
			case peerCommand := <-repo.Peers[n].Command:
				fmt.Println("recived command \"" + peerCommand + "\"")

				switch peerCommand {
				case "pull":
					fmt.Println("Pull accepted")
					pullAccept(repo, repo.Peers[n])
				}
			default:
			}
		}
	}
}

// TODO Pulling should come from the Owners repository
// TODO Create an Onwer repository and delete the .tar and .tar.gz files

func pullRequest(repo *Repository, peer Node) {
	_, err := fmt.Fprintf(peer.Conn, "pull ")
	handleError(err, "Failed to send pull command")
	fmt.Println("Sent pull request")
	getRepo(repo.RepoStore+peer.Name+".tar.gz", peer.Conn)
}

func pullAccept(repo *Repository, peer Node) {
	_, err := fmt.Fprintf(peer.Conn, "Garbage ")
	handleError(err, "Failed to send Garbage word")
	sendRepo(repo.RepoStore+repo.Self+".tar.gz", peer.Conn)
}
