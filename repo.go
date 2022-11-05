package main

import (
	"bufio"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
)

type repoPeer struct {
	name  string
	read  *bufio.Reader
	write *bufio.Writer
}

func newRepoPeer(name string, port int) repoPeer {

	var rp repoPeer

	rp.name = name

	if port == 0 {
		// Listen for connections

		// Print out the port we are listanign on

	} else {
		conn, err := net.Dial("tcp", "127.0.0.1:8585")
		if err != nil {
			fmt.Println("Error connecting", err.Error)
			panic(err)
		}
		rp.read = bufio.NewReader(conn)
		rp.write = bufio.NewWriter(conn)
	}

	return rp
}

type repository struct {
	name       string
	path       string
	initilised bool
	peers      []repoPeer
}

func newRepository(path string) repository {

	var newRepo repository

	newRepo.name = filepath.Base(path)

	newRepo.path = "./repos/" + filepath.Base(path)

	git.PlainClone(newRepo.path, true, &git.CloneOptions{
		URL: path})
	//err := filepath.Walk

	return newRepo
}

func openRepository(path string) repository {

	var openedRepo repository

	openedRepo.name = filepath.Base(path)

	openedRepo.path = "./repos/" + filepath.Base(path)

	return openedRepo
}

func (repo *repository) Run(commandChannel chan string) {
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
			repo.peers = append(repo.peers, newRepoPeer(command[1], 0))
		case "connect":
			peerPort, _ := strconv.Atoi(command[2])
			repo.peers = append(repo.peers, newRepoPeer(command[1], peerPort))
		default:
			fmt.Println("Unknown command", cmd)
		}

	// If there is nothing to do, don't block
	default:
	}

	// Execute peer commands
	for _, peer := range repo.peers {
		rawMessage, _ := peer.read.ReadString('\n')

		message := strings.Split(rawMessage, "\n")

		switch message[0] {
		case "pull":
			pushToPeer(repo, peer)
		}

	}

}

func pullFromPeer(repo *repository, peer string) {

}

func cloneRepo(repo *repository) {

}

func pushToPeer(repo *repository, peer repoPeer) {

}
