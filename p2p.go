package main

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	libp2ptls "github.com/libp2p/go-libp2p-tls"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/multiformats/go-multiaddr"
)

type P2PHost struct {
	DHT *dht.IpfsDHT
}

func newP2PHost(port int, ctx context.Context) host.Host {
	var err error

	prvKey, _ := loadKeys()

	connmgr, err := connmgr.NewConnManager(
		100,
		400,
		connmgr.WithGracePeriod(time.Minute),
	)
	handleError(err, "Error creating Connection Manager")

	//sourceMultiiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", repo.Port))

	host, err := libp2p.New(
		libp2p.Identity(prvKey),
		//libp2p.ListenAddrs(sourceMultiiAddr),
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port),
		),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.DefaultTransports,
		libp2p.ConnectionManager(connmgr),
		libp2p.NATPortMap(),
		//libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
		//	newP2PHost.DHT, err = dht.New(newP2PHost.CTX, h)
		//	return newP2PHost.DHT, err
		//}),
		libp2p.EnableAutoRelay(),
	)
	handleError(err, "Error creaging host")

	fmt.Println("PeerID:", host.ID())
	//fmt.Printf("/ip4/127.0.0.1/tcp/%v/p2p/%s\n", port, host.ID().Pretty())

	return host
}

// This function won't be called
func connectToPeer(repo *Repository, host host.Host, address string, clone bool) Node {
	var node Node

	for _, la := range host.Addrs() {
		fmt.Printf(" - %v\n", la)
	}
	fmt.Println()

	maddr, err := multiaddr.NewMultiaddr(address)
	handleError(err, "Error creating peer's multiaddress")

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	handleError(err, "Error getting peer's info from multiaddr")

	host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	stream, err := host.NewStream(context.Background(), info.ID, "/dgs/0.1.0")
	handleError(err, "Error makeing a stream to peer")

	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	node.Read = make(chan string)
	go readData(rw, &node, repo)
	node.Write = make(chan string)
	go writeData(rw, &node)

	if !clone {
		node.NewClientNode(repo)
	}

	return node
}

func hostPort(host host.Host) {
	var port string
	for _, la := range host.Network().ListenAddresses() {
		if p, err := la.ValueForProtocol(multiaddr.P_TCP); err == nil {
			port = p
			break
		}
	}

	if port == "" {
		fmt.Println("Unable to find local port")
	}
	fmt.Printf("%s\n", port)
}

func makeBob() Node {
	var bob Node
	bob.Name = "Bob"
	return bob
}

func setStreamHandler(repo *Repository, host host.Host, config *UserConfig) {

	var port string
	for _, la := range host.Network().ListenAddresses() {
		if p, err := la.ValueForProtocol(multiaddr.P_TCP); err == nil {
			port = p
			break
		}
	}

	if port == "" {
		fmt.Println("Unable to find local port")
	}
	fmt.Printf("%s\n", port)

	// TODO see if I can add the name of the repository to the stream hander string
	// That would be so great
	host.SetStreamHandler("/dgs/0.1.0",
		func(stream network.Stream) {
			logger.Info("Got a new peer!")

			var node Node

			rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

			node.Daemons = true

			node.Read = make(chan string)
			go readData(rw, &node, repo)
			node.Write = make(chan string)
			go writeData(rw, &node)

			fmt.Println("Stream handler making peeer")
			node.NewP2PNode(repo, config, false)

			fmt.Printf("/ip4/127.0.0.1/tcp/%v/p2p/%s \n", port, host.ID().Pretty())

			logger.Info("Appending peer to peer list")
			repo.Peers = append(repo.Peers, node)
		})

	fmt.Printf("%s\n", port)

	fmt.Println("Stream Handler Set")

}

func initMDNS(host host.Host, ctx context.Context, repo *Repository, config *UserConfig) {
	n := &newMDNSpeer{}
	n.PeerChan = make(chan peer.AddrInfo)
	ser := mdns.NewMdnsService(host, repo.Name, n)
	err := ser.Start()
	handleError(err, "Error creating MDNS service")

	go MDNSdaemon(host, repo, ctx, n.PeerChan, config)
}

func MDNSdaemon(host host.Host, repo *Repository, ctx context.Context, peerChan chan peer.AddrInfo, config *UserConfig) {
	for {
		select {
		case peer := <-peerChan:
			logger.Info("Got a new MDNS peer!")
			var node Node

			err := host.Connect(ctx, peer)
			handleError(err, "Error connecting to MDNDS peer")

			stream, err := host.NewStream(context.Background(), peer.ID, "/dgs/0.1.0")
			handleError(err, "Error makeing a stream to peer")

			rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

			node.Daemons = true

			node.Read = make(chan string)
			go readData(rw, &node, repo)
			node.Write = make(chan string)
			go writeData(rw, &node)

			fmt.Println("MDNS daemon making peeer")
			node.NewP2PNode(repo, config, false)

			//logger.Info("Appending peer to peer list")
			repo.Peers = append(repo.Peers, node)
		}
	}
}

type newMDNSpeer struct {
	PeerChan chan peer.AddrInfo
}

func (n *newMDNSpeer) HandlePeerFound(peerInfo peer.AddrInfo) {
	n.PeerChan <- peerInfo
}

func readData(rw *bufio.ReadWriter, node *Node, repo *Repository) {
	for {
		str, _ := rw.ReadString('\n')

		if str == "" {
			return
		}
		str = strings.TrimSpace(str)

		switch str {
		case "pull":
			go pullAccept(repo, node)

		default:
			node.Read <- str
		}

		/*
			if !node.Daemons {
				fmt.Println("Daemon Closed")
				return
			}
		*/

	}
}

func writeData(rw *bufio.ReadWriter, node *Node) {
	var message string
	for {
		message = <-node.Write
		rw.WriteString(message + "\n")
		rw.Flush()

		/*
			if !node.Daemons {
				fmt.Println("Daemon Closed")
				return
			}
		*/
	}
}
