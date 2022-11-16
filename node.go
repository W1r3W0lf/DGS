package main

import (
	"bufio"
	"context"
	"fmt"
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
	Name        string
	Address     string
	Conn        net.Conn
	Reader      *bufio.Reader
	Writer      *bufio.Writer
	ReadChannel chan string
	KillDaemon  chan string
}

func (node *Node) NodeDaemon() {
	for {
		select {
		case <-node.KillDaemon:
			node.Conn.Close()
			return
		default:
			command, _ := node.Reader.ReadString(' ')

			if command != "" {
				select {
				case node.ReadChannel <- command:
				default:
				}
			}
		}
	}
}

func newServerNode(address string, repo *Repository) Node {
	var node Node
	// Set the node's name
	node.Address = address

	listen, err := net.Listen("tcp", address)
	handleError(err, "Error listaning")

	fmt.Println("Waiting for a client")
	node.Conn, err = listen.Accept()
	handleError(err, "Error connecting to client")

	fmt.Println("Client Connected")
	node.Reader = bufio.NewReader(node.Conn)
	node.Writer = bufio.NewWriter(node.Conn)

	var mode string
	// Get command from client
	_, err = fmt.Fscanf(node.Conn, "%s", &mode)
	handleError(err, "Error getting command from client")

	switch mode {
	case "clone":
		fmt.Println("Sending name")
		// Send my name to peer
		_, err = fmt.Fprintf(node.Conn, repo.Self+" ")
		handleError(err, "Error sending name to client")

		fmt.Println("Sending repo name")
		// Send repository name
		_, err = fmt.Fprintf(node.Conn, repo.Name+" ")
		handleError(err, "Error sending repository name to client")

		fmt.Println("Compressing file")
		// Compress My repository
		err = compressRepo(repo.RepoStore+repo.Self, repo.RepoStore)
		handleError(err, "Error compressing repo")

		repoTarPath := repo.RepoStore + repo.Self + ".tar.gz"

		// Get the size of the compressed repository
		repoTar, err := os.Open(repoTarPath)
		handleError(err, "Error opening repo tar file")
		defer repoTar.Close()

		fmt.Println("Sending File Size")
		// Send the size of the repository
		fileInfo, err := repoTar.Stat()
		handleError(err, "Error getting tarfile size")

		fileSize := strconv.FormatInt(fileInfo.Size(), 10)

		_, err = fmt.Fprintf(node.Conn, fileSize+" ")
		handleError(err, "Error sending file size")
		fmt.Println(fileSize)

		// Send the compressed repository
		fmt.Println("Sending File")

		sendBuffer := make([]byte, fileInfo.Size())
		_, err = repoTar.Read(sendBuffer)
		handleError(err, "Error reading repo into buffer")

		_, err = node.Conn.Write(sendBuffer)
		handleError(err, "Error sending data to client")
		fmt.Println("Finished Sending File")

		// Get the client's name
		_, err = fmt.Fscanf(node.Conn, "%s", &node.Name)
		handleError(err, "Error getting client's name")

		// Add client to knwon peers
		repo.AllPeers = append(repo.AllPeers, node.Name)

	case "connect":
		// Send my name to peer
		fmt.Fprintf(node.Conn, repo.Self)

		// Get client's name
		fmt.Fscanf(node.Conn, "%s", node.Name)

		// maybe? I'm not shure about this yet
		// Search for the client's name

		// If not found throw an error
	}

	return node
}

func newClientNode(address string, repo *Repository) Node {
	var node Node

	node.Address = address

	var err error
	node.Conn, err = net.Dial("tcp", address)
	handleError(err, "Error listaning")

	node.Reader = bufio.NewReader(node.Conn)
	node.Writer = bufio.NewWriter(node.Conn)

	// Ask to node's name

	_, err = fmt.Fprintf(node.Conn, "connect")
	handleError(err, "Error sending connection command to server")

	// Get the server's name
	_, err = fmt.Fscanf(node.Reader, "%s", node.Name)
	handleError(err, "Error getting server's name")

	// Send my name
	_, err = fmt.Fprintf(node.Conn, repo.Self)
	handleError(err, "Error sending name")

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
	handleError(err, "Error initalizing resource manager")

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
	handleError(err, "Error finding peers")

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
		handleError(err, "Error reading from buffer")

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
		handleError(err, "Error reading from stdin")

		_, err = rw.WriteString(fmt.Sprintf("%s\n", sendData))
		handleError(err, "Error writting buffer")

		err = rw.Flush()
		handleError(err, "Error flushing buffer")
	}
}
