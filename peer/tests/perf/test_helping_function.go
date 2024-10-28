//go:build performance
// +build performance

package perf

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/n2ccrypto"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/peer/impl"
	"go.dedis.ch/cs438/storage"
	"go.dedis.ch/cs438/storage/file"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const KeyLen = n2ccrypto.DefaultKeyLen

var (
	// logout is the logger configuration
	logout = zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		FormatCaller: func(i interface{}) string {
			return filepath.Base(fmt.Sprintf("%s", i))
		},
	}
)

func myLogger() *zerolog.Logger {
	defaultLevel := zerolog.InfoLevel
	if strings.TrimSpace(os.Getenv("GLOG")) == "no" {
		defaultLevel = zerolog.Disabled
	}

	mylogger := zerolog.New(logout).
		Level(defaultLevel).
		With().Timestamp().Logger().
		With().Caller().Logger().
		With().Str("role", "test").Logger()
	return &mylogger
}

func resetStorage() {
	err := os.RemoveAll(impl.StoragePath)
	if err != nil {
		log.Error().Msgf("Err %v", err)
	}
	err = os.RemoveAll(impl.KeyStoragePath)
	if err != nil {
		log.Error().Msgf("Err %v", err)
	}
}

// GenerateChordNodes creates nbNodes nodes with chordIDs and returns the array of nodes, the array of chord nodes and the array of chord IDs
func GenerateChordNodes(t *testing.T, nbNodes int, transp transport.Transport) (nodesArray []z.TestNode, chordNodesArray []types.ChordNode, chordIDsArray [][KeyLen]byte, err error) {
	// Set number of nodes
	nodesArray = make([]z.TestNode, nbNodes)
	//var chordNodesArray []types.ChordNode
	//var chordIDsArray [][KeyLen]byte

	myLogger().Info().Msgf("[Test_GenerateChordNodes] Test Chord protocol with %d nodes", nbNodes)

	// Create nbNodes nodes.
	for i := range nodesArray {
		var store storage.Storage
		store, err = file.NewPersistency(impl.StoragePath + strconv.Itoa(i) + "/")
		if err != nil {
			myLogger().Error().Msgf("[GenerateChordNodes] failed creating NewPersistency storage")
		}

		myLogger().Info().Msgf("[Test_GenerateChordNodes] Creating node%d...", i+1)
		node := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(5*time.Second), z.WithChord(true))
		nodesArray[i] = node

		//myLogger().Info().Msgf("[Test_Multiple_Nodes] [%s] FingerTable after creating new node: %s", nodesArray[i].GetAddr(), nodesArray[i].FingerTableToString())
		chordNodesArray = append(chordNodesArray, nodesArray[i].GetChordNode())
	}

	// display the ring with nodes
	for _, chordNode := range chordNodesArray {
		chordIDsArray = append(chordIDsArray, chordNode.ID)
	}
	ring, err := impl.DisplayRing(chordIDsArray)
	require.NoError(t, err)
	myLogger().Info().Msgf(ring)

	// Join all nodes to the chord network
	for i, node := range nodesArray {
		if i == 0 {
			// if first node in the network
			err = node.Join(types.ChordNode{})
			require.NoError(t, err)
			myLogger().Info().Msgf("[Test_GenerateChordNodes] node%d.Join()............................", i)
		} else {
			// if not first node in the network, join chord node 1
			err = node.Join(nodesArray[0].GetChordNode())
			require.NoError(t, err)
			myLogger().Info().Msgf("[Test_GenerateChordNodes] node%d.Join(chordNode1)............................", i)
		}
	}
	return nodesArray, chordNodesArray, chordIDsArray, err
}

// Generate a post and store it
func GenerateAndStorePost(t require.TestingT, node peer.Peer, postNum int) (types.Post, error) {
	postContent := "Post " + strconv.Itoa(postNum) + "from node " + node.GetNodeIDStr()

	//_, err := rand.Read(postContent)
	//if err != nil {
	//	return types.Post{}, xerrors.Errorf("[GenerateAndStorePost] failed to read random bytes: %v", err)
	//}

	post := node.CreatePost([]byte(postContent))
	err := node.StorePrivatePost(post)
	if err != nil {
		return types.Post{}, xerrors.Errorf("[GenerateAndStorePost] failed to store post: %v", err)
	}

	return post, nil
}
