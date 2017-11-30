package node

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	ps "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	swarm "gx/ipfs/QmU219N3jn7QadVCeBUqGnAkwoXoUomrCwDuVQVuL7PB5W/go-libp2p-swarm"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	crypto "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"

	b58 "gx/ipfs/QmT8rehPR3F6bmwL6zjUN8XpiDBFFpMP2myPdC6ApsWfJf/go-base58"
)

// helper method - create a lib-p2p host to listen on a port
func createLocalNode(port int, done chan bool) *Node {
	// Ignoring most errors for brevity
	// See echo example for more details and better implementation
	priv, pub, _ := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	pid, _ := peer.IDFromPublicKey(pub)
	listen, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
	peerStore := ps.NewPeerstore()
	peerStore.AddPrivKey(pid, priv)
	peerStore.AddPubKey(pid, pub)
	n, _ := swarm.NewNetwork(context.Background(), []ma.Multiaddr{listen}, pid, peerStore, nil)
	host := bhost.New(n)

	// log some info on public keys and ids

	// this is implemented as the protobuf serialized key data
	pubKeyBytes, _ := pub.Bytes()
	log.Printf("Public key bytes: %d", len(pubKeyBytes))

	pubKeyStr := b58.Encode(pubKeyBytes)
	log.Printf("Public key (base58): %s, length:%d", pubKeyStr, len(pubKeyStr))
	log.Printf("Node id bytes: %d", len(pid))
	nodeIdStr := peer.IDB58Encode(pid)
	log.Printf("Node id (base58): %s, length: %d", nodeIdStr, len(nodeIdStr))

	return NewNode(host, done)
}

func TestP2pProtocols() {

	// Choose random ports between 10000-10100
	rand.Seed(666)
	port1 := rand.Intn(100) + 10000
	port2 := port1 + 1

	done := make(chan bool, 1)

	// Make 2 nodes
	h1 := createLocalNode(port1, done)
	h2 := createLocalNode(port2, done)
	h1.Peerstore().AddAddrs(h2.ID(), h2.Addrs(), ps.PermanentAddrTTL)
	h2.Peerstore().AddAddrs(h1.ID(), h1.Addrs(), ps.PermanentAddrTTL)

	log.Printf("This is a conversation between %s and %s\n", h1.ID(), h2.ID())

	// send messages using the protocols
	h1.Ping(h2.Host)
	h2.Ping(h1.Host)
	h1.Echo(h2.Host)
	h2.Echo(h1.Host)

	// block until all responses have been processed
	for i := 0; i < 4; i++ {
		<-done
	}
}
