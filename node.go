package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"

	"github.com/libp2p/go-libp2p-core/host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

type NodeP2P struct {
	host.Host
	DHT *dht.IpfsDHT
}

type Node struct {
	Name    string
	Address string
	Daemons bool
	Read    chan string
	Write   chan string
}

func (node *Node) SendRepo(repoTarPath string, repo *Repository) {

	if _, err := os.Stat(repoTarPath); err != nil {
		fmt.Println("Compressing file")
		err := compressRepo(repo.RepoStore+repo.Self, repo.RepoStore)
		handleError(err, "Error compressing repo")
	}

	// Get the size of the compressed repository
	repoTar, err := os.Open(repoTarPath)
	handleError(err, "Error opening repo tar file")
	defer repoTar.Close()

	// Send the size of the repository
	fileInfo, err := repoTar.Stat()
	handleError(err, "Error getting tarfile size")

	fileSize := strconv.FormatInt(fileInfo.Size(), 10)

	node.Write <- fileSize

	fmt.Println(fileSize)

	// Send the compressed repository
	sendBuffer := make([]byte, fileInfo.Size())
	_, err = repoTar.Read(sendBuffer)
	handleError(err, "Error reading repo into buffer")

	sendString := base64.StdEncoding.EncodeToString(sendBuffer)

	node.Write <- sendString

	//_, err = out.Write(sendBuffer)
	//handleError(err, "Error sending data to client")
	fmt.Println("Finished Sending File")
}

func (node *Node) GetRepo(repoPath string) {

	fmt.Println("Getting Repo's size")
	// Get the number of bytes that need to be accepted

	repoSizeString := <-node.Read

	repoSize, err := strconv.Atoi(repoSizeString)
	handleError(err, "error converting tar size to int")
	buffer := make([]byte, repoSize)

	fmt.Println("Reading Bytes into buffer")
	//n, err := in.Read(buffer)
	//handleError(err, "Error Downloading repo")

	fileBase64 := <-node.Read
	n, err := base64.StdEncoding.Decode(buffer, ([]byte(fileBase64)))
	fmt.Println("Finishded Reading Bytes into buffer")

	if n != repoSize {
		fmt.Println("Didn't recive enough bytes")
	}

	f, err := os.Create(repoPath)
	handleError(err, "Error Creating Repository File")

	defer f.Close()
	f.Write(buffer)

}

func (node *Node) NewP2PNode(repo *Repository, config *UserConfig, handler bool) {

	fmt.Println("Client Connected")

	var myRequest string

	if repo.Initilised() || handler {
		fmt.Println("Requested connect")
		myRequest = "connect"
	} else {
		fmt.Println("Requested clone")
		myRequest = "clone"
	}
	node.Write <- myRequest

	// Get command from peer
	peerRequest := <-node.Read

	fmt.Println("got:", peerRequest)

	if peerRequest == "clone" && myRequest == "clone" {
		// 11
		fmt.Println("Error, self and peer don't have repo")
		return
	} else if peerRequest == "clone" {
		// 10
		postClone(node, repo)
	} else if myRequest == "clone" {
		// 01
		requestClone(node, repo, config)
	} else {
		//00
		P2Pconnect(node, repo)
	}
}

func requestClone(node *Node, repo *Repository, config *UserConfig) {

	repo.Self = config.Name
	repo.ActiveRepo = repo.Self

	fmt.Println("Sending name")
	// Send my name to peer
	node.Write <- repo.Self

	// Get peer's name
	node.Name = <-node.Read

	fmt.Println("Getting Repo's name")

	repo.Name = <-node.Read
	fmt.Println(repo.Name)

	// make the ./repos/NAME-vs/ direcotry
	repo.RepoStore = config.RepoPath + repo.Name + "-vs/"
	err := os.Mkdir(config.RepoPath+repo.Name+"-vs/", os.FileMode(0777))
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

	// Add server to known peers
	repo.AllPeers = append(repo.AllPeers, node.Name)

	repo.Peers = append(repo.Peers, *node)

	repo.SetRepoSymLink(repo.Self)

}

func postClone(node *Node, repo *Repository) {
	fmt.Println("Sending name")
	// Send my name to peer
	node.Write <- repo.Self

	// Get peer's name
	node.Name = <-node.Read

	fmt.Println("Sending repo name")
	// Send repository name
	node.Write <- repo.Name

	// Compress My repository
	fmt.Println("Compressing file")
	err := compressRepo(repo.RepoStore+repo.Self, repo.RepoStore)
	handleError(err, "Error compressing repo")

	repoTarPath := repo.RepoStore + repo.Self + ".tar.gz"

	// Send repo
	node.SendRepo(repoTarPath, repo)

	// Add client to knwon peers
	repo.AllPeers = append(repo.AllPeers, node.Name)
}

func P2Pconnect(node *Node, repo *Repository) {
	// Send my name to peer
	fmt.Println("Sending name to peer")
	node.Write <- repo.Self

	// Get peer's name
	fmt.Println("Getting peer's name")
	node.Name = <-node.Read
}

func (node *Node) NewServerNode(repo *Repository) {

	fmt.Println("Client Connected")

	// Get command from client
	mode := <-node.Read

	switch mode {
	case "clone":
		fmt.Println("Sending name")
		// Send my name to peer
		node.Write <- repo.Self

		fmt.Println("Sending repo name")
		// Send repository name
		node.Write <- repo.Name

		// Compress My repository
		fmt.Println("Compressing file")
		err := compressRepo(repo.RepoStore+repo.Self, repo.RepoStore)
		handleError(err, "Error compressing repo")

		repoTarPath := repo.RepoStore + repo.Self + ".tar.gz"

		// Send repo
		node.SendRepo(repoTarPath, repo)

		// Get the client's name
		node.Name = <-node.Read

		// Add client to knwon peers
		repo.AllPeers = append(repo.AllPeers, node.Name)

	case "connect":
		// Send my name to peer
		fmt.Println("Sending name to peer")
		node.Write <- repo.Self

		// Get client's name
		fmt.Println("Getting peer's name")
		node.Name = <-node.Read
		fmt.Println(node.Name)

		// maybe? I'm not shure about this yet
		// Search for the client's name

		for _, peer := range repo.AllPeers {
			if node.Name == peer {
			}
		}

		repo.AllPeers = append(repo.AllPeers, node.Name)

		// If not found throw an error
	}

}

func (node *Node) NewClientNode(repo *Repository) {

	// Send connect command
	node.Write <- "connect"

	// Get the server's name
	node.Name = <-node.Read

	// Send my name
	node.Write <- repo.Self

	repo.AllPeers = append(repo.AllPeers, node.Name)

}
