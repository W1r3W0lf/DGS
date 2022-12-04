package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/libp2p/go-libp2p-core/host"
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

func (repo *Repository) newRepository(path string, uConfig UserConfig, host host.Host) {

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

	setStreamHandler(repo, host, &uConfig)
}

func (repo *Repository) openRepository(name string, config *UserConfig, host host.Host, ctx context.Context) error {

	for _, rp := range config.Repos {
		if rp.Name == name {
			*repo = rp

			setStreamHandler(repo, host, config)

			initMDNS(host, ctx, repo, config)

			return nil
		}
	}

	return errors.New("No such repository")
}

func P2PCloneRepository(address string, config *UserConfig, host host.Host, ctx context.Context) Repository {
	var repo Repository

	repo.Name = address

	setStreamHandler(&repo, host, config)

	initMDNS(host, ctx, &repo, config)

	return repo
}

func cloneRepository(address string, config UserConfig, host host.Host, ctx context.Context) Repository {

	var repo Repository
	var err error

	repo.Self = config.Name
	repo.ActiveRepo = repo.Self

	// Make a TCP connection to the server
	node := connectToPeer(&repo, host, address, true)

	fmt.Println("Sending Clone command")
	// Send the clone command
	node.Write <- "clone"

	fmt.Println("Getting Server's name")
	// Get the Repository name, and the server's peer name
	node.Name = <-node.Read
	fmt.Println(node.Name)

	fmt.Println("Getting Repo's name")

	repo.Name = <-node.Read
	fmt.Println(repo.Name)

	// make the ./repos/NAME-vs/ direcotry
	repo.RepoStore = config.RepoPath + repo.Name + "-vs/"
	err = os.Mkdir(config.RepoPath+repo.Name+"-vs/", os.FileMode(0777))
	handleError(err, "Error Creating repo folder")

	fmt.Println("Getting repo")
	node.GetRepo(repo.RepoStore + node.Name + ".tar.gz")

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
	node.Write <- config.Name

	// Add server to known peers
	repo.AllPeers = append(repo.AllPeers, node.Name)

	repo.Peers = append(repo.Peers, node)

	repo.SetRepoSymLink(repo.Self)

	return repo
}

func (repo *Repository) Run(command []string, host host.Host) {

	// Execute user commands
	switch command[0] {
	case "pull":
		if len(command) == 2 {
			fmt.Println("Pulling from " + command[1])
			for _, peer := range repo.Peers {
				if peer.Name == command[1] {
					pullRequest(repo, &peer)
				}
			}
		} else {
			fmt.Println("Incorrect number of arguments")
		}
	case "accept":
		fmt.Println("ACCEPT HAS BEEN DEPRICATED")
		/*
			if len(command) == 2 {
				fmt.Println("Starting Server")
				newPeer := newNode(command[1], repo, true)
				repo.Peers = append(repo.Peers, newPeer)
			} else {
				fmt.Println("Incorrect number of arguments")
			}
		*/
	case "connect":
		if len(command) == 2 {
			fmt.Println("Connecting to Server")
			//newPeer := newNode(command[1], repo, false)
			newPeer := connectToPeer(repo, host, command[1], false)
			repo.Peers = append(repo.Peers, newPeer)
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
		fmt.Printf("Connected to %d peers\n", len(repo.Peers))
		fmt.Println("\nAll:")
		for _, peer := range repo.AllPeers {
			fmt.Println(peer)
		}

	default:
		fmt.Println("Unknown command")
	}

}

// TODO Pulling should come from the Owners repository
// TODO Create an Onwer repository and delete the .tar and .tar.gz files

func pullRequest(repo *Repository, peer *Node) {
	peer.Write <- "pull"
	fmt.Println("Sent pull request")
	peer.GetRepo(repo.RepoStore + peer.Name + ".tar.gz")
}

func pullAccept(repo *Repository, peer *Node) {
	fmt.Println("Accepting pull request")
	peer.SendRepo(repo.RepoStore+repo.Self+".tar.gz", repo)
}
