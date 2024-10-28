package impl

import (
	"encoding/json"
	"errors"
	fm "fmt"
	"regexp"

	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"go.dedis.ch/cs438/utils"
	"golang.org/x/xerrors"
)

func (n *node) FakeMessageCallback(msg types.Message, pkt transport.Packet) error {
	return nil
}

func (n *node) ChatMessageCallback(msg types.Message, pkt transport.Packet) error {
	return nil
}

func (n *node) EmptyMessageCallback(msg types.Message, pkt transport.Packet) error {
	return nil
}

func (n *node) RumorsMessageCallback(msg types.Message, pkt transport.Packet) error {
	rumorsMsg, ok := msg.(*types.RumorsMessage)

	if !ok {
		errmsg := fm.Sprintf("Unable to convert message to RumorsMessage! %s\n", msg)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	neighbors := n.getNeighbors("")

	foundExpected := false
	for _, rumor := range rumorsMsg.Rumors {
		lastRumorSeqno := n.getLastRumorSeqno(rumor.Origin)

		if rumor.Sequence != lastRumorSeqno+1 {
			continue
		}

		foundExpected = true

		// Update (local) rumor table
		n.updateRumorTable(rumor)

		if n.Socket.GetAddress() != rumor.Origin && !containsElement(neighbors, rumor.Origin) {
			mapAdd(n.RoutingTable, rumor.Origin, pkt.Header.RelayedBy, &n.routingTable.Mutex)
		}

		if n.Socket.GetAddress() != rumor.Origin {
			// Use registry to process the embedded message.
			err := n.MessageRegistry.ProcessPacket(transport.Packet{Header: pkt.Header, Msg: rumor.Msg})
			if err != nil {
				return err
			}
		}
	}

	if pkt.Header.Source != n.Socket.GetAddress() {
		// Send ACK message to source.
		newPkt := transport.Packet{Header: pkt.Header, Msg: pkt.Msg}
		err := n.sendACK(newPkt, pkt.Header.Source)
		if err != nil {
			return err
		}
	}

	if foundExpected && n.Socket.GetAddress() != pkt.Header.Source {
		// Send to another random neighbor (excluding the one we received this from).
		excludeList := []string{pkt.Header.Source}
		return n.sendRumorsMessage(*rumorsMsg, excludeList)
	}

	return nil
}

func (n *node) StatusMessageCallback(msg types.Message, pkt transport.Packet) error {
	var err error

	remoteStatusMessage, ok := msg.(*types.StatusMessage)
	remoteAddr := pkt.Header.Source

	if msg == nil {
		return nil
	}

	if !ok {
		errmsg := fm.Sprintf("Unable to convert message %s to StatusMessage", msg)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	localStatusMessage := n.getStatusMessage()

	if n.statusMessagesEqual(localStatusMessage, *remoteStatusMessage) {
		err = n.forwardStatusMessage(&localStatusMessage, remoteAddr)
		if err != nil {
			if errors.Is(err, NoNeighborsError{}) {
				// Could not find any neighbor, exit gracefully.
				return nil
			}
			return logAndReturnError(n.Logger, err)
		}

		return nil
	}

	sendRumors, sendStatus := n.checkStatusMessages(localStatusMessage, *remoteStatusMessage)

	if sendRumors {
		err = n.sendMissingRumors(n.getMissingRumors(localStatusMessage, *remoteStatusMessage), remoteAddr)
		if err != nil {
			return logAndReturnError(n.Logger, err)
		}
	}
	if sendStatus {
		err = n.sendStatusMessage(localStatusMessage, remoteAddr)
		if err != nil {
			return logAndReturnError(n.Logger, err)
		}
	}

	return nil
}

func (n *node) AckMessageCallback(msg types.Message, pkt transport.Packet) error {
	ackMsg, ok := msg.(*types.AckMessage)

	if !ok {
		errmsg := fm.Sprintf("Unable to convert message %s to AckMessage!\n", msg)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	ackChannel, exists := mapReadRW(n.SyncTable, ackMsg.AckedPacketID, &n.syncTable.RWMutex)
	if exists {
		close(ackChannel)
	}

	statusMsg, err := n.MessageRegistry.MarshalMessage(&ackMsg.Status)
	if err != nil {
		return logAndReturnError(n.Logger, err)
	}
	statusPkt := transport.Packet{Header: pkt.Header, Msg: &statusMsg}

	return n.MessageRegistry.ProcessPacket(statusPkt)
}

func (n *node) PrivateMessageCallback(msg types.Message, pkt transport.Packet) error {
	privateMsg, ok := msg.(*types.PrivateMessage)

	if !ok {
		errmsg := fm.Sprintf("Unable to convert message to PrivateMessage! %s\n", msg)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	// Check if the peer is in the list of recipients.
	addr := n.Socket.GetAddress()
	if _, exists := privateMsg.Recipients[addr]; !exists {
		return nil
	}

	return n.MessageRegistry.ProcessPacket(transport.Packet{Header: pkt.Header, Msg: privateMsg.Msg})
}

func (n *node) DataRequestMessageCallback(msg types.Message, pkt transport.Packet) error {
	dataRequestMsg, ok := msg.(*types.DataRequestMessage)

	if !ok {
		errmsg := fm.Sprintf("Unable to convert message to DataRequestMessage! %s\n", msg)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	_, exists := mapReadRW(n.Requests, dataRequestMsg.RequestID, &n.requests.RWMutex)
	if exists {
		// Duplicate
		return nil
	}

	mapAddRW(n.Requests, dataRequestMsg.RequestID, struct{}{}, &n.requests.RWMutex)

	blobStore := n.Storage.GetDataBlobStore()
	key := dataRequestMsg.Key
	val := blobStore.Get(dataRequestMsg.Key)
	dataReplyMessage := types.DataReplyMessage{RequestID: dataRequestMsg.RequestID, Key: key, Value: val}
	dataReplyMsg, err := n.MessageRegistry.MarshalMessage(dataReplyMessage)
	if err != nil {
		return logAndReturnError(n.Logger, err)
	}

	return n.Unicast(pkt.Header.Source, dataReplyMsg)
}

func (n *node) DataReplyMessageCallback(msg types.Message, pkt transport.Packet) error {
	dataReplyMsg, ok := msg.(*types.DataReplyMessage)

	if !ok {
		errmsg := fm.Sprintf("Unable to convert message to DataReplyMessage! %s\n", msg)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	n.Storage.GetDataBlobStore().Set(dataReplyMsg.Key, dataReplyMsg.Value)

	dataRequestChannel, exists := mapReadRW(n.SyncTable, dataReplyMsg.RequestID, &n.syncTable.RWMutex)
	if exists {
		close(dataRequestChannel)
	}

	return nil
}

func (n *node) SearchRequestMessageCallback(msg types.Message, pkt transport.Packet) error {
	searchRequestMsg, ok := msg.(*types.SearchRequestMessage)

	if !ok {
		errmsg := fm.Sprintf("Unable to convert message to SearchRequestMessage! %s\n", msg)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	reg := regexp.MustCompile(searchRequestMsg.Pattern)

	_, exists := mapReadRW(n.Requests, searchRequestMsg.RequestID, &n.requests.RWMutex)
	if exists {
		// Duplicate
		return nil
	}

	mapAddRW(n.Requests, searchRequestMsg.RequestID, struct{}{}, &n.requests.RWMutex)

	searchRequestMsg.Budget--
	if searchRequestMsg.Budget > 0 {
		err := n.forwardSearchToNeighbors(searchRequestMsg, n.getNeighbors(pkt.Header.Source))
		if err != nil {
			return logAndReturnError(n.Logger, err)
		}
	}

	responses := n.getSearchRequestResponses(*reg)

	searchReplyMessage := types.SearchReplyMessage{RequestID: searchRequestMsg.RequestID, Responses: responses}
	searchReplyMsg, err := n.MessageRegistry.MarshalMessage(searchReplyMessage)
	if err != nil {
		return logAndReturnError(n.Logger, err)
	}

	hdr := transport.NewHeader(n.Socket.GetAddress(), n.Socket.GetAddress(), searchRequestMsg.Origin, 0)
	n.sendPacketConcurrently(transport.Packet{Header: &hdr, Msg: &searchReplyMsg}, pkt.Header.RelayedBy)

	return nil
}

func (n *node) SearchReplyMessageCallback(msg types.Message, pkt transport.Packet) error {
	searchReplyMsg, ok := msg.(*types.SearchReplyMessage)

	if !ok {
		errmsg := fm.Sprintf("Unable to convert message to SearchReplyMessage! %s\n", msg)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	n.updateNamingStorage(searchReplyMsg, pkt.Header.Source)
	n.updateSearchReplies(searchReplyMsg)

	return nil
}

func (n *node) FollowRequestMessageCallback(msg types.Message, pkt transport.Packet) error {
	followRequestMsg, ok := msg.(*types.FollowRequestMessage)

	if !ok {
		errmsg := fm.Sprintf("Unable to convert message to FollowRequestMessage! %s\n", msg)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	// Ignore the request if the peer is blocked.
	_, exists := mapRead(n.blockedPeers.BlockedPeers, pkt.Header.Source, &n.blockedPeers.Mutex)
	if exists {
		fm.Printf(Yellow("Dropping follow request from blocked peer %s"), pkt.Header.Source)
		return nil
	}

	accept := n.Configuration.FollowRequestUserPrompt(pkt.Header.Source)

	if accept {
		// Add them to our followers list.
		mapAdd(n.followers.Followers, pkt.Header.Source, followRequestMsg.ChordID, &n.followers.Mutex)
		if n.Configuration.PostEncryption {
			n.updateNodePostKeys()
		}
		return n.sendFollowReplyMessage(followRequestMsg, pkt.Header.Source)
	}

	return nil
}

func (n *node) FollowReplyMessageCallback(msg types.Message, pkt transport.Packet) error {
	followReplyMsg, ok := msg.(*types.FollowReplyMessage)

	if !ok {
		errmsg := fm.Sprintf("Unable to convert message to FollowReplyMessage! %s\n", msg)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	n.Logger.Debug().Msgf("[%s] Received follow reply message from %s", n.Socket.GetAddress(), pkt.Header.Source)

	_, exists := mapRead(n.pendingRequests.PendingRequests, followReplyMsg.RequestID, &n.pendingRequests.Mutex)
	if exists {
		//n.Configuration.FollowReplyDisplay(pkt.Header.Source)
		n.Logger.Debug().Msgf("[%s] Added %s to following list!", n.Socket.GetAddress(), pkt.Header.Source)
		// We have indeed sent a follow request to that peer. Remove from pending and add them to our following list.
		mapDelete(n.pendingRequests.PendingRequests, followReplyMsg.RequestID, &n.pendingRequests.Mutex)
		mapAdd(n.following.Following, pkt.Header.Source, followReplyMsg.ChordID, &n.following.Mutex)
	}

	return nil
}

func (n *node) BlockMessageCallback(msg types.Message, pkt transport.Packet) error {
	blockMsg, ok := msg.(*types.BlockMessage)
	if !ok {
		errmsg := fm.Sprintf("Unable to convert message to BlockMessage! %s\n", msg)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	n.following.Lock()
	defer n.following.Unlock()

	// Replace chord id with the new one.
	_, exists := n.following.Following[pkt.Header.Source]
	if exists {
		n.following.Following[pkt.Header.Source] = blockMsg.NewChordID
	}

	return nil
}

func (n *node) FlagPostMessageCallback(msg types.Message, pkt transport.Packet) error {
	flagPostMsg, ok := msg.(*types.FlagPostMessage)

	if !ok {
		errmsg := fm.Sprintf("Unable to convert message to FlagPostMessage! %s\n", msg)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}
	post, err := n.GetPost(flagPostMsg.PostID)
	if err != nil {
		return logAndReturnError(n.Logger, err)
	}
	_, isFriend := mapRead(n.following.Following, flagPostMsg.FlaggerID, &n.following.Mutex)
	if n.isPostFlaggedByMe(post.PostID) || !isFriend {
		return nil
	}
	n.registerPostFlag(post.PostID, flagPostMsg.FlaggerID)
	if n.isPostFlaggedByMyFriends(post.PostID) {
		return n.FlagPost(post.PostID)
	}
	return nil
}

func (n *node) ChordRequestMessageCallback(msg types.Message, pkt transport.Packet) error {
	myAddr := n.Socket.GetAddress()

	n.AddPeer(pkt.Header.Source)

	chordQueryMsg, ok := msg.(*types.ChordRequestMessage)
	if !ok {
		return xerrors.Errorf("[ChordRequestMessageCallback] "+
			"[%s] Unable to convert message [%s] to ChordRequestMessage!\n", n.Socket.GetAddress(), msg)
	}
	//n.Debug().Msgf("[ChordRequestMessageCallback] [%s] handle chord query msg [%s] of type [%d: %s] received"+
	//	" from [%s]", myAddr, chordQueryMsg.RequestID, chordQueryMsg.Query,
	//	types.ChordQueryStr[chordQueryMsg.Query], pkt.Header.Source)

	var chordNode types.ChordNode
	var err error
	switch chordQueryMsg.Query {
	case types.RPCSuccessor:
		//n.Debug().Msgf("[ChordRequestMessageCallback] [%s] case RPCSuccessor: Get mySucessor node", myAddr)
		chordNode = n.MySuccessor()
		n.Debug().Msgf("[ChordRequestMessageCallback] [%s] case RPCSuccessor: mySucessor node [%v]",
			myAddr, chordNode)
	case types.RPCClosestPrecedingFinger:
		//n.Debug().Msgf("[ChordRequestMessageCallback] [%s] case RPCClosestPrecedingFinger", myAddr)
		chordNode = n.closestPrecedingFinger(chordQueryMsg.ChordNode.ID)
		n.Debug().Msgf("[ChordRequestMessageCallback] [%s] closest preceding finger found: [%s]",
			myAddr, chordNode.IPAddr)
	case types.RPCFindSuccessor:
		//n.Debug().Msgf("[ChordRequestMessageCallback] [%s] case RPCFindSuccessor", myAddr)
		chordNode, err = n.findSuccessor(chordQueryMsg.ChordNode.ID)
		if err != nil {
			return xerrors.Errorf("[ChordRequestMessageCallback] "+
				"failed to get chordNode (types.RPCFindSuccessor: %v)", err)
		}
	case types.RPCGetPredecessor:
		//n.Debug().Msgf("[ChordRequestMessageCallback] [%s] case RPCGetPredecessor", myAddr)
		chordNode = n.predecessorChord
	case types.RPCSetPredecessor:
		n.predecessorChord = chordQueryMsg.ChordNode
		n.Debug().Msgf("[ChordRequestMessageCallback] [%s] Set predecessor to: %s", myAddr, n.predecessorChord.IPAddr)
	case types.RPCSetSuccessor:
		err := n.fingerTable.FingerTableSetNode(0, chordQueryMsg.ChordNode)
		if err != nil {
			return xerrors.Errorf("[ChordRequestMessageCallback] failed to handle Set Successor: %v", err)
		}
		n.Debug().Msgf("[ChordRequestMessageCallback] [%s] Set successor to: %s", myAddr, chordQueryMsg.ChordNode)
	case types.RPCUpdateFingerTable:
		n.Debug().Msgf("[ChordRequestMessageCallback] [%s] case RPCUpdateFingerTable", myAddr)
		err = n.updateFingerTable(chordQueryMsg.ChordNode, chordQueryMsg.IthFinger)
		if err != nil {
			return err
		}
	case types.PostRequest:
		n.Debug().Msgf("[ChordRequestMessageCallback] [%s] case PostRequest", myAddr)
		err = n.handlePostRequest(chordQueryMsg)
		if err != nil {
			return xerrors.Errorf("[ChordRequestMessageCallback] failed to handle post request: %v", err)
		}
	default:
		return xerrors.Errorf("[ChordRequestMessageCallback] containing something else than ChordQuery enum: %d",
			chordQueryMsg.Query)

	}

	answ := types.ChordReplyMessage{RequestID: chordQueryMsg.RequestID, ChordNode: chordNode, Query: chordQueryMsg.Query,
		PostID: chordQueryMsg.Post.PostID}

	reply, err := n.TypesMsgToTranspMsg(answ)

	if err != nil {
		errmsg := fm.Sprintf("ChordRequestMessage error while TypesMsgToTranspMsg: %v\n", err)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	//n.Debug().Msgf("[ChordRequestMessageCallback] [%s] Reply to RequestID [%v] back to [%s]",
	//	myAddr, chordQueryMsg.RequestID, pkt.Header.Source)
	err = n.Unicast(pkt.Header.Source, reply)
	if err != nil {
		errmsg := fm.Sprintf("[ChordRequestMessageCallback] failed to Unicast: %d\n", err)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	return nil
}

func (n *node) ChordReplyMessageCallback(msg types.Message, pkt transport.Packet) error {
	myAddr := n.Socket.GetAddress()

	n.AddPeer(pkt.Header.Source)

	chordAnswerMsg, ok := msg.(*types.ChordReplyMessage)

	n.AddPeer(chordAnswerMsg.ChordNode.IPAddr)

	if !ok {
		errmsg := fm.Sprintf("Unable to convert message to ChordReplyMessage! %s\n", msg)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	n.Debug().Msgf("[ChordReplyMessageCallback] "+
		"[%s] handle chord answer msg [%s] of type [%d: %s] received from [%s]",
		myAddr, chordAnswerMsg.RequestID, chordAnswerMsg.Query,
		types.ChordQueryStr[chordAnswerMsg.Query], pkt.Header.Source)

	//n.Debug().Msgf("[ChordReplyMessageCallback] [%s] MSG Callback for AnswerChord [%s]",
	//	myAddr, chordAnswerMsg.RequestID)

	sentToChan := n.chanHandler.SendToChan(chordAnswerMsg.RequestID, chordAnswerMsg)
	if !sentToChan {
		n.Warn().Msgf("[ChordReplyMessageCallback] "+
			"[%s] No channel found for Request ID [%s]. Return nil", myAddr, chordAnswerMsg.RequestID)
		return nil
	}

	return nil
}

func (n *node) PoSChallengeRequestMessageCallback(msg types.Message, pkt transport.Packet) error {
	challengeRequestMsg, ok := msg.(*types.PoSChallengeRequestMessage)

	if !ok {
		errmsg := fm.Sprintf("Unable to convert message to PoSChallengeRequestMsg! %s\n", msg)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	// Solve the challenge and send back a reply.
	challengeReplyMsg, err := n.posModule.SolveChallenge(challengeRequestMsg)
	if err != nil {
		return logAndReturnError(n.Logger, err)
	}

	n.Debug().Msgf("[%s] Sending challenge reply", n.Socket.GetAddress())

	return n.sendChallengeReply(challengeReplyMsg, pkt.Header.Source)
}

func (n *node) PoSChallengeReplyMessageCallback(msg types.Message, pkt transport.Packet) error {
	challengeReplyMsg, ok := msg.(*types.PoSChallengeReplyMessage)

	if !ok {
		errmsg := fm.Sprintf("Unable to convert message to PoSChallengeReplyMessage! %s\n", msg)
		return logAndReturnError(n.Logger, &PError{message: errmsg})
	}

	n.chanHandler.SendToChan(challengeReplyMsg.RequestID, challengeReplyMsg)

	return nil
}

func (n *node) PostRequestMessageCallback(msg types.Message, pkt transport.Packet) error {
	myAddr := n.Socket.GetAddress()

	n.AddPeer(pkt.Header.Source)

	postMsg, ok := msg.(*types.PostRequestMessage)
	if !ok {
		return xerrors.Errorf("[PostRequestMessageCallback] "+
			"Unable to convert message to PostRequestMessage! %s\n", msg)
	}

	n.Debug().Msgf("[PostRequestMessageCallback] "+
		"[%s] PostRequestMessageCallback with RequestID [%s] received from [%s]",
		myAddr, postMsg.RequestID, pkt.Header.Source)

	// retrieve message from local storage
	retrievedPostBytes := n.Storage.GetPostStore().Get(postMsg.PostID)

	n.Debug().Msgf("[PostRequestMessageCallback] "+
		"[%s] retrievedPostBytes: [%s]", myAddr, retrievedPostBytes)

	retrievedPost := types.Post{}
	err := json.Unmarshal(retrievedPostBytes, &retrievedPost)
	if err != nil {
		return xerrors.Errorf("[%s] [PostRequestMessageCallback] failed to unmarshal post: %v", myAddr, err)
	}

	// send ack back to source
	postReplyMsg := types.PostReplyMessage{RequestID: postMsg.RequestID, Post: retrievedPost}

	transpPostReplyMsg, err := n.TypesMsgToTranspMsg(postReplyMsg)
	if err != nil {
		return logAndReturnError(n.Logger, err)
	}
	err = n.Unicast(pkt.Header.Source, transpPostReplyMsg)
	if err != nil {
		return xerrors.Errorf("[PostRequestMessageCallback] failed to Unicast postReplyMsg [%s]: %v",
			postReplyMsg.RequestID, err)
	}

	n.Debug().Msgf("[PostRequestMessageCallback] [%s] Sent postReplyMsg with RequestID [%s] back to [%s]",
		myAddr, postReplyMsg.RequestID, pkt.Header.Source)

	return nil
}

func (n *node) PostReplyMessageCallback(msg types.Message, pkt transport.Packet) error {
	myAddr := n.Socket.GetAddress()

	n.AddPeer(pkt.Header.Source)

	postReplyMsg, ok := msg.(*types.PostReplyMessage)
	if !ok {
		return xerrors.Errorf("[PostReplyMessageCallback] Unable to convert message to PostReplyMessage! %s\n", msg)
	}

	postIDBigInt, err := utils.StrToBigInt(postReplyMsg.Post.PostID)
	if err != nil {
		return xerrors.Errorf("[PostReplyMessageCallback] failed to convert postID [%s] to bigInt: %v",
			postReplyMsg.Post.PostID, err)
	}
	n.Debug().Msgf(
		"[PostReplyMessageCallback] [%s] PostReplyMessage [%s] received from [%s] for PostID [%s]", myAddr,
		postReplyMsg.RequestID, pkt.Header.Source, postIDBigInt)

	sentToChan := n.chanHandler.SendToChan(postReplyMsg.RequestID, postReplyMsg)
	if !sentToChan {
		n.Warn().Msgf("[PostReplyMessageCallback] [%s] No channel found for Request ID [%s]. Return nil",
			myAddr, postReplyMsg.RequestID)
	}

	return nil
}

func (n *node) CatalogRequestMessageCallback(msg types.Message, pkt transport.Packet) error {
	myAddr := n.Socket.GetAddress()

	n.AddPeer(pkt.Header.Source)

	catalogRequestMsg, ok := msg.(*types.CatalogRequestMessage)
	if !ok {
		return xerrors.Errorf("[CatalogRequestMessageCallback] "+
			"Unable to convert message to CatalogRequestMessage! %s\n", msg)
	}

	n.Debug().Msgf("[CatalogRequestMessageCallback] [%s] CatalogRequestMsg with RequestID [%s]"+
		" received from [%s]", myAddr, catalogRequestMsg.RequestID, pkt.Header.Source)

	// retrieve whole catalog from local storage

	catalog := make(map[string][]byte)
	if n.Storage.GetPostCatalogStore().Len() == 0 {
		n.Debug().Msgf("[CatalogRequestMessageCallback] [%s] PostCatalogStore is empty", myAddr)
	}
	n.Storage.GetPostCatalogStore().ForEach(func(authorID string, postIDsByAuthorBytes []byte) bool {
		catalog[authorID] = postIDsByAuthorBytes
		return true // Returning true continues the iteration
	})

	// send ack back to source
	catalogReplyMsg := types.CatalogReplyMessage{RequestID: catalogRequestMsg.RequestID, Catalog: catalog}

	transpPostReplyMsg, err := n.TypesMsgToTranspMsg(catalogReplyMsg)
	if err != nil {
		return xerrors.Errorf("[CatalogRequestMessageCallback] failed to marshal catalogReplyMsg: %v", err)
	}
	err = n.Unicast(pkt.Header.Source, transpPostReplyMsg)
	if err != nil {
		return xerrors.Errorf("[CatalogRequestMessageCallback] failed to Unicast catalogReplyMsg [%s]: %v",
			catalogReplyMsg.RequestID, err)
	}

	n.Debug().Msgf("[CatalogRequestMessageCallback] [%s] Sent catalogReplyMsg with RequestID [%s] back to [%s]",
		myAddr, catalogReplyMsg.RequestID, pkt.Header.Source)

	return nil
}

func (n *node) CatalogReplyMessageCallback(msg types.Message, pkt transport.Packet) error {
	myAddr := n.Socket.GetAddress()

	n.AddPeer(pkt.Header.Source)

	catalogReplyMsg, ok := msg.(*types.CatalogReplyMessage)
	if !ok {
		return xerrors.Errorf("[CatalogReplyMessageCallback] Unable to convert message to CatalogReplyMsg! %s\n", msg)
	}

	n.Debug().Msgf("[CatalogReplyMessageCallback] [%s] CatalogReplyMessage [%s] received from [%s]", myAddr,
		catalogReplyMsg.RequestID, pkt.Header.Source)

	sentToChan := n.chanHandler.SendToChan(catalogReplyMsg.RequestID, catalogReplyMsg)
	if !sentToChan {
		n.Warn().Msgf("[CatalogReplyMessageCallback] [%s] No channel found for Request ID [%s]. Return nil",
			myAddr, catalogReplyMsg.RequestID)
	}
	return nil
}
