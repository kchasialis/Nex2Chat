package unit

import (
	"fmt"
	"github.com/stretchr/testify/require"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/peer/impl"
	"go.dedis.ch/cs438/storage/file"
	"go.dedis.ch/cs438/types"
	"go.dedis.ch/cs438/utils"
	"testing"
	"time"
)

// Testing the creation of a Solo Node Chord Network
// If a node is alone, it's successor is itself and predecessor too
func Test_N2C_Chord_SoloNodeChord(t *testing.T) {

	resetStorage()

	store, err := file.NewPersistency(impl.StoragePath)
	if err != nil {
		myLogger().Error().Msgf("[Test_N2C_Chord_SoloNodeChord] NewPersistency storage has failed")
	}

	transp := channelFac()

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(3*time.Second), z.WithChord(true))
	defer node1.Stop()

	// first node to enter the network
	err = node1.Join(types.ChordNode{})
	require.NoError(t, err)

	// some delay
	time.Sleep(time.Millisecond)

	require.Equal(t, node1.MyPredecessor().IPAddr, node1.GetAddr())
	require.Equal(t, node1.MySuccessor().IPAddr, node1.GetAddr())

	arrayChord := make([][KeyLen]byte, 1)
	arrayChord[0] = node1.GetChordNode().ID

	ring, err := impl.DisplayRing(arrayChord)
	require.NoError(t, err)
	myLogger().Info().Msgf(ring)

	myLogger().Info().Msgf("[Test_N2C_Chord_SoloNodeChord] [%s] \n%s", node1.GetAddr(), node1.PredAndSuccToString())

}

// Testing the creation of a Two Node Chord Network
// If a node is alone, it's successor is itself and predecessor too
// Then adding a second node, the predecessor and successor must be the other node
func Test_N2C_Chord_TwoNodeChord(t *testing.T) {

	resetStorage()

	store, err := file.NewPersistency(impl.StoragePath)
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage has failed")
	}

	transp := channelFac()

	// Node 1
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	// Node 2
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node2.Stop()

	// print finger table after creating the node
	myLogger().Info().Msgf("[Test_N2C_Chord_TwoNodeChord] [%s] FingerTable after creating new node: %s", node1.GetAddr(), node1.FingerTableToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_TwoNodeChord] [%s] FingerTable after creating new node: %s", node2.GetAddr(), node2.FingerTableToString())

	// display the ring with nodes
	arrayChord := make([][KeyLen]byte, 2)
	arrayChord[0] = node1.GetChordNode().ID
	arrayChord[1] = node2.GetChordNode().ID
	ring, err := impl.DisplayRing(arrayChord)
	require.NoError(t, err)
	myLogger().Info().Msgf(ring)

	// first node to enter the network
	err = node1.Join(types.ChordNode{})
	require.NoError(t, err)

	// some delay
	time.Sleep(time.Millisecond)

	myLogger().Info().Msgf("[Test_N2C_Chord_TwoNodeChord] [%s] FingerTable after joining the network (first node): %s", node1.GetAddr(), node1.FingerTableToString())
	require.Equal(t, node1.MyPredecessor().IPAddr, node1.GetAddr())
	require.Equal(t, node1.MySuccessor().IPAddr, node1.GetAddr())

	// 2nd node to enter the network
	err = node2.Join(node1.GetChordNode())
	require.NoError(t, err)

	// some delay
	time.Sleep(500 * time.Millisecond)

	myLogger().Info().Msgf("[Test_N2C_Chord_TwoNodeChord] [%s] FingerTable after joining the network (2nd node): %s", node2.GetAddr(), node2.FingerTableToString())
	require.Equal(t, node2.MyPredecessor().IPAddr, node1.GetAddr())
	require.Equal(t, node2.MySuccessor().IPAddr, node1.GetAddr())
	require.Equal(t, node1.MyPredecessor().IPAddr, node2.GetAddr())
	require.Equal(t, node1.MySuccessor().IPAddr, node2.GetAddr())

	// Temporary print
	time.Sleep(3 * time.Second)
	myLogger().Info().Msgf("[Test_N2C_Chord_TwoNodeChord] [%s] \n %s", node1.GetAddr(), node1.FingerTableToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_TwoNodeChord] [%s] \n %s", node1.GetAddr(), node1.PredAndSuccToString())
	time.Sleep(1 * time.Second)
	myLogger().Info().Msgf("[Test_N2C_Chord_TwoNodeChord] [%s] \n %s", node2.GetAddr(), node2.FingerTableToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_TwoNodeChord] [%s] \n %s", node2.GetAddr(), node2.PredAndSuccToString())

	ring, err = impl.DisplayRing(arrayChord)
	require.NoError(t, err)
	myLogger().Info().Msgf(ring)

}

// =================================================
// Temporary Test which are not completed UNIT TEST
// =================================================

// Testing the creation of a post and get post functions, as well as equal function
func Test_N2C_Chord_InitFingerTable(t *testing.T) {

	resetStorage()

	store, err := file.NewPersistency(impl.StoragePath)
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage has failed")
	}

	transp := channelFac()

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(3*time.Second), z.WithChord(true))
	defer node1.Stop()

	chordNode1 := node1.GetChordNode()
	myLogger().Info().Msgf("[Test_N2C_Chord_InitFingerTable] \n\tchordNode1.ID: %d\n\tchordNode1.IPAddr: %s",
		chordNode1.ID, chordNode1.IPAddr)

	// first node to enter the network
	err = node1.Join(types.ChordNode{})
	require.NoError(t, err)

	// some delay
	time.Sleep(time.Millisecond)

	// create a new node
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(3*time.Second), z.WithChord(true))
	defer node2.Stop()

	chordNode2 := node2.GetChordNode()
	myLogger().Info().Msgf("[Test_N2C_Chord_InitFingerTable] \n\tchordNode2.ID: %d\n\tchordNode2.IPAddr: %s",
		chordNode2.ID, chordNode2.IPAddr)

	myLogger().Info().Msg("[Test_N2C_Chord_InitFingerTable] node2.Join(chordNode1) .....................")

	err = node2.Join(chordNode1)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)
	myLogger().Info().Msgf("[Test_N2C_Chord_InitFingerTable] [%s] \n %s", node1.GetAddr(), node1.FingerTableToString())
	time.Sleep(1 * time.Second)
	myLogger().Info().Msgf("[Test_N2C_Chord_InitFingerTable] [%s] \n %s", node2.GetAddr(), node2.FingerTableToString())

	time.Sleep(1 * time.Second)
	myLogger().Info().Msgf("[Test_N2C_Chord_InitFingerTable] [%s] Successor: %s Predecessor: %s",
		node2.GetAddr(), node2.MySuccessor().IPAddr, node2.MyPredecessor().IPAddr)
	myLogger().Info().Msgf("[Test_N2C_Chord_InitFingerTable] [%s] Successor: %s Predecessor: %s",
		node1.GetAddr(), node1.MySuccessor().IPAddr, node1.MyPredecessor().IPAddr)

}

// Testing the creation of a post and get post functions, as well as equal function
func Test_N2C_Chord_JoinNetworkChordTest(t *testing.T) {

	resetStorage()

	store, err := file.NewPersistency(impl.StoragePath)
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage has failed")
	}

	transp := channelFac()

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(3*time.Second), z.WithChord(true))
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(3*time.Second), z.WithChord(true))
	defer node1.Stop()
	defer node2.Stop()

	// first node to enter the network
	node1.Join(types.ChordNode{})
	// some delay
	time.Sleep(time.Millisecond)
	node2.Join(node1.GetChordNode())

	myLogger().Info().Msgf("\nTEST\n: %v", node1.GetOuts())
}

func Test_N2C_Chord_3Nodes(t *testing.T) {

	resetStorage()

	store, err := file.NewPersistency(impl.StoragePath)
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage has failed")
	}

	transp := channelFac()

	// Node 1
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	// Node 2
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node2.Stop()

	node3 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node3.Stop()

	// print finger table after creating the node
	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] [%s] FingerTable after creating new node: %s", node1.GetAddr(), node1.FingerTableToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] [%s] FingerTable after creating new node: %s", node2.GetAddr(), node2.FingerTableToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] [%s] FingerTable after creating new node: %s", node3.GetAddr(), node3.FingerTableToString())

	// display the ring with nodes
	arrayChord := make([][KeyLen]byte, 3)
	arrayChord[0] = node1.GetChordNode().ID
	arrayChord[1] = node2.GetChordNode().ID
	arrayChord[2] = node3.GetChordNode().ID
	ring, err := impl.DisplayRing(arrayChord)
	myLogger().Info().Msgf(ring)

	// node 1
	chordNode1 := node1.GetChordNode()
	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] \n\tchordNode1.ID: %d\n\tchordNode1.IPAddr: %s",
		chordNode1.ID, chordNode1.IPAddr)

	// first node to enter the network
	err = node1.Join(types.ChordNode{})
	require.NoError(t, err)

	// some delay
	//time.Sleep(time.Millisecond)

	// node 2
	chordNode2 := node2.GetChordNode()
	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] \n\tchordNode2.ID: %d\n\tchordNode2.IPAddr: %s",
		chordNode2.ID, chordNode2.IPAddr)

	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] node2.Join(chordNode1) .....................")
	err = node2.Join(chordNode1)
	require.NoError(t, err)

	//time.Sleep(5 * time.Second)
	// state after node 2 joined
	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] [%s] state after node 2 joined: \n %s", node1.GetAddr(), node1.FingerTableToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] [%s] state after node 2 joined: \n %s", node2.GetAddr(), node2.FingerTableToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] [%s] state after node 2 joined: \n %s", node3.GetAddr(), node3.FingerTableToString())
	ring, err = impl.DisplayRing(arrayChord)
	require.NoError(t, err)
	myLogger().Info().Msgf(ring)

	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] [%s] state after node 2 joined: \n%s", node1.GetAddr(), node1.PredAndSuccToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] [%s] state after node 2 joined: \n%s", node2.GetAddr(), node2.PredAndSuccToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] [%s] state after node 2 joined: \n%s", node3.GetAddr(), node3.PredAndSuccToString())

	// node 3
	chordNode3 := node3.GetChordNode()
	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] \n\tchordNode3.ID: %d\n\tchordNode3.IPAddr: %s",
		chordNode3.ID, chordNode3.IPAddr)

	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] node3.Join(chordNode1) .....................")
	err = node3.Join(chordNode1)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)

	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] [%s] \n %s", node1.GetAddr(), node1.FingerTableToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] [%s] \n %s", node2.GetAddr(), node2.FingerTableToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] [%s] \n %s", node3.GetAddr(), node3.FingerTableToString())

	ring, err = impl.DisplayRing(arrayChord)
	require.NoError(t, err)
	myLogger().Info().Msgf(ring)

	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] [%s] \n%s", node1.GetAddr(), node1.PredAndSuccToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] [%s] \n%s", node2.GetAddr(), node2.PredAndSuccToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_3Nodes] [%s] \n%s", node3.GetAddr(), node3.PredAndSuccToString())

}

func Test_N2C_Chord_4Nodes(t *testing.T) {
	resetStorage()

	store, err := file.NewPersistency(impl.StoragePath)
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage has failed")
	}

	transp := channelFac()

	// Node 1
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	// Node 2
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node2.Stop()

	node3 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node3.Stop()

	node4 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node4.Stop()

	// print finger table after creating the node
	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] [%s] FingerTable after creating new node: %s", node1.GetAddr(), node1.FingerTableToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] [%s] FingerTable after creating new node: %s", node2.GetAddr(), node2.FingerTableToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] [%s] FingerTable after creating new node: %s", node3.GetAddr(), node3.FingerTableToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] [%s] FingerTable after creating new node: %s", node4.GetAddr(), node4.FingerTableToString())

	// display the ring with nodes
	arrayChord := make([][KeyLen]byte, 4)
	arrayChord[0] = node1.GetChordNode().ID
	arrayChord[1] = node2.GetChordNode().ID
	arrayChord[2] = node3.GetChordNode().ID
	arrayChord[3] = node4.GetChordNode().ID
	ring, err := impl.DisplayRing(arrayChord)
	myLogger().Info().Msgf(ring)

	// node 1
	chordNode1 := node1.GetChordNode()
	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] \n\tchordNode1.ID: %d\n\tchordNode1.IPAddr: %s",
		chordNode1.ID, chordNode1.IPAddr)

	// first node to enter the network
	err = node1.Join(types.ChordNode{})
	require.NoError(t, err)

	// node 2
	chordNode2 := node2.GetChordNode()
	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] \n\tchordNode2.ID: %d\n\tchordNode2.IPAddr: %s",
		chordNode2.ID, chordNode2.IPAddr)

	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] node2.Join(chordNode1) .....................")
	err = node2.Join(chordNode1)
	require.NoError(t, err)

	// node 3
	chordNode3 := node3.GetChordNode()
	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] \n\tchordNode3.ID: %d\n\tchordNode3.IPAddr: %s",
		chordNode3.ID, chordNode3.IPAddr)

	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] node3.Join(chordNode1) .....................")
	err = node3.Join(chordNode1)
	require.NoError(t, err)

	// node 4
	chordNode4 := node4.GetChordNode()
	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] \n\tchordNode4.ID: %d\n\tchordNode4.IPAddr: %s",
		chordNode4.ID, chordNode4.IPAddr)

	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] node4.Join(chordNode1) .....................")
	err = node4.Join(chordNode1)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)

	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] [%s] \n %s", node1.GetAddr(), node1.FingerTableToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] [%s] \n %s", node2.GetAddr(), node2.FingerTableToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] [%s] \n %s", node3.GetAddr(), node3.FingerTableToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] [%s] \n %s", node4.GetAddr(), node4.FingerTableToString())

	ring, err = impl.DisplayRing(arrayChord)
	require.NoError(t, err)
	myLogger().Info().Msgf(ring)

	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] [%s] \n%s", node1.GetAddr(), node1.PredAndSuccToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] [%s] \n%s", node2.GetAddr(), node2.PredAndSuccToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] [%s] \n%s", node3.GetAddr(), node3.PredAndSuccToString())
	myLogger().Info().Msgf("[Test_N2C_Chord_4Nodes] [%s] \n%s", node4.GetAddr(), node4.PredAndSuccToString())

}

func Test_N2C_Chord_Multiple_Nodes(t *testing.T) {
	resetStorage()

	transp := channelFac()

	nbNodes := 4
	nodesArray, _, chordIDsArray, err := GenerateChordNodes(t, nbNodes, transp)

	for i := range nodesArray {
		myLogger().Info().Msgf("[Test_Multiple_Nodes] [%s] \n %s", nodesArray[i].GetAddr(), nodesArray[i].FingerTableToString())
	}

	ring, err := impl.DisplayRing(chordIDsArray)
	require.NoError(t, err)
	myLogger().Info().Msgf(ring)

	for i := range nodesArray {
		myLogger().Info().Msgf("[Test_Multiple_Nodes] [%s] \n%s", nodesArray[i].GetAddr(), nodesArray[i].PredAndSuccToString())
	}
}

// Generate a Chord network with multiple nodes, create a post, and store it on the network
func Test_N2C_Chord_StorePostToChord(t *testing.T) {
	resetStorage()

	transp := channelFac()

	nbNodes := 4
	nodesArray, _, chordIDsArray, err := GenerateChordNodes(t, nbNodes, transp)

	for i := range nodesArray {
		myLogger().Info().Msgf("[Test_N2C_Chord_StorePostToChord] [%s] \n %s", nodesArray[i].GetAddr(), nodesArray[i].FingerTableToString())
	}

	for i := range nodesArray {
		myLogger().Info().Msgf("[Test_N2C_Chord_StorePostToChord] [%s] \n%s", nodesArray[i].GetAddr(), nodesArray[i].PredAndSuccToString())
	}

	// Create and Store a post
	postContent1 := []byte("This is my first post")

	post := nodesArray[0].CreatePost(postContent1)
	err = nodesArray[0].StorePrivatePost(post)
	require.NoError(t, err)

	// convert postID str to bigIng
	postIDBytes, err := utils.DecodeChordIDFromStr(post.PostID)
	require.NoError(t, err)

	var labels []string

	// labels of nodes before posts
	for i := range chordIDsArray {
		labels = append(labels, fmt.Sprintf("n%d", i+1))
	}

	// add post IDs and labels
	chordIDsArray = append(chordIDsArray, postIDBytes)
	labels = append(labels, "p1")
	ring, err := impl.DisplayRing(chordIDsArray, labels...)
	myLogger().Info().Msgf(ring)

	myLogger().Info().Msgf("[Test_N2C_Chord_StorePostToChord] [%s] post1ByNode1: %v", nodesArray[0].GetAddr(), post)

}

// Generate a Chord network with multiple nodes, create a post, store it on the network, and retrieve it.
func Test_N2C_Chord_StoreAndRetrievePostFromChord(t *testing.T) {
	resetStorage()

	transp := channelFac()

	nbNodes := 3
	nodesArray, _, chordIDsArray, err := GenerateChordNodes(t, nbNodes, transp)

	for i := range nodesArray {
		myLogger().Info().Msgf("[Test_N2C_Chord_StoreAndRetrievePostFromChord] [%s] \n %s", nodesArray[i].GetAddr(), nodesArray[i].FingerTableToString())
	}

	ring, err := impl.DisplayRing(chordIDsArray)
	require.NoError(t, err)
	myLogger().Info().Msgf(ring)

	for i := range nodesArray {
		myLogger().Info().Msgf("[Test_N2C_Chord_StoreAndRetrievePostFromChord] [%s] \n%s", nodesArray[i].GetAddr(), nodesArray[i].PredAndSuccToString())
	}

	// Create and Store a post
	postContent1 := []byte("This is my first post")

	post := nodesArray[0].CreatePost(postContent1)
	err = nodesArray[0].StorePrivatePost(post)
	require.NoError(t, err)

	// convert postID str to bigIng
	postIDBytes, err := utils.DecodeChordIDFromStr(post.PostID)
	require.NoError(t, err)

	var labels []string

	// labels of nodes before posts
	for i := range chordIDsArray {
		labels = append(labels, fmt.Sprintf("n%d", i+1))
	}

	// add post IDs and labels
	chordIDsArray = append(chordIDsArray, postIDBytes)
	labels = append(labels, "p1")
	ring, err = impl.DisplayRing(chordIDsArray, labels...)
	require.NoError(t, err)
	myLogger().Info().Msgf(ring)

	myLogger().Info().Msgf("[Test_N2C_Chord_StoreAndRetrievePostFromChord] [%s] post1ByNode1: %v", nodesArray[0].GetAddr(), post)

	myLogger().Info().Msgf("[Test_N2C_Chord_StoreAndRetrievePostFromChord] ------------------------------ RETRIEVE POST" +
		"--------------------------")

	// retrieve post (possibly from another node) given postID
	retrievedPost, err := nodesArray[0].GetPost(post.PostID)
	require.NoError(t, err)
	myLogger().Info().Msgf("[Test_N2C_Chord_StoreAndRetrievePostFromChord] [%s] retrievedPost: %s", nodesArray[0].GetAddr(), retrievedPost)

	// retrieve post (possibly from another node) given postID
	retrievedPost2, err := nodesArray[1].GetPost(post.PostID)
	require.NoError(t, err)
	myLogger().Info().Msgf("[Test_N2C_Chord_StoreAndRetrievePostFromChord] [%s] retrievedPost: %s", nodesArray[1].GetAddr(), retrievedPost2)

	// Retrieved post must be equal to the original post
	require.True(t, retrievedPost.Equal(post))
}

// Generate a Chord network with multiple nodes, create a post, store it on the network, and retrieve it.
func Test_N2C_Chord_UpdateCatalog(t *testing.T) {
	resetStorage()

	transp := channelFac()

	nbNodes := 3
	nodesArray, _, chordIDsArray, err := GenerateChordNodes(t, nbNodes, transp)

	for i := range nodesArray {
		myLogger().Info().Msgf("[Test_N2C_Chord_UpdateCatalog] [%s] \n %s", nodesArray[i].GetAddr(), nodesArray[i].FingerTableToString())
	}

	ring, err := impl.DisplayRing(chordIDsArray)
	require.NoError(t, err)
	myLogger().Info().Msgf(ring)

	for i := range nodesArray {
		myLogger().Info().Msgf("[Test_N2C_Chord_UpdateCatalog] [%s] \n%s", nodesArray[i].GetAddr(), nodesArray[i].PredAndSuccToString())
	}

	myLogger().Info().Msgf("[Test_N2C_Chord_UpdateCatalog] [%s] PostCatalog before posting: %s", nodesArray[0].GetAddr(), nodesArray[0].PostCatalogToString())

	myLogger().Info().Msgf("[Test_N2C_Chord_UpdateCatalog] ---------------------------- CREATE POSTS ---------------------")

	// Create content from node 2
	n2c1 := []byte("This is node2's first post")
	n2c2 := []byte("This is node2's second post")
	n2c3 := []byte("This is node2's third post")

	n2p1 := nodesArray[1].CreatePost(n2c1)
	err = nodesArray[1].StorePrivatePost(n2p1)
	require.NoError(t, err)

	n2p2 := nodesArray[1].CreatePost(n2c2)
	err = nodesArray[1].StorePrivatePost(n2p2)
	require.NoError(t, err)

	n2p3 := nodesArray[1].CreatePost(n2c3)
	err = nodesArray[1].StorePrivatePost(n2p3)
	require.NoError(t, err)

	// Create content from node 3
	n3c1 := []byte("This is node3's first post")
	n3c2 := []byte("This is node3's second post")
	n3c3 := []byte("This is node3's third post")

	n3p1 := nodesArray[2].CreatePost(n3c1)
	err = nodesArray[2].StorePrivatePost(n3p1)
	require.NoError(t, err)

	n3p2 := nodesArray[2].CreatePost(n3c2)
	err = nodesArray[2].StorePrivatePost(n3p2)
	require.NoError(t, err)

	n3p3 := nodesArray[2].CreatePost(n3c3)
	err = nodesArray[2].StorePrivatePost(n3p3)
	require.NoError(t, err)

	var listOfPosts = []types.Post{n2p1, n2p2, n2p3, n3p1, n3p2, n3p3}
	for i := range listOfPosts {
		myLogger().Info().Msgf("[Test_N2C_Chord_UpdateCatalog] listOfPosts[%d]: %s", i, listOfPosts[i])
	}

	//convert postID str to []bytes
	n2p1IDB, err := utils.DecodeChordIDFromStr(n2p1.PostID)
	require.NoError(t, err)
	n2p2IDB, err := utils.DecodeChordIDFromStr(n2p2.PostID)
	require.NoError(t, err)
	n2p3IDB, err := utils.DecodeChordIDFromStr(n2p3.PostID)
	require.NoError(t, err)

	n3p1IDB, err := utils.DecodeChordIDFromStr(n3p1.PostID)
	require.NoError(t, err)
	n3p2IDB, err := utils.DecodeChordIDFromStr(n3p2.PostID)
	require.NoError(t, err)
	n3p3IDB, err := utils.DecodeChordIDFromStr(n3p3.PostID)
	require.NoError(t, err)

	var labels []string

	// labels of nodes before posts
	for i := range chordIDsArray {
		labels = append(labels, fmt.Sprintf("n%d", i+1))
	}

	postsIDB := [][KeyLen]byte{n2p1IDB, n2p2IDB, n2p3IDB, n3p1IDB, n3p2IDB, n3p3IDB}

	// add post IDs and labels
	chordIDsArray = append(chordIDsArray, postsIDB...)

	postsLabels := []string{"n2p1", "n2p2", "n2p3", "n3p1", "n3p2", "n3p3"}
	labels = append(labels, postsLabels...)
	ring, err = impl.DisplayRing(chordIDsArray, labels...)
	require.NoError(t, err)
	myLogger().Info().Msgf(ring)

	myLogger().Info().Msgf("[Test_N2C_Chord_UpdateCatalog] ---------------------------- CATALOG ---------------------")

	myLogger().Info().Msgf("[Test_N2C_Chord_UpdateCatalog] [%s] PostCatalog before update: %s",
		nodesArray[0].GetAddr(), nodesArray[0].PostCatalogToString())

	err = nodesArray[0].UpdatePostCatalog()
	require.NoError(t, err)
	myLogger().Info().Msgf("[Test_N2C_Chord_UpdateCatalog] [%s] PostCatalog after update: %s",
		nodesArray[0].GetAddr(), nodesArray[0].PostCatalogToString())

	myLogger().Info().Msgf("[Test_N2C_Chord_UpdateCatalog] ---------------- NODE 1 FOLLOW PEERS 2 AND 3 --------------")

	nodesArray[0].AddPeer(nodesArray[1].GetChordNode().IPAddr)
	nodesArray[0].AddPeer(nodesArray[2].GetChordNode().IPAddr)
	nodesArray[1].AddPeer(nodesArray[0].GetChordNode().IPAddr)
	nodesArray[2].AddPeer(nodesArray[0].GetChordNode().IPAddr)

	err = nodesArray[0].FollowRequest(nodesArray[1].GetAddr())
	require.NoError(t, err)
	err = nodesArray[0].FollowRequest(nodesArray[2].GetAddr())
	require.NoError(t, err)

	time.Sleep(1000 * time.Millisecond)

	require.Len(t, nodesArray[0].GetFollowing(), 2)
	require.Len(t, nodesArray[0].GetFollowers(), 0)

	require.Len(t, nodesArray[1].GetFollowing(), 0)
	require.Len(t, nodesArray[1].GetFollowers(), 1)

	require.Len(t, nodesArray[2].GetFollowing(), 0)
	require.Len(t, nodesArray[2].GetFollowers(), 1)

	myLogger().Info().Msgf("[Test_N2C_Chord_UpdateCatalog] ---------------------------- CATALOG ---------------------")
	myLogger().Info().Msgf("[Test_N2C_Chord_UpdateCatalog] [%s] PostCatalog before UpdatePostCatalog: %s",
		nodesArray[0].GetAddr(), nodesArray[0].PostCatalogToString())

	time.Sleep(2 * time.Second)

	err = nodesArray[0].UpdatePostCatalog()

	time.Sleep(2 * time.Second)

	require.NoError(t, err)
	myLogger().Info().Msgf("[Test_N2C_Chord_UpdateCatalog] [%s] PostCatalog after UpdatePostCatalog: %s",
		nodesArray[0].GetAddr(), nodesArray[0].PostCatalogToString())

}

// Generate a Chord network with multiple nodes, create a post, store it on the network, and retrieve it, then modify it, store it again and retrieve it again.
func Test_N2C_Chord_StoreAndRetrievePostFromChordAndUpdate(t *testing.T) {
	resetStorage()

	transp := channelFac()

	nbNodes := 3
	nodesArray, _, chordIDsArray, err := GenerateChordNodes(t, nbNodes, transp)

	for i := range nodesArray {
		myLogger().Info().Msgf("[Test_N2C_Chord_StoreAndRetrievePostFromChord] [%s] \n %s", nodesArray[i].GetAddr(), nodesArray[i].FingerTableToString())
	}

	ring, err := impl.DisplayRing(chordIDsArray)
	require.NoError(t, err)
	myLogger().Info().Msgf(ring)

	for i := range nodesArray {
		myLogger().Info().Msgf("[Test_N2C_Chord_StoreAndRetrievePostFromChord] [%s] \n%s", nodesArray[i].GetAddr(), nodesArray[i].PredAndSuccToString())
	}

	// Create and Store a post
	postContent1 := []byte("This is my first post")

	post := nodesArray[0].CreatePost(postContent1)
	err = nodesArray[0].StorePrivatePost(post)
	require.NoError(t, err)

	// convert postID str to bigIng
	postIDBytes, err := utils.DecodeChordIDFromStr(post.PostID)
	require.NoError(t, err)

	var labels []string

	// labels of nodes before posts
	for i := range chordIDsArray {
		labels = append(labels, fmt.Sprintf("n%d", i+1))
	}

	// add post IDs and labels
	chordIDsArray = append(chordIDsArray, postIDBytes)
	labels = append(labels, "p1")
	ring, err = impl.DisplayRing(chordIDsArray, labels...)
	require.NoError(t, err)
	myLogger().Info().Msgf(ring)

	myLogger().Info().Msgf("[Test_N2C_Chord_StoreAndRetrievePostFromChord] [%s] post1ByNode1: %v", nodesArray[0].GetAddr(), post)

	myLogger().Info().Msgf("[Test_N2C_Chord_StoreAndRetrievePostFromChord] ------------------------------ RETRIEVE POST" +
		"--------------------------")

	// retrieve post (possibly from another node) given postID
	retrievedPost, err := nodesArray[0].GetPost(post.PostID)
	require.NoError(t, err)
	myLogger().Info().Msgf("[Test_N2C_Chord_StoreAndRetrievePostFromChord] [%s] retrievedPost: %s", nodesArray[0].GetAddr(), retrievedPost)

	// retrieve post (possibly from another node) given postID
	retrievedPost2, err := nodesArray[1].GetPost(post.PostID)
	require.NoError(t, err)
	myLogger().Info().Msgf("[Test_N2C_Chord_StoreAndRetrievePostFromChord] [%s] retrievedPost: %s", nodesArray[1].GetAddr(), retrievedPost2)

	// Retrieved post must be equal to the original post
	require.True(t, retrievedPost.Equal(post))

	time.Sleep(500 * time.Millisecond)
	post.Likes = 10
	err = nodesArray[2].StorePrivatePost(post)
	require.NoError(t, err)
	// retrieve post (possibly from another node) given postID
	retrievedPost, err = nodesArray[0].GetPost(post.PostID)
	require.NoError(t, err)
	myLogger().Info().Msgf("[Test_N2C_Chord_StoreAndRetrievePostFromChord] [%s] retrievedPost: %s", nodesArray[0].GetAddr(), retrievedPost)

	// retrieve post (possibly from another node) given postID
	retrievedPost2, err = nodesArray[1].GetPost(post.PostID)
	require.NoError(t, err)
	myLogger().Info().Msgf("[Test_N2C_Chord_StoreAndRetrievePostFromChord] [%s] retrievedPost: %s", nodesArray[1].GetAddr(), retrievedPost2)
	require.Equal(t, 10, retrievedPost.Likes)
	require.Equal(t, 10, retrievedPost2.Likes)
}
