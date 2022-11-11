package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-peerstore/pstoremem"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
)

type NodeP2P struct {
	host.Host
	DHT *dht.IpfsDHT
}

type Node struct {
	Name    string
	Address string
	Reader  *bufio.Reader
	Writer  *bufio.Writer
}

func newServerNode(address string, repo *Repository) Node {
	var node Node
	// Set the node's name
	node.Address = address

	listen, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Fprint(os.Stderr, "ERORR listaning", err.Error)
		panic(err)
	}
	conn, err := listen.Accept()
	if err != nil {
		fmt.Fprint(os.Stderr, "ERORR connecting to clinet", err.Error)
		panic(err)
	}
	node.Reader = bufio.NewReader(conn)
	node.Writer = bufio.NewWriter(conn)

	mode, err := node.Reader.ReadString('\n')
	if err != nil {
		fmt.Fprint(os.Stderr, "ERORR getting data from clinet", err.Error)
		panic(err)
	}

	switch mode {
	case "clone":
		// Send my name to peer
		fmt.Fprintf(conn, repo.Self)

		// Send repository name
		fmt.Fprintf(conn, repo.Name)

		// Compress My repository
		repoTarPath := compressRepo(repo.Path)

		// Get the size of the compressed repository
		repoTar, err := os.Open(repoTarPath)
		if err != nil {
			fmt.Fprint(os.Stderr, "ERORR Opening repo tar", err.Error)
			panic(err)
		}

		// Send the size of the repository
		fileInfo, _ := repoTar.Stat()
		fileSize := strconv.FormatInt(fileInfo.Size(), 10)

		fmt.Fprintf(conn, fileSize)

		// Send the compressed repository
		sendBuffer := make([]byte, 1000)

		for {
			_, err = repoTar.Read(sendBuffer)
			if err != io.EOF {
				break
			}
			conn.Write(sendBuffer)
		}

		// Get the client's name
		fmt.Fscanf(conn, "%s", node.Name)

	case "connect":
		// Send my name to peer
		fmt.Fprintf(conn, repo.Self)

		// Get client's name
		fmt.Fscanf(conn, "%s", node.Name)

		// Search for the client's name

		// If not found throw an error
	}

	return node
}

func newClientNode(address string) Node {
	var node Node
	// Set the node's name
	node.Address = address

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Fprint(os.Stderr, "ERORR listaning", err.Error)
		panic(err)
	}
	node.Reader = bufio.NewReader(conn)
	node.Writer = bufio.NewWriter(conn)

	fmt.Fprintf(conn, "connect")

	// Get the server's name

	// Send my name
	var name string
	fmt.Println("What's your name?")
	fmt.Fscanln(os.Stdin, name)
	fmt.Fprintf(conn, name)

	return node
}

/*
func newLimiter() network.ResourceManagerState {

	// Start with the default scaling limits.
	scalingLimits := rcmgr.DefaultLimits

	// Add limits around included libp2p protocols
	libp2p.SetDefaultServiceLimits(&scalingLimits)

	// Turn the scaling limits into a static set of limits using `.AutoScale`. This
	// scales the limits proportional to your system memory.
	limits := scalingLimits.AutoScale()

	// The resource manager expects a limiter, se we create one from our limits.
	limiter := rcmgr.NewFixedLimiter(limits)

	//limiter := rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits)

	// Initialize the resource manager
	rm, err := rcmgr.NewResourceManager(limiter)
	if err != nil {
		panic(err)
	}

	return rm
}
*/

func newP2PNode(config Config) host.Host {

	prvKey, _ := loadKeys()

	//rm := newLimiter()

	ps, err := pstoremem.NewPeerstore()

	// libp2p.New constructs a new libp2p Host. Other options can be added
	// here.
	host, err := libp2p.New(
		libp2p.ListenAddrs([]multiaddr.Multiaddr(config.ListenAddresses)...),
		libp2p.Identity(prvKey),
		//libp2p.ResourceManager(rm),
		libp2p.Peerstore(ps),
		//		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error){

		//})
	)
	if err != nil {
		panic(err)
	}
	logger.Info("Host created. We are:", host.ID())
	logger.Info(host.Addrs())

	return host
}

func connectToNetwork(host host.Host, config Config) {

	// Set a function as stream handler. This function is called when a peer
	// initiates a connection and starts a stream with this peer.
	host.SetStreamHandler(protocol.ID(config.ProtocolID), handleStream)

	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping node of the DHT can go down without
	// inhibiting future peer discovery.
	ctx := context.Background()
	kademliaDHT, err := dht.New(ctx, host)
	if err != nil {
		panic(err)
	}

	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.
	logger.Debug("Bootstrapping the DHT")
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	// Let's connect to the bootstrap nodes first. They will tell us about the
	// other nodes in the network.
	var wg sync.WaitGroup
	for _, peerAddr := range config.BootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *peerinfo); err != nil {
				logger.Warning(err)
			} else {
				logger.Info("Connection established with bootstrap node:", *peerinfo)
			}
		}()
	}
	wg.Wait()

	// We use a rendezvous point "meet me here" to announce our location.
	// This is like telling your friends to meet you at the Eiffel Tower.
	logger.Info("Announcing ourselves...")
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, config.RendezvousString)
	logger.Debug("Successfully announced!")

	// Now, look for others who have announced
	// This is like your friend telling you the location to meet you.
	logger.Debug("Searching for other peers...")
	peerChan, err := routingDiscovery.FindPeers(ctx, config.RendezvousString)
	if err != nil {
		panic(err)
	}

	//peerList := make([]*bufio.ReadWriter, 0, 0)

	connectToPeers(host, ctx, peerChan, config)

}

func connectToPeers(host host.Host, ctx context.Context, peerChan <-chan peer.AddrInfo, config Config) {

	for {
		select {
		case peer := <-peerChan:
			if peer.ID == host.ID() {
				continue
			}
			logger.Debug("Found peer:", peer)

			logger.Debug("Connecting to:", peer)
			stream, err := host.NewStream(ctx, peer.ID, protocol.ID(config.ProtocolID))

			if err != nil {
				logger.Warning("Connection failed:", err)
				continue
			} else {
				rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
				//peerList = append(peerList, rw)

				go writeData(rw)
				go readData(rw)
			}
			//fmt.Println(len(peerList))

			logger.Info("Connected to:", peer)

		}
	}
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}

		if str == "" {
			return
		}
		if str != "\n" {
			// Green console colour: 	\x1b[32m
			// Reset console colour: 	\x1b[0m
			fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
		}

	}
}

func writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from stdin")
			panic(err)
		}

		_, err = rw.WriteString(fmt.Sprintf("%s\n", sendData))
		if err != nil {
			fmt.Println("Error writing to buffer")
			panic(err)
		}
		err = rw.Flush()
		if err != nil {
			fmt.Println("Error flushing buffer")
			panic(err)
		}
	}
}
