package impl

import (
	"encoding/json"
	fm "fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rs/xid"
	"go.dedis.ch/cs438/utils"

	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/entityverify"
	"go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/n2ccrypto"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

var (
	// logout is the logger configuration
	logout = zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		FormatCaller: func(i interface{}) string {
			return filepath.Base(fm.Sprintf("%s", i))
		},
	}
)

const KeyStoragePath = "../keyStorage/"
const StoragePath = "../fileStorage/"
const KeyLen = n2ccrypto.DefaultKeyLen

// public key for node
const PublicKeyFormatStr = "Node_%s_PublicKey"

// encrypted decode key for TargetNode
const NodeDecodingKeyFormatStr = "Node_%s_KeyID_%d_TargetNodeID_%s"

// NewPeer creates a new peer. You can change the content and location of this
// function but you MUST NOT change its signature and package location.
func NewPeer(conf peer.Configuration) peer.Peer {
	logger := loggerSetup()

	myAddr := conf.Socket.GetAddress()

	err := os.MkdirAll(KeyStoragePath, os.ModePerm)
	if err != nil {
		panic("failed to create KeyStorage folder")
	}

	var ppkFileName string
	var ppkModulePtr *n2ccrypto.PPKModule
	var tmpchordID [KeyLen]byte
	var posModule *entityverify.PoSModule
	var fingerTable SafeFingerTable
	if conf.WithChord {
		ppkFileName, ppkModulePtr, tmpchordID = createNodeID()
		posModule = entityverify.NewPoSModule(0, tmpchordID[:])
		// initialize finger table and start values
		logger.Debug().Msgf("[NewPeer] [%s] ChordID: (bigInt) [%v], (hex) [%x]", myAddr,
			utils.BytesToBigInt(tmpchordID), tmpchordID)
		fingerTable, err = NewSafeFingerTable(M, tmpchordID)
		if err != nil {
			panic("Fail to initialize a FingerTable")
		}
	}

	n := node{
		Configuration:   conf,
		stopCalledChan:  make(chan any),
		WaitGroup:       sync.WaitGroup{},
		routingTable:    routingTable{RoutingTable: make(map[string]string), Mutex: sync.Mutex{}},
		rumorTable:      rumorTable{RumorTable: make(map[string][]types.Rumor), Mutex: sync.Mutex{}},
		syncTable:       syncTable{SyncTable: make(map[string]chan any), RWMutex: sync.RWMutex{}},
		requests:        requests{Requests: make(map[string]struct{}), RWMutex: sync.RWMutex{}},
		catalog:         catalog{Catalog: make(map[string]map[string]struct{}), Mutex: sync.Mutex{}},
		searchReplies:   searchReplies{SearchReplies: make(map[string]map[string]bool), Mutex: sync.Mutex{}},
		pendingRequests: pendingRequests{PendingRequests: make(map[string]struct{}), Mutex: sync.Mutex{}},
		following:       following{Following: make(map[string][KeyLen]byte), Mutex: sync.Mutex{}},
		followers:       followers{Followers: make(map[string][KeyLen]byte), Mutex: sync.Mutex{}},
		blockedPeers:    blockedPeers{BlockedPeers: make(map[string]string), Mutex: sync.Mutex{}},
		flaggedPosts:    flaggedPosts{FlaggedPosts: make(map[string]map[string]struct{}), Mutex: sync.Mutex{}},
		nodesReputation: nodesReputation{NodesReputation: make(map[string]int), Mutex: sync.Mutex{}},
		challenges: challenges{Challenges: make(map[string]*types.PoSChallengeRequestMessage),
			Mutex: sync.Mutex{}},
		likes:                 likes{Likes: make(map[string]struct{}), Mutex: sync.Mutex{}},
		Logger:                logger,
		chordID:               tmpchordID,
		predecessorChord:      types.ChordNode{},
		fingerTable:           &fingerTable,
		chanHandler:           NewAsyncHandler(),
		ppkModule:             ppkModulePtr,
		ppkFilename:           ppkFileName,
		posModule:             posModule,
		hasJoinedChordNetwork: false,
		postPpkModules:        make([]*n2ccrypto.DCypherModule, 0),
	}

	n.Debug().Msgf("New node with ID: %v", utils.BytesToBigInt(n.chordID))

	mapAdd(n.RoutingTable, myAddr, myAddr, &n.routingTable.Mutex)

	// Register message callbacks.
	n.MessageRegistry.RegisterMessageCallback(testing.FakeMessage{}, n.FakeMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.ChatMessage{}, n.ChatMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.EmptyMessage{}, n.EmptyMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.RumorsMessage{}, n.RumorsMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.StatusMessage{}, n.StatusMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.AckMessage{}, n.AckMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.PrivateMessage{}, n.PrivateMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.DataRequestMessage{}, n.DataRequestMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.DataReplyMessage{}, n.DataReplyMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.SearchRequestMessage{}, n.SearchRequestMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.SearchReplyMessage{}, n.SearchReplyMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.FollowRequestMessage{}, n.FollowRequestMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.FollowReplyMessage{}, n.FollowReplyMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.FlagPostMessage{}, n.FlagPostMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.ChordRequestMessage{}, n.ChordRequestMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.ChordReplyMessage{}, n.ChordReplyMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.PoSChallengeRequestMessage{}, n.PoSChallengeRequestMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.PoSChallengeReplyMessage{}, n.PoSChallengeReplyMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.PostRequestMessage{}, n.PostRequestMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.PostReplyMessage{}, n.PostReplyMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.CatalogRequestMessage{}, n.CatalogRequestMessageCallback)
	n.MessageRegistry.RegisterMessageCallback(types.CatalogReplyMessage{}, n.CatalogReplyMessageCallback)

	if conf.WithChord && conf.PostEncryption {
		// Setup keys of node and store them on chord
		err = n.setupNodeKeys()
		if err != nil {
			panic("Failed to setup node keys!")
		}
	}

	return &n
}

func loggerSetup() zerolog.Logger {
	defaultLevel := zerolog.InfoLevel
	if strings.TrimSpace(os.Getenv("GLOG")) == "no" {
		defaultLevel = zerolog.Disabled
	}
	logger := zerolog.New(logout).
		Level(defaultLevel).
		With().Timestamp().Logger().
		With().Caller().Logger()
	return logger
}

// node implements a peer to build a Peerster system
//
// - implements peer.Peer
type node struct {
	peer.Peer
	// The configuration of the node.
	peer.Configuration
	// A channel with a flag that signals other channels to stop listening.
	stopCalledChan chan any
	// Used in order to wait for all goroutines to finish.
	sync.WaitGroup
	// origin -> relay map.
	routingTable
	// origin -> rumors from this origin.
	rumorTable
	// PacketID -> channel map, used for asynchronous notification.
	syncTable
	// RequestID -> origin map, used for asynchronous notification (searchRequest & dataRequest).
	requests
	// catalog
	catalog
	// requestID -> searchReplies map
	searchReplies
	// Pending (follow) requests.
	pendingRequests
	// Nodes that the peer is following.
	following
	// Nodes that follow the peer.
	followers
	// Nodes that are blocked by the peer.
	blockedPeers
	// Posts and the nodes that flagged them
	flaggedPosts
	// Reputation of the nodes
	nodesReputation
	// Challenges sent.
	challenges
	// Likes sent.
	likes
	// logger.
	zerolog.Logger
	// id of the predecessor node in the chord ring (SHA-256 hash)
	predecessorChord types.ChordNode
	// finger table of the node
	fingerTable *SafeFingerTable
	// thread safe map handling channels
	chanHandler AsyncHandler
	// PPKModule has node's keypairs and handles signing/encryption
	ppkModule *n2ccrypto.PPKModule
	// PPKModule filename
	ppkFilename string
	// PPKModules for post encryption
	postPpkModules []*n2ccrypto.DCypherModule
	// PoSModule is the node's proof of stake module
	posModule *entityverify.PoSModule
	// id of the node in the chord ring (SHA-256 hash)
	chordID [KeyLen]byte
	// boolean to check if the node has successfully joined the chord network
	hasJoinedChordNetwork bool
}

// Start implements peer.Service
func (n *node) Start() error {
	if n.checkStopCalled() {
		return nil
	}

	// Add to waiting group.
	n.Add(1)

	go func() {
		defer n.Done()

		for {
			if n.checkStopCalled() {
				return
			}

			pkt, err := n.Socket.Recv(time.Second * 1)
			if err != nil {
				_ = logAndReturnError(n.Logger, err)
				continue
			}

			if n.Configuration.RequestValidation {
				n.processPacketWithValidate(pkt)
			} else {
				if n.Configuration.WithChord {
					go n.processPacket(pkt)
				} else {
					n.processPacket(pkt)
				}
			}
		}
	}()

	if n.AntiEntropyInterval != 0 {
		n.antiEntropy()
	}
	if n.HeartbeatInterval != 0 {
		n.heartBeat()
	}

	return nil
}

// Stop implements peer.Service
func (n *node) Stop() error {
	close(n.stopCalledChan)
	n.Wait()

	// Here it is safe to re-open the stop channel in case Start() is called again.
	//n.stopCalledChan = make(chan any)

	return nil
}

// Unicast implements peer.Messaging
func (n *node) Unicast(dest string, msg transport.Message) error {
	// Check if we have a route to destination.
	relay, exists := mapRead(n.RoutingTable, dest, &n.routingTable.Mutex)
	if exists {
		source := n.Socket.GetAddress()

		if relay == dest {
			// Relay equals dest. We can send the package directly, acting ourselves as relay.
			relay = n.Socket.GetAddress()
			hdr := transport.NewHeader(source, relay, dest, 0)
			pkt := transport.Packet{Header: &hdr, Msg: &msg}
			n.sendPacketConcurrently(pkt, dest)
		} else {
			// Relay does not equal dest. We have to send the package to our relay.
			hdr := transport.NewHeader(source, source, dest, 0)
			pkt := transport.Packet{Header: &hdr, Msg: &msg}
			n.sendPacketConcurrently(pkt, relay)
		}

		return nil
	}

	errmsg := fm.Sprintf("(%s) Could not find peer to Unicast to!", n.Socket.GetAddress())
	return logAndReturnError(n.Logger, &PError{message: errmsg})
}

// Broadcast implements peer.Messaging
func (n *node) Broadcast(msg transport.Message) error {
	laddr := n.Socket.GetAddress()

	// Send RumorsMessage to random neighbor.
	rumorsMsg := types.RumorsMessage{Rumors: []types.Rumor{n.createRumor(msg)}}

	err := n.sendRumorsMessage(rumorsMsg, make([]string, 0))
	if err != nil {
		return logAndReturnError(n.Logger, err)
	}

	// Process packet locally.
	hdr := transport.NewHeader(laddr, laddr, laddr, 0)
	pkt := transport.Packet{Header: &hdr, Msg: &msg}
	if err := n.MessageRegistry.ProcessPacket(pkt); err != nil {
		return logAndReturnError(n.Logger, err)
	}

	return nil
}

// AddPeer implements peer.Service
func (n *node) AddPeer(addr ...string) {
	for _, a := range addr {
		// First check if "a" refers to ourselves.
		if a == n.Socket.GetAddress() {
			continue
		}

		mapAdd(n.RoutingTable, a, a, &n.routingTable.Mutex)
	}
}

// GetRoutingTable implements peer.Messaging
func (n *node) GetRoutingTable() peer.RoutingTable {
	return mapCopy(n.RoutingTable, &n.routingTable.Mutex)
}

// SetRoutingEntry implements peer.Messaging
func (n *node) SetRoutingEntry(origin, relayAddr string) {
	if relayAddr == "" {
		// Record must be deleted.
		mapDelete(n.RoutingTable, origin, &n.routingTable.Mutex)
	} else {
		// (Over)write entry.
		mapAdd(n.RoutingTable, origin, relayAddr, &n.routingTable.Mutex)
	}
}

func (n *node) Upload(data io.Reader) (string, error) {
	if n.ChunkSize == 0 {
		err := PError{message: "Invalid chunkSize provided (0)"}
		return "", logAndReturnError(n.Logger, &err)
	}

	blobStorage := n.Storage.GetDataBlobStore()
	chunk := make([]byte, n.ChunkSize)
	metaHashes := make([]string, 0)
	for {
		nread, err := data.Read(chunk)
		if nread > 0 {
			metaHash := storeChunk(blobStorage, chunk[:nread])
			metaHashes = append(metaHashes, metaHash)
		}
		if err == io.EOF || nread == 0 {
			break
		}
		if err != nil {
			return "", logAndReturnError(n.Logger, err)
		}
	}

	if len(metaHashes) == 0 {
		return "", nil
	}

	metaFileValue := ""
	for i, metaHash := range metaHashes {
		metaFileValue += metaHash
		if i != len(metaHashes)-1 {
			metaFileValue += peer.MetafileSep
		}
	}
	metaHashKey := storeChunk(blobStorage, []byte(metaFileValue))

	return metaHashKey, nil
}

func (n *node) Download(metahash string) ([]byte, error) {
	// Get metafile.
	metaFileValue, err := n.getMetaHashValue(metahash)
	if err != nil {
		return nil, logAndReturnError(n.Logger, err)
	}

	// Parse metafile.
	metaHashes := strings.Split(string(metaFileValue), peer.MetafileSep)

	// Reconstruct file.
	file := make([]byte, 0)
	for _, metaHash := range metaHashes {
		var chunk []byte
		chunk, err = n.getMetaHashValue(metaHash)
		if err != nil {
			return nil, logAndReturnError(n.Logger, err)
		}
		file = append(file, chunk...)
	}

	return file, nil
}

func (n *node) Tag(name string, mh string) error {
	n.Storage.GetNamingStore().Set(name, []byte(mh))

	return nil
}

func (n *node) Resolve(name string) (metahash string) {
	return string(n.Storage.GetNamingStore().Get(name))
}

func (n *node) GetCatalog() peer.Catalog {
	return n.Catalog
}

func (n *node) UpdateCatalog(key string, peer string) {
	if peer != n.Socket.GetAddress() {
		n.catalog.Mutex.Lock()
		defer n.catalog.Mutex.Unlock()

		catalogMap, exists := n.Catalog[key]
		if !exists {
			catalogMap = make(map[string]struct{})
		}
		// Does not matter if we overwrite here.
		catalogMap[peer] = struct{}{}
		n.Catalog[key] = catalogMap
	}
}

func (n *node) SearchAll(reg regexp.Regexp, budget uint, timeout time.Duration) ([]string, error) {
	localNames := n.searchAllLocal(reg)

	// Now get from neighbors.
	neighborNames, err := n.searchAllRemote(reg, budget, timeout)
	if err != nil {
		return nil, logAndReturnError(n.Logger, err)
	}

	return append(localNames, neighborNames...), nil
}

func (n *node) SearchFirst(pattern regexp.Regexp, conf peer.ExpandingRing) (string, error) {
	name := n.searchFirstLocal(pattern)
	if name != "" {
		return name, nil
	}

	name, err := n.searchFirstRemote(pattern, conf)
	if err != nil {
		return name, logAndReturnError(n.Logger, err)
	}

	return name, nil
}

func (n *node) StorePrivatePost(post types.Post) (err error) {
	myAddr := n.Socket.GetAddress()

	if !n.hasJoinedChordNetwork {
		return xerrors.Errorf("[StorePrivatePost] [%s] Cannot store a private post before "+
			"joining the chord network", myAddr)
	}

	if post.Content == nil {
		return xerrors.Errorf("[StorePrivatePost] [%s] Cannot store an empty content post", myAddr)
	}

	if n.Configuration.PostEncryption {

		// Encrypt post content
		mostRecentPPKModule, keyID := n.getMostRecentPostPPKModule()
		n.Debug().Msgf("[StorePrivatePost] most recent PPK keyID %v ", keyID)
		postContent, err := mostRecentPPKModule.Encrypt(post.Content)

		if err != nil {
			n.Debug().Msgf("[StorePrivatePost] [%s] couldnt encrypt private post content before storing", myAddr)
			return err
		}
		post.Content = postContent
		post.KeyID = keyID

	}

	// convert the post chord id to []byte format
	keyChordID, err := utils.DecodeChordIDFromStr(post.PostID)
	if err != nil {
		n.Error().Msgf("[StorePrivatePost] [%s] failed to DecodeChordIDFromStr for str [%s]: %v",
			myAddr, post.PostID, err)
	}

	n.Debug().Msgf("[StorePrivatePost] [%s] keyChordID [%s]", myAddr, utils.BytesToBigInt(keyChordID))

	// Find 1st node whose id is equal or follows k in the Chord ring
	keySuccessorNode, err := n.findSuccessor(keyChordID)
	if err != nil {
		n.Error().Msgf("[StorePrivatePost] [%s] failed to findSuccessor of keyChordID [%s]: %v",
			myAddr, keyChordID, err)
		return err
	}
	n.Debug().Msgf("[StorePrivatePost] [%s] found keySuccessorNode [%s]", myAddr, keySuccessorNode.IPAddr)

	postIDBigInt, err := utils.StrToBigInt(post.PostID)
	if err != nil {
		return xerrors.Errorf("[StorePrivatePost] failed to convert postID [%s] to bigInt: %v", post.PostID, err)
	}

	// if k's successor node is our node, store the data locally
	if keySuccessorNode.IPAddr == myAddr {
		n.Debug().Msgf("[StorePrivatePost] [%s] key's successor node [%s] is ourself. "+
			"Store post with PostID [%d] to PostStore.", myAddr, keySuccessorNode.IPAddr, postIDBigInt)

		postBytes, err := json.Marshal(post)
		if err != nil {
			return logAndReturnError(n.Logger, err)
		}

		n.Storage.GetPostStore().Set(post.PostID, postBytes)
	} else {
		// create a message containing the Post and send it to k's successor node
		msg := types.ChordRequestMessage{
			RequestID: xid.New().String(),
			Query:     types.PostRequest,
			Post:      post,
			IsPrivate: true,
		}
		transpMsg, _ := n.TypesMsgToTranspMsg(msg)

		// send PostMessage and wait for PostReplyMessage / acknowledgment
		_, err = n.sendRequestAndWaitReply(keySuccessorNode, msg.RequestID, transpMsg, types.ChordQueryStr[msg.Query])
		if err != nil {
			return xerrors.Errorf("[StorePrivatePost] failed to sendRequestAndWaitReply: %v", err)
		}
	}

	// store private post in PostCatalog
	n.Debug().Msgf("[StorePrivatePost] [%s] Store private post with PostID [%d] in PostCatalog",
		myAddr, postIDBigInt)
	err = n.storePrivatePostInCatalog(post)
	if err != nil {
		return xerrors.Errorf("[StorePrivatePost] failed to store private post in PostCatalog: %v", err)
	}
	return nil
}

func (n *node) StorePublicPost(post types.Post) error {
	myAddr := n.Socket.GetAddress()

	if !n.hasJoinedChordNetwork {
		return xerrors.Errorf("[StorePublicPost] [%s] Cannot store a public post before "+
			"joining the chord network", myAddr)
	}

	if post.Content == nil {
		return xerrors.Errorf("[StorePrivatePost] [%s] Cannot store an empty content post", myAddr)
	}

	// convert the post chord id to []byte format
	keyChordID, err := utils.DecodeChordIDFromStr(post.PostID)
	if err != nil {
		n.Error().Msgf("[StorePublicPost] [%s] failed to DecodeChordIDFromStr for str [%s]: %v",
			myAddr, post.PostID, err)
	}
	n.Debug().Msgf("[StorePublicPost] [%s] keyChordID [%s]", myAddr, utils.BytesToBigInt(keyChordID))

	// Find 1st node whose id is equal or follows k in the Chord ring
	keySuccessorNode, err := n.findSuccessor(keyChordID)
	if err != nil {
		n.Error().Msgf("[StorePublicPost] [%s] failed to findSuccessor of keyChordID [%s]: %v",
			myAddr, keyChordID, err)
		return err
	}
	n.Debug().Msgf("[StorePublicPost] [%s] found keySuccessorNode [%s]", myAddr, keySuccessorNode.IPAddr)

	postIDBigInt, err := utils.StrToBigInt(post.PostID)
	if err != nil {
		return xerrors.Errorf("[PostMessageCallback] failed to convert postID [%s] to bigInt: %v", post.PostID, err)
	}

	// if k's successor node is our node, store the data locally
	if keySuccessorNode.IPAddr == myAddr {
		n.Debug().Msgf("[StorePublicPost] [%s] key's successor node [%s] is ourself. "+
			"Store post locally and return.", myAddr, keySuccessorNode.IPAddr)

		post.KeyID = 0
		postBytes, err := json.Marshal(post)
		if err != nil {
			return logAndReturnError(n.Logger, err)
		}
		n.Storage.GetPostStore().Set(post.PostID, postBytes)

	} else {
		// create a message containing the Post and send it to k's successor node
		msg := types.ChordRequestMessage{
			RequestID: xid.New().String(),
			Query:     types.PostRequest,
			Post:      post,
			IsPrivate: false,
		}
		transpMsg, _ := n.TypesMsgToTranspMsg(msg)

		// send PostMessage and wait for PostReplyMessage / acknowledgment
		_, err = n.sendRequestAndWaitReply(keySuccessorNode, msg.RequestID, transpMsg, types.ChordQueryStr[msg.Query])
		if err != nil {
			return xerrors.Errorf("[StorePublicPost] failed to sendRequestAndWaitReply: %v", err)
		}
	}

	// store public post in PostCatalog
	n.Debug().Msgf("[StorePublicPost] [%s] Store public post with PostID [%d] in catalog",
		myAddr, postIDBigInt)
	err = n.storePublicPostInCatalog(post)
	if err != nil {
		return xerrors.Errorf("[StorePublicPost] failed to store public post in PostCatalog: %v", err)
	}
	return nil
}

func (n *node) GetPost(postID string) (types.Post, error) {
	myAddr := n.Socket.GetAddress()

	// convert the post chord id to []byte format
	keyChordID, err := utils.DecodeChordIDFromStr(postID)
	if err != nil {
		n.Error().Msgf("[GetPost] [%s] failed to DecodeChordIDFromStr for str [%s]: %v", myAddr, postID, err)
		return types.Post{}, err
	}
	n.Debug().Msgf("[GetPost] [%s] Get Post with keyChordID [%s]", myAddr, utils.BytesToBigInt(keyChordID))

	// Find 1st node whose id is equal or follows k in the Chord ring
	keySuccessorNode, err := n.findSuccessor(keyChordID)
	if err != nil {
		n.Error().Msgf("[GetPost] [%s] failed to findSuccessor of keyChordID [%s]: %v", myAddr, keyChordID, err)
		return types.Post{}, err
	}

	// if k's successor node is our node, retrieve the data locally
	if keySuccessorNode.IPAddr == myAddr {
		postBytes := n.Storage.GetPostStore().Get(postID)
		res := types.Post{}
		err = json.Unmarshal(postBytes, &res)
		if err != nil {
			return types.Post{}, logAndReturnError(n.Logger, err)
		}

		if n.Configuration.PostEncryption {
			// decrypt post content
			res.Content, err = n.decryptPostContent(&res)
			if err != nil {
				res.Content = append([]byte("ENCRYPTED POST CONTENT>>"), res.Content...)
			}
		}
		return res, nil
	}

	// create a message containing the PostID of the post to retrieve and send it to k's successor node
	postReqMsg := types.PostRequestMessage{
		RequestID: xid.New().String(),
		PostID:    postID,
	}
	transpPosReqtMsg, _ := n.TypesMsgToTranspMsg(postReqMsg)

	post, err := n.waitForPostMessageAck(postReqMsg, keySuccessorNode, transpPosReqtMsg)
	if err != nil {
		return post, logAndReturnError(n.Logger, err)
	}

	return post, err
}

// UpdatePostCatalog function sends a message to all followed peers
// to get their post catalog and update ours with missing postIDs
func (n *node) UpdatePostCatalog() error {
	myAddr := n.Socket.GetAddress()

	n.Debug().Msgf("[UpdatePostCatalog] [%s] Get catalog from all followed peers", myAddr)

	n.following.Mutex.Lock()
	defer n.following.Mutex.Unlock()
	// send a message to my followed peers to get their catalog
	for followedIPAddr, followedChordID := range n.Following {
		postCatalog, err := n.rpcGetPostCatalog(types.ChordNode{ID: followedChordID, IPAddr: followedIPAddr})
		if err != nil {
			return xerrors.Errorf("[UpdatePostCatalog] failed to get catalog from peer: %v", err)
		}

		n.Debug().Msgf("[UpdatePostCatalog] [%s] Update my catalog with [%s]'s catalog", myAddr, followedIPAddr)
		// update my catalog with theirs
		for authorIDForeign, postIDsByAuthorBytesForeign := range postCatalog {
			n.Debug().Msgf("[UpdatePostCatalog] [%s] Update my catalog PostsIDs of [%s]'s from [%s] catalog",
				myAddr, authorIDForeign, followedIPAddr)

			postIDsByAuthor, err := utils.ByteArrayToMapStringStruct(n.Storage.GetPostCatalogStore().Get(authorIDForeign))
			if err != nil {
				return logAndReturnError(n.Logger, xerrors.Errorf("%v", err))
			}

			postIDsByAuthorForeign, err := utils.ByteArrayToMapStringStruct(postIDsByAuthorBytesForeign)
			if err != nil {
				return logAndReturnError(n.Logger, xerrors.Errorf("%v", err))
			}

			for postID := range postIDsByAuthorForeign {
				postIDsByAuthor[postID] = struct{}{}
			}

			newPostIDsByAuthorBytes, err := json.Marshal(postIDsByAuthor)
			if err != nil {
				return logAndReturnError(n.Logger, xerrors.Errorf("%v", err))
			}

			n.Storage.GetPostCatalogStore().Set(authorIDForeign, newPostIDsByAuthorBytes)
		}
	}
	n.Debug().Msgf("[UpdatePostCatalog] [%s] Successfully updated my Post Catalog Store (len=%d)", myAddr,
		n.Storage.GetPostCatalogStore().Len())
	return nil
}

func (n *node) GetPrivateFeed() ([]types.Post, error) {
	myAddr := n.Socket.GetAddress()
	if !n.hasJoinedChordNetwork {
		return []types.Post{}, xerrors.Errorf("[GetPrivateFeed] [%s] Cannot get private feed before "+
			"joining the chord network", myAddr)
	}

	err := n.UpdatePostCatalog()
	if err != nil {
		return nil, logAndReturnError(n.Logger, xerrors.Errorf("%v", err))
	}
	n.following.Lock()
	defer n.following.Unlock()

	var postIDs = make(map[string]struct{})

	for _, followingAuthorID := range n.following.Following {
		var postIDsFromAuthor map[string]struct{}
		postIDsByAuthorBytes := n.Storage.GetPostCatalogStore().Get(utils.EncodeChordIDToStr(followingAuthorID))
		if postIDsByAuthorBytes != nil {
			err := json.Unmarshal(postIDsByAuthorBytes, &postIDsFromAuthor)
			if err != nil {
				return nil, logAndReturnError(n.Logger, xerrors.Errorf("%v", err))
			}
		}

		for pidFromAuthor := range postIDsFromAuthor {
			postIDs[pidFromAuthor] = struct{}{}
		}

	}

	var postIDsFromAuthor map[string]struct{}
	postIDsByAuthorBytes := n.Storage.GetPostCatalogStore().Get(n.GetNodeIDStr())
	if postIDsByAuthorBytes != nil {
		err := json.Unmarshal(postIDsByAuthorBytes, &postIDsFromAuthor)
		if err != nil {
			return nil, logAndReturnError(n.Logger, xerrors.Errorf("%v", err))
		}
	}

	for pidFromAuthor := range postIDsFromAuthor {
		postIDs[pidFromAuthor] = struct{}{}
	}

	postArray := make([]types.Post, 0)

	for postID := range postIDs {
		if !n.isPostFlaggedByMe(postID) {
			post, err := n.GetPost(postID)
			if err != nil {
				return nil, logAndReturnError(n.Logger, xerrors.Errorf("%v", err))
			}
			postArray = append(postArray, post)
		}
	}

	n.Debug().Msgf("[%s] CHECKPOINT", n.Socket.GetAddress())

	return postArray, nil
}

func (n *node) GetPublicFeed() ([]types.Post, error) {
	myAddr := n.Socket.GetAddress()
	if !n.hasJoinedChordNetwork {
		return []types.Post{}, xerrors.Errorf("[GetPublicFeed] [%s] Cannot get public feed before "+
			"joining the chord network", myAddr)
	}

	err := n.UpdatePostCatalog()
	if err != nil {
		return nil, logAndReturnError(n.Logger, xerrors.Errorf("%v", err))
	}

	postIDsByAuthorBytes := n.Storage.GetPostCatalogStore().Get(peer.PublicPostPk)
	var postIDs map[string]struct{}
	if postIDsByAuthorBytes != nil {
		err := json.Unmarshal(postIDsByAuthorBytes, &postIDs)
		if err != nil {
			return nil, logAndReturnError(n.Logger, xerrors.Errorf("%v", err))
		}
	}

	postArray := make([]types.Post, 0)

	for postID := range postIDs {
		if !n.isPostFlaggedByMe(postID) {
			post, err := n.GetPost(postID)
			/*
				if err != nil {
					return nil, logAndReturnError(n.Logger, xerrors.Errorf("%v", err))
				}
			*/
			if err == nil && !n.isBlocked(post.AuthorID) {
				postArray = append(postArray, post)
			}
		}
	}

	return postArray, nil
}

// Like adds a like to a post
func (n *node) Like(postID string) error {
	post, err := n.GetPost(postID)
	if err != nil {
		return logAndReturnError(n.Logger, err)
	}

	_, exists := mapRead(n.Likes, postID, &n.likes.Mutex)
	if !exists {
		// Post not already liked
		post.Likes++
		err = n.StorePrivatePost(post)
		if err != nil {
			return logAndReturnError(n.Logger, err)
		}
		mapAdd(n.Likes, postID, struct{}{}, &n.likes.Mutex)

		return nil
	}

	return fm.Errorf("you already liked this post")
}

// Comment adds a comment to a post
func (n *node) Comment(postID string, content string) error {
	post, err := n.GetPost(postID)
	if err != nil {
		return logAndReturnError(n.Logger, err)
	}

	post.Comments = append(post.Comments, n.createComment(postID, content))

	err = n.StorePrivatePost(post)
	if err != nil {
		return logAndReturnError(n.Logger, err)
	}

	return nil
}

func (n *node) FollowRequest(peer string) error {
	// Send a follow request.
	requestID, err := n.sendFollowRequestMessage(peer)
	if err != nil {
		return logAndReturnError(n.Logger, err)
	}

	// Add it to pending requests (requests that are not answered yet).
	mapAdd(n.pendingRequests.PendingRequests, requestID, struct{}{}, &n.pendingRequests.Mutex)

	return nil
}

func (n *node) StopFollowing(peer string) {
	ok := mapDelete(n.following.Following, peer, &n.following.Mutex)
	if !ok {
		fm.Printf(Yellow("Request to unfollow peer (%s) that is not followed!"), peer)
		n.Info().Msgf("[%s] Request to unfollow peer (%s) that is not followed!", n.Socket.GetAddress(), peer)
		return
	}
}

func (n *node) RemoveFollower(peer string) error {

	// Remove peer from followers list.
	_ = mapDelete(n.followers.Followers, peer, &n.followers.Mutex)

	if n.Configuration.PostEncryption {
		err := n.generateNewPostPPKModule()
		if err != nil {
			fm.Printf(Red("Error occurred while trying to block %s!"), peer)
			return logAndReturnError(n.Logger, err)
		}
	}

	return nil
}

func (n *node) Block(peer string, nodeID string) error {
	// Add peer to blocked peers.
	mapAdd(n.blockedPeers.BlockedPeers, peer, nodeID, &n.blockedPeers.Mutex)

	// Stop following this peer.
	if containsElement(n.GetFollowing(), peer) || containsElement(n.GetFollowing(), nodeID) {
		n.StopFollowing(peer)
	}

	// remove peer as follower
	err := n.RemoveFollower(peer)

	fm.Printf(Green("Blocked peer (%s)!\n"), peer)

	return err
}

func (n *node) GetFollowing() []string {
	return mapKeys(n.following.Following, &n.following.Mutex)
}

func (n *node) GetFollowers() []string {
	return mapKeys(n.followers.Followers, &n.followers.Mutex)
}

func (n *node) FlagPost(postID string) error {
	if n.isPostFlaggedByMe(postID) {
		return nil
	}
	post, err := n.GetPost(postID)
	if err != nil {
		return logAndReturnError(n.Logger, err)
	}
	n.registerPostFlag(postID, n.GetNodeIDStr())
	n.dropReputation(post.AuthorID)
	// Broadcast post flag.
	err = n.broadcastPostFlag(postID)
	if err != nil {
		return logAndReturnError(n.Logger, err)
	}
	return nil
}

func (n *node) GetReputation() map[string]int {
	return mapCopy(n.NodesReputation, &n.nodesReputation.Mutex)
}

func (n *node) GetFlagsForPost(postID string) (map[string]struct{}, bool) {
	// get the set of peer who flagged this post
	return n.getFlagsForPost(postID)
}

func (n *node) GetNodeID() [KeyLen]byte {
	return n.chordID
}

func (n *node) GetNodeIDStr() string {
	return utils.EncodeChordIDToStr(n.chordID)
}

func (n *node) GetAddress() string {
	return n.Socket.GetAddress()
}

func (n *node) GetNeighbors() []string {
	return n.getNeighbors("")
}
