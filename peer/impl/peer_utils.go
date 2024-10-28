package impl

import (
	"encoding/json"
	"errors"
	fm "fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/rs/xid"
	"go.dedis.ch/cs438/n2ccrypto"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"go.dedis.ch/cs438/utils"
	"golang.org/x/xerrors"
)

const M = KeyLen * 8 // nb of bits in the chord identifier space// todo: define it in conf

// Send functions
func (n *node) sendPacketConcurrently(pkt transport.Packet, dest string) {
	//n.Add(1)

	go func() {
		//defer n.Done()

		// Avoid overflow in uint if TTL is already 0.
		if pkt.Header.TTL > 0 {
			pkt.Header.TTL--
		} else {
			pkt.Header.TTL = 0
		}

		if pkt.Header.TTL > 0 {
			err := n.Socket.Send(dest, pkt, time.Second*1)
			if err != nil {
				_ = logAndReturnError(n.Logger, err)
			}
		}
	}()
}

func (n *node) sendPacket(pkt transport.Packet, dest string) error {
	if pkt.Header.TTL > 0 {
		pkt.Header.TTL--
	} else {
		pkt.Header.TTL = 0
	}

	if pkt.Header.TTL > 0 {
		err := n.Socket.Send(dest, pkt, time.Second*1)
		if err != nil {
			return err
		}
		return nil
	}

	return &PError{message: "Packet reached TTL 0. Dropping the packet."}
}

func (n *node) sendACK(pkt transport.Packet, dest string) error {
	ackMessage := types.AckMessage{AckedPacketID: pkt.Header.PacketID, Status: n.getStatusMessage()}
	ackMsg, err := n.MessageRegistry.MarshalMessage(&ackMessage)
	if err != nil {
		return err
	}

	ackHdr := transport.NewHeader(n.Socket.GetAddress(), n.Socket.GetAddress(), dest, 0)
	n.sendPacketConcurrently(transport.Packet{Header: &ackHdr, Msg: &ackMsg}, dest)

	return nil
}

func (n *node) sendStatusMessage(statusMsg types.StatusMessage, dest string) error {
	statMsg, err := n.MessageRegistry.MarshalMessage(&statusMsg)
	if err != nil {
		return err
	}

	hdr := transport.NewHeader(n.Socket.GetAddress(), n.Socket.GetAddress(), dest, 0)
	n.sendPacketConcurrently(transport.Packet{Header: &hdr, Msg: &statMsg}, dest)

	return nil
}

func (n *node) sendMissingRumors(missingRumors []types.Rumor, dest string) error {
	rumorsMessage := types.RumorsMessage{Rumors: missingRumors}

	rumorsMsg, err := n.MessageRegistry.MarshalMessage(&rumorsMessage)
	if err != nil {
		return err
	}

	rumorHdr := transport.NewHeader(n.Socket.GetAddress(), n.Socket.GetAddress(), dest, 0)
	n.sendPacketConcurrently(transport.Packet{Header: &rumorHdr, Msg: &rumorsMsg}, dest)

	return nil
}

// Send a message to a random neighbor other than the neighbors in "exclude".
func (n *node) sendPacketToRandomNeighbor(pkt transport.Packet, neighbors []string, excludeList []string) error {
	if len(neighbors) != 0 {
		excludedNeighbors := 0
		for _, neighbor := range neighbors {
			if containsElement(excludeList, neighbor) {
				excludedNeighbors++
			}
		}
		if excludedNeighbors == len(neighbors) {
			return NoNeighborsError{}
		}

		rgen := rand.New(rand.NewSource(time.Now().UnixNano()))
		neighbor := neighbors[rgen.Int31n(int32(len(neighbors)))]
		for containsElement(excludeList, neighbor) {
			neighbor = neighbors[rgen.Int31n(int32(len(neighbors)))]
		}

		pkt.Header.Destination = neighbor
		return n.sendPacket(pkt, neighbor)
	}

	return NoNeighborsError{}
}

func (n *node) sendRumorsAndWaitForAck(rumorsPkt transport.Packet, excludeList []string) {
	// Create and add a channel associated to that packetID.
	ackChannel := make(chan any)
	mapAddRW(n.SyncTable, rumorsPkt.Header.PacketID, ackChannel, &n.syncTable.RWMutex)

	if n.checkStopCalled() {
		return
	}

	n.Add(1)
	go func() {
		defer n.Done()

		ackReceived := false
		neighbors := n.getNeighbors("")
		// While no ACK was received and there are still possible neighbors.
		for !ackReceived && len(neighbors) > len(excludeList) {
			if n.checkStopCalled() {
				return
			}

			err := n.sendPacketToRandomNeighbor(rumorsPkt, neighbors, excludeList)
			if err != nil {
				if errors.Is(err, NoNeighborsError{}) {
					// Could not find any neighbor, exit gracefully.
					break
				}

				// Send failed for some reason (maybe destination is unreachable).
				// Exclude that destination and try again.
				_ = logAndReturnError(n.Logger, err)
				excludeList = append(excludeList, rumorsPkt.Header.Destination)
				continue
			}

			if n.AckTimeout == 0 {
				break
			}

			select {
			case <-time.After(n.AckTimeout):
				// Timed out. Add this neighbor to exclude list and try again.
				excludeList = append(excludeList, rumorsPkt.Header.Destination)
			case <-ackChannel:
				ackReceived = true
			}
		}
	}()
}

// Sends a rumorsMessage and waits for ACK.
func (n *node) sendRumorsMessage(rmsg types.RumorsMessage, excludeList []string) error {
	rumorsMsg, err := n.MessageRegistry.MarshalMessage(&rmsg)
	if err != nil {
		return logAndReturnError(n.Logger, err)
	}

	origin := n.Socket.GetAddress()
	relayedBy := n.Socket.GetAddress()
	// Create packet w/o destination for now, just to use its packet ID.
	hdr := transport.NewHeader(origin, relayedBy, "", 0)
	rumorsPkt := transport.Packet{Header: &hdr, Msg: &rumorsMsg}

	n.sendRumorsAndWaitForAck(rumorsPkt, excludeList)

	return nil
}

func (n *node) sendAntiEntropyStatusMessage() {
	// Send StatusMessage to a random neighbor.
	statusMessage := n.getStatusMessage()
	statusMsg, err := n.MessageRegistry.MarshalMessage(&statusMessage)
	if err != nil {
		_ = logAndReturnError(n.Logger, err)
	}

	hdr := transport.NewHeader(n.Socket.GetAddress(), n.Socket.GetAddress(), "", 0)
	pkt := transport.Packet{Header: &hdr, Msg: &statusMsg}
	if err = n.sendPacketToRandomNeighbor(pkt, n.getNeighbors(""), make([]string, 0)); err != nil {
		if !errors.Is(err, NoNeighborsError{}) {
			_ = logAndReturnError(n.Logger, err)
			return
		}
	}
}

func reputationToDifficultFactor(reputation int) int {
	if reputation < 0 {
		return -reputation
	}

	return reputation
}

func (n *node) sendChallengeRequest(peer string, peerNodeID [KeyLen]byte) (string, error) {
	// Send the same challenge if present on the map.
	challengeRequestMsg, exists := mapRead(n.Challenges, peer, &n.challenges.Mutex)
	if !exists {
		// Does not exist, create a new challenge.
		challengeRequestMsgNew, err := n.posModule.GenerateChallenge(peerNodeID[:], []byte(xid.New().String()[:]),
			uint(reputationToDifficultFactor(n.getReputation(peer)-1)))
		if err != nil {
			return challengeRequestMsgNew.RequestID, err
		}
		mapAdd(n.Challenges, peer, challengeRequestMsgNew, &n.challenges.Mutex)
		challengeRequestMsg = challengeRequestMsgNew

		n.Debug().Msgf("[%s] Sending challenge to chordID: %v", n.Socket.GetAddress(),
			utils.BytesToBigInt(peerNodeID))
	}

	challengeRequestTransMsg, err := n.MessageRegistry.MarshalMessage(challengeRequestMsg)
	if err != nil {
		return challengeRequestMsg.RequestID, err
	}

	err = n.Unicast(peer, challengeRequestTransMsg)
	if err != nil {
		return challengeRequestMsg.RequestID, err
	}

	return challengeRequestMsg.RequestID, nil
}

func (n *node) sendChallengeReply(challengeReplyMsg *types.PoSChallengeReplyMessage, peer string) error {
	challengeReplyTransMsg, err := n.MessageRegistry.MarshalMessage(challengeReplyMsg)
	if err != nil {
		return err
	}

	hdr := transport.NewHeader(n.Socket.GetAddress(), n.Socket.GetAddress(), peer, 0)
	n.sendPacketConcurrently(transport.Packet{Header: &hdr, Msg: &challengeReplyTransMsg}, peer)

	return nil
}

func (n *node) forwardStatusMessage(statusMessage *types.StatusMessage, remoteAddr string) error {
	paddr := n.Socket.GetAddress()
	rgen := rand.New(rand.NewSource(time.Now().UnixNano()))
	if rgen.Float64() < n.ContinueMongering {

		pStatusMsg, err := n.MessageRegistry.MarshalMessage(statusMessage)
		if err != nil {
			return err
		}

		statusHdr := transport.NewHeader(paddr, paddr, "", 0)
		statusPkt := transport.Packet{Header: &statusHdr, Msg: &pStatusMsg}
		// Send status message to random neighbor other than the source.
		return n.sendPacketToRandomNeighbor(statusPkt, n.getNeighbors(remoteAddr), make([]string, 0))
	}

	return nil
}

func (n *node) statusMessagesEqual(localStatusMessage, remoteStatusMessage types.StatusMessage) bool {
	if len(localStatusMessage) != len(remoteStatusMessage) {
		return false
	}

	for paddr, pseq := range localStatusMessage {
		rseq, exists := remoteStatusMessage[paddr]
		if !exists || pseq != rseq {
			return false
		}
	}
	return true
}

func (n *node) checkStatusMessages(localStatusMessage, remoteStatusMessage types.StatusMessage) (bool, bool) {
	n.rumorTable.Lock()
	defer n.rumorTable.Unlock()

	sendRumors := false
	sendStatus := false
	for paddr, pseq := range localStatusMessage {
		rseq, exists := remoteStatusMessage[paddr]
		if !exists || pseq > rseq {
			// This peer has rumors remote peer does not have.
			sendRumors = true
		} else if pseq < rseq {
			// Remote peer has rumors this peer does not have,
			sendStatus = true
		}
		if sendRumors && sendStatus {
			break
		}
	}

	// Remote peer has rumors this peer does not have,
	for raddr := range remoteStatusMessage {
		_, exists := localStatusMessage[raddr]
		if !exists {
			sendStatus = true
			break
		}
	}

	return sendRumors, sendStatus
}

func (n *node) createDataRequest(metahash string) (chan any, transport.Message, error) {
	dataRequestMessage := types.DataRequestMessage{RequestID: xid.New().String(), Key: metahash, ChordID: n.chordID}
	dataRequestChannel := make(chan any)
	mapAddRW(n.SyncTable, dataRequestMessage.RequestID, dataRequestChannel, &n.syncTable.RWMutex)

	dataRequestMsg, err := n.MessageRegistry.MarshalMessage(&dataRequestMessage)
	if err != nil {
		return nil, dataRequestMsg, err
	}

	return dataRequestChannel, dataRequestMsg, err
}

func (n *node) sendDataRequest(remotePeer string, metahash string) error {
	dataReplyReceived := false
	waitDuration := n.BackoffDataRequest.Initial
	var i uint = 1
	for ; i <= n.BackoffDataRequest.Retry; i++ {
		// Send dataRequestMessage to remotePeer. It has to be a request with a new ID otherwise peer will ignore it.
		dataRequestChannel, dataRequestMsg, err := n.createDataRequest(metahash)
		if err != nil {
			return err
		}
		err = n.Unicast(remotePeer, dataRequestMsg)
		if err != nil {
			return err
		}

		select {
		case <-dataRequestChannel:
			dataReplyReceived = true
		case <-time.After(waitDuration):
			break
		case <-n.stopCalledChan:
			return &PError{message: "Received stop signal while trying to get chunk from remote peers!"}
		}

		if dataReplyReceived {
			break
		}

		waitDuration *= time.Duration(n.BackoffDataRequest.Factor)
	}

	if !dataReplyReceived {
		n.removePeerFromCatalog(metahash, remotePeer)
		return &PError{message: "Did not receive data reply for data request!"}
	}

	return nil
}

func (n *node) sendSearchRequest(dest string, searchRequestMessage types.SearchRequestMessage) error {
	hdr := transport.NewHeader(n.Socket.GetAddress(), n.Socket.GetAddress(), dest, 0)
	msg, err := n.MessageRegistry.MarshalMessage(searchRequestMessage)
	if err != nil {
		return err
	}
	pkt := transport.Packet{Header: &hdr, Msg: &msg}

	n.sendPacketConcurrently(pkt, dest)

	return nil
}

func (n *node) getStatusMessage() types.StatusMessage {
	n.rumorTable.Lock()
	defer n.rumorTable.Unlock()

	// Create StatusMessage
	statusMessage := types.StatusMessage{}
	for k, v := range n.RumorTable {
		statusMessage[k] = uint(len(v))
	}

	return statusMessage
}

func (n *node) getNeighbors(exclude string) []string {
	n.routingTable.Lock()
	defer n.routingTable.Unlock()

	neighbors := make([]string, 0)
	for origin, relay := range n.RoutingTable {
		if origin == n.Socket.GetAddress() {
			continue
		}
		if origin == relay {
			if origin != exclude {
				neighbors = append(neighbors, origin)
			}
		}
	}

	return neighbors
}

func (n *node) getMissingRumors(pstatusMsg types.StatusMessage, remoteStatusMsg types.StatusMessage) []types.Rumor {
	var missingRumors []types.Rumor

	for addr, seqno := range pstatusMsg {
		if rseqno, exists := remoteStatusMsg[addr]; !exists {
			// Remote peer does not have any rumors from this peer.
			lastRumors, _ := mapRead(n.RumorTable, addr, &n.rumorTable.Mutex)
			missingRumors = append(missingRumors, lastRumors...)
		} else {
			if seqno > rseqno {
				// Remote peer has only some rumors of this addr.
				lastRumors, _ := mapRead(n.RumorTable, addr, &n.rumorTable.Mutex)
				missingRumors = append(missingRumors, lastRumors[rseqno:]...)
			}
		}
	}

	// sort in increasing sequence number order.
	sort.Slice(missingRumors, func(i, j int) bool {
		return missingRumors[i].Sequence < missingRumors[j].Sequence
	})

	return missingRumors
}

func (n *node) processPacket(pkt transport.Packet) {
	// If we received a message but the destination is not our address, we should relay the message.
	var err error
	if pkt.Header.Destination != n.Socket.GetAddress() {
		relay, exists := mapRead(n.RoutingTable, pkt.Header.Destination, &n.routingTable.Mutex)
		if exists {
			hdr := transport.NewHeader(pkt.Header.Source, n.Socket.GetAddress(), pkt.Header.Destination, 0)
			// Relay is the destination here.
			n.sendPacketConcurrently(transport.Packet{Header: &hdr, Msg: pkt.Msg}, relay)
		} else {
			// Log error here.
			msg := fm.Sprintf("(%s) Cannot find route to destination (%s)", n.Socket.GetAddress(), pkt.Header.Destination)
			err = &PError{message: msg}
		}
	} else {
		//n.Debug().Msgf("[processPacket] [%s] Received message %s", n.Socket.GetAddress(), pkt.Msg.Type)
		err = n.MessageRegistry.ProcessPacket(pkt)
	}

	if err != nil {
		_ = logAndReturnError(n.Logger, err)
	}
}

func (n *node) processPacketWithValidate(pkt transport.Packet) {
	// Receiving thread should keep processing challenge messages.
	if !shouldValidateMsg(pkt.Msg) {
		go n.processPacket(pkt)
	} else {
		n.Add(1)
		go func() {
			defer n.Done()
			validated := n.validatePeer(pkt.Header.Source, pkt.Msg)
			if validated {
				go n.processPacket(pkt)
			}
		}()
	}
}

func (n *node) antiEntropy() {
	if n.checkStopCalled() {
		return
	}

	n.Add(1)
	n.sendAntiEntropyStatusMessage()

	go func() {
		defer n.Done()

		for {

			select {
			case <-time.After(n.AntiEntropyInterval):
				n.sendAntiEntropyStatusMessage()
			case <-n.stopCalledChan:
				return
			}
		}
	}()
}

func (n *node) broadcastEmptyMessage() error {
	emptyMessage := types.EmptyMessage{}
	emptyMsg, err := n.MessageRegistry.MarshalMessage(&emptyMessage)
	if err != nil {
		return err
	}

	return n.Broadcast(emptyMsg)
}

func (n *node) heartBeat() {
	n.Add(1)

	if err := n.broadcastEmptyMessage(); err != nil {
		return
	}

	go func() {
		defer n.Done()

		for {
			select {
			case <-time.After(n.HeartbeatInterval):
				if err := n.broadcastEmptyMessage(); err != nil {
					return
				}
			case <-n.stopCalledChan:
				return
			}

		}
	}()
}

func (n *node) getLastRumorSeqno(origin string) uint {
	// Get the last sequence number.
	n.rumorTable.Lock()
	defer n.rumorTable.Unlock()

	rumors, exists := n.RumorTable[origin]
	var lastRumorSeqno uint
	if !exists || len(rumors) == 0 {
		// No rumor from this origin yet.
		lastRumorSeqno = 0
	} else {
		lastRumorSeqno = rumors[len(rumors)-1].Sequence
	}

	return lastRumorSeqno
}

func (n *node) updateRumorTable(rumor types.Rumor) {
	n.rumorTable.Lock()
	defer n.rumorTable.Unlock()

	rumors := n.RumorTable[rumor.Origin]
	rumors = append(rumors, rumor)
	n.RumorTable[rumor.Origin] = rumors
}

func (n *node) createRumor(msg transport.Message) types.Rumor {
	n.rumorTable.Lock()
	defer n.rumorTable.Unlock()

	addr := n.Socket.GetAddress()
	lastRumors := n.RumorTable[addr]
	rumor := types.Rumor{Origin: addr, Sequence: uint(len(lastRumors) + 1), Msg: &msg}
	lastRumors = append(lastRumors, rumor)
	n.RumorTable[addr] = lastRumors

	return rumor
}

func (n *node) getRemotePeers(metahash string) []string {
	// Check catalog.
	n.catalog.Mutex.Lock()
	defer n.catalog.Mutex.Unlock()

	catalogMap, exists := n.Catalog[metahash]
	if !exists || len(catalogMap) == 0 {
		// We did not find the file locally, and we do not know anyone who might have it. Return error.
		return nil
	}

	remotePeers := make([]string, len(catalogMap))
	i := 0
	for k := range catalogMap {
		remotePeers[i] = k
		i++
	}

	return remotePeers
}

func (n *node) getChunkFromRemotePeer(metahash string) (string, error) {
	remotePeers := n.getRemotePeers(metahash)
	if remotePeers == nil {
		return "", &PError{message: "Could not find remote peers that contain this chunk!"}
	}

	rgen := rand.New(rand.NewSource(time.Now().UnixNano()))
	choice := rgen.Int31n(int32(len(remotePeers)))
	remotePeer := remotePeers[choice]

	err := n.sendDataRequest(remotePeer, metahash)
	if err != nil {
		return remotePeer, err
	}

	return remotePeer, nil
}

func (n *node) removePeerFromCatalog(metahash, remotePeer string) {
	n.catalog.Lock()
	defer n.catalog.Unlock()

	catalogMap, exists := n.Catalog[metahash]
	if exists {
		delete(catalogMap, remotePeer)
		if len(catalogMap) == 0 {
			delete(n.Catalog, metahash)
		}
	}
}

func (n *node) updateNamingStorage(searchReplyMsg *types.SearchReplyMessage, source string) {
	namingStorage := n.Storage.GetNamingStore()
	for _, fileInfo := range searchReplyMsg.Responses {
		n.UpdateCatalog(fileInfo.Metahash, source)
		namingStorage.Set(fileInfo.Name, []byte(fileInfo.Metahash))
		for _, metaHash := range fileInfo.Chunks {
			if metaHash != nil {
				metaHashStr := string(metaHash)
				n.UpdateCatalog(metaHashStr, source)
			}
		}
	}
}

func (n *node) updateSearchReplies(searchReplyMsg *types.SearchReplyMessage) {
	n.searchReplies.Lock()
	defer n.searchReplies.Unlock()

	var exists bool
	searchRepliesMap, exists := n.SearchReplies[searchReplyMsg.RequestID]
	if !exists {
		searchRepliesMap = make(map[string]bool)
	}

	for _, fileInfo := range searchReplyMsg.Responses {
		_, exists := searchRepliesMap[fileInfo.Name]
		isFullFile := hasAllChunks(fileInfo.Chunks)
		if !exists {
			searchRepliesMap[fileInfo.Name] = isFullFile
		} else {
			if isFullFile {
				// Replace only if it is a full file.
				searchRepliesMap[fileInfo.Name] = isFullFile
			}
		}

		if isFullFile {
			// Found a full file. Close the channel associated with it.
			var requestChannel chan any
			requestChannel, exists = mapReadRW(n.SyncTable, searchReplyMsg.RequestID, &n.syncTable.RWMutex)
			if exists {
				close(requestChannel)
				mapDeleteRW(n.SyncTable, searchReplyMsg.RequestID, &n.syncTable.RWMutex)
			}
		}
	}

	n.SearchReplies[searchReplyMsg.RequestID] = searchRepliesMap
}

func (n *node) getMetaHashValue(metahash string) ([]byte, error) {
	// First check if peer has the file locally.
	blobStorage := n.Storage.GetDataBlobStore()
	metaHashValue := blobStorage.Get(metahash)
	if metaHashValue == nil {
		remotePeer, err := n.getChunkFromRemotePeer(metahash)
		if err != nil {
			return nil, err
		}

		// Successfully stored the data. Check for its integrity.
		chunk := blobStorage.Get(metahash)
		if !checkDataIntegrity(chunk, metahash) {
			blobStorage.Delete(metahash)
			n.removePeerFromCatalog(metahash, remotePeer)
			return chunk, &PError{message: "Received wrong data from remote peer!"}
		}

		return chunk, nil
	}

	return metaHashValue, nil
}

func (n *node) searchAllLocal(reg regexp.Regexp) []string {
	namingStore := n.Storage.GetNamingStore()

	names := make([]string, 0)
	namingStore.ForEach(func(key string, val []byte) bool {
		if reg.MatchString(key) {
			names = append(names, key)
		}
		return true
	})

	return names
}

func (n *node) sendSearchRequests(reg regexp.Regexp, budget uint, fullSearch bool) (string, chan any, error) {
	neighbors := n.getNeighbors("")

	requestID := xid.New().String()

	var requestChannel chan any
	if fullSearch {
		requestChannel = make(chan any)
		mapAddRW(n.SyncTable, requestID, requestChannel, &n.syncTable.RWMutex)
	}

	var neighborsToSend []string
	var budgets []uint
	if budget <= uint(len(neighbors)) {
		// Pick budget random neighbors and send SearchRequestMessages.
		rgen := rand.New(rand.NewSource(time.Now().UnixNano()))
		neighborIndices := rgen.Perm(len(neighbors))
		neighborsToSend = make([]string, 0)
		budgets = make([]uint, len(neighborIndices))
		for i, ni := range neighborIndices {
			neighborsToSend = append(neighborsToSend, neighbors[ni])
			budgets[i] = 1
		}
	} else if len(neighbors) > 0 {
		// Evenly distribute budget between neighbors.
		budgets = distributeBudget(budget, uint(len(neighbors)))
		neighborsToSend = neighbors
	}

	if len(neighborsToSend) > 0 {
		for i, neighbor := range neighborsToSend {
			if budgets[i] > 0 {
				searchRequestMessage := types.SearchRequestMessage{RequestID: requestID, Origin: n.Socket.GetAddress(),
					Pattern: reg.String(), Budget: budgets[i], ChordID: n.chordID}
				err := n.sendSearchRequest(neighbor, searchRequestMessage)
				if err != nil {
					return requestID, requestChannel, err
				}
			}
		}

	}

	return requestID, requestChannel, nil
}

func (n *node) searchAllRemote(reg regexp.Regexp, budget uint, timeout time.Duration) ([]string, error) {
	requestID, _, err := n.sendSearchRequests(reg, budget, false)
	if err != nil {
		return nil, err
	}

	return n.gatherSearchRequestReplies(requestID, timeout)
}

func (n *node) gatherSearchRequestReplies(requestID string, timeout time.Duration) ([]string, error) {
	names := make([]string, 0)

	select {
	case <-time.After(timeout):
		break
	case <-n.stopCalledChan:
		n.Warn().Msgf("(%s) Stop called while waiting to gather search replies", n.Socket.GetAddress())
		return names, nil
	}

	n.searchReplies.Lock()
	defer n.searchReplies.Unlock()
	replies, exists := n.SearchReplies[requestID]
	if !exists {
		return names, nil
	}
	for fileName := range replies {
		if !containsElement(names, fileName) {
			names = append(names, fileName)
		}
	}

	return names, nil
}

func (n *node) getSearchRequestResponses(reg regexp.Regexp) []types.FileInfo {
	responses := make([]types.FileInfo, 0)
	blobStorage := n.Storage.GetDataBlobStore()
	namingStore := n.Storage.GetNamingStore()
	namingStore.ForEach(func(key string, val []byte) bool {
		if reg.MatchString(key) {
			valstr := string(val)
			chunk := blobStorage.Get(valstr)
			if chunk != nil {
				metaHashes := strings.Split(string(chunk), peer.MetafileSep)
				fileInfoChunks := make([][]byte, 0)
				for _, metaHash := range metaHashes {
					if blobStorage.Get(metaHash) != nil {
						fileInfoChunks = append(fileInfoChunks, []byte(metaHash))
					} else {
						fileInfoChunks = append(fileInfoChunks, nil)
					}
				}
				responses = append(responses, types.FileInfo{Name: key, Metahash: valstr, Chunks: fileInfoChunks})
			}
		}
		return true
	})

	return responses
}

func (n *node) searchFirstLocal(pattern regexp.Regexp) string {
	namingStore := n.Storage.GetNamingStore()
	blobStorage := n.Storage.GetDataBlobStore()

	name := ""
	namingStore.ForEach(func(key string, val []byte) bool {
		if pattern.MatchString(key) {
			var stop bool
			name, stop = getNameFirst(blobStorage, key, val)
			return stop
		}
		return true
	})

	return name
}

func (n *node) fullFileFound(requestID string, requestChannel chan any, timeout time.Duration) string {
	select {
	case <-requestChannel:
		break
	case <-time.After(timeout):
		break
	case <-n.stopCalledChan:
		n.Warn().Msgf("(%s) Stop called while waiting to receive full file.", n.Socket.GetAddress())
		return ""
	}

	n.searchReplies.Lock()
	defer n.searchReplies.Unlock()
	replies, exists := n.SearchReplies[requestID]
	if !exists {
		return ""
	}

	for fileName, isFullFile := range replies {
		if isFullFile {
			return fileName
		}
	}

	return ""
}

func (n *node) searchFirstRemote(pattern regexp.Regexp, conf peer.ExpandingRing) (string, error) {
	var i uint
	budget := conf.Initial
	for ; i < conf.Retry; i++ {
		requestID, requestChannel, err := n.sendSearchRequests(pattern, budget, true)
		if err != nil {
			return "", err
		}

		name := n.fullFileFound(requestID, requestChannel, conf.Timeout)
		if name != "" {
			return name, nil
		}
		// Retry
		budget *= conf.Factor
	}

	return "", nil
}

func (n *node) forwardSearchToNeighbors(searchRequestMsg *types.SearchRequestMessage, neighbors []string) error {
	if len(neighbors) > 0 {
		budgets := distributeBudget(searchRequestMsg.Budget, uint(len(neighbors)))
		for i, neighbor := range neighbors {
			if budgets[i] > 0 {
				searchRequestFwd := types.SearchRequestMessage{RequestID: searchRequestMsg.RequestID,
					Origin: searchRequestMsg.Origin, Pattern: searchRequestMsg.Pattern, Budget: budgets[i]}
				err := n.sendSearchRequest(neighbor, searchRequestFwd)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (n *node) checkStopCalled() bool {
	select {
	case <-n.stopCalledChan:
		return true
	default:
	}

	return false
}

// CreatePost from content
func (n *node) CreatePost(content []byte) types.Post {
	// assign a Chord identifier to the post
	createdAt := time.Now()
	authorID := n.GetNodeID()
	postChordID := createPostID(authorID, createdAt)

	return types.Post{
		PostID:    utils.EncodeChordIDToStr(postChordID),
		AuthorID:  utils.EncodeChordIDToStr(authorID),
		Content:   content,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Likes:     0,
		Comments:  nil,
	}
}

func (n *node) storePrivatePostInCatalog(post types.Post) error {
	postIDsByAuthorBytes := n.Storage.GetPostCatalogStore().Get(post.AuthorID)
	var postIDsByAuthor map[string]struct{}

	if postIDsByAuthorBytes == nil {
		postIDsByAuthor = make(map[string]struct{}, 0)
	} else {
		err := json.Unmarshal(postIDsByAuthorBytes, &postIDsByAuthor)
		if err != nil {
			return logAndReturnError(n.Logger, xerrors.Errorf("%v", err))
		}
	}

	// add a post to the catalog (bag or unique postIDs)  of an author
	postIDsByAuthor[post.PostID] = struct{}{}

	newPostIDsByAuthorBytes, err := json.Marshal(postIDsByAuthor)
	if err != nil {
		return logAndReturnError(n.Logger, xerrors.Errorf("%v", err))
	}

	// store the updated catalog of an author to the PostCatalog
	n.Storage.GetPostCatalogStore().Set(post.AuthorID, newPostIDsByAuthorBytes)
	return nil
}

func (n *node) storePublicPostInCatalog(post types.Post) error {
	postIDsByAuthorBytes := n.Storage.GetPostCatalogStore().Get(peer.PublicPostPk)
	var postIDsByAuthor map[string]struct{}
	if postIDsByAuthorBytes == nil {
		postIDsByAuthor = make(map[string]struct{}, 0)
	} else {
		err := json.Unmarshal(postIDsByAuthorBytes, &postIDsByAuthor)
		if err != nil {
			return logAndReturnError(n.Logger, xerrors.Errorf("%v", err))
		}
	}

	// add a post to the catalog (bag or unique postIDs)  of an author
	postIDsByAuthor[post.PostID] = struct{}{}

	newPostIDsByAuthorBytes, err := json.Marshal(postIDsByAuthor)
	if err != nil {
		return logAndReturnError(n.Logger, xerrors.Errorf("%v", err))
	}

	n.Storage.GetPostCatalogStore().Set(peer.PublicPostPk, newPostIDsByAuthorBytes)
	return nil
}

func (n *node) sendFollowRequestMessage(peer string) (string, error) {
	requestID := xid.New().String()
	followRequestMsg := types.FollowRequestMessage{RequestID: requestID, ChordID: n.chordID}
	followRequestTransMsg, err := n.MessageRegistry.MarshalMessage(&followRequestMsg)
	if err != nil {
		return requestID, err
	}

	return requestID, n.Unicast(peer, followRequestTransMsg)
}

func (n *node) sendFollowReplyMessage(followRequestMsg *types.FollowRequestMessage, source string) error {
	followReplyMsg := types.FollowReplyMessage{RequestID: followRequestMsg.RequestID, ChordID: n.chordID}
	followReplyTransMsg, err := n.MessageRegistry.MarshalMessage(&followReplyMsg)
	if err != nil {
		return err
	}

	return n.Unicast(source, followReplyTransMsg)
}

//func (n *node) broadcastBlockMessage() error {
//	blockMsg := types.BlockMessage{NewChordID: n.chordID}
//	blockMsgTransMsg, err := n.MessageRegistry.MarshalMessage(&blockMsg)
//	if err != nil {
//		return err
//	}
//
//	// Recipients are all followers except the one we are blocking (who is already removed from the map).
//	recipients := make(map[string]struct{})
//	followersList := n.GetFollowers()
//	for _, follower := range followersList {
//		recipients[follower] = struct{}{}
//	}
//
//	privateMsg := types.PrivateMessage{Recipients: recipients, Msg: &blockMsgTransMsg}
//	privateTransMsg, err := n.MessageRegistry.MarshalMessage(privateMsg)
//	if err != nil {
//		return err
//	}
//
//	return n.Broadcast(privateTransMsg)
//}

func (n *node) broadcastPostFlag(postID string) error {
	flagPostMsg := types.FlagPostMessage{PostID: postID, FlaggerID: n.GetNodeIDStr()}
	flagPostTransMsg, err := n.MessageRegistry.MarshalMessage(&flagPostMsg)
	if err != nil {
		return err
	}
	return n.Broadcast(flagPostTransMsg)
}

func (n *node) registerPostFlag(postID string, flaggingPeer string) {
	// add a flag to a post
	n.flaggedPosts.Lock()
	defer n.flaggedPosts.Unlock()
	flaggers, exists := n.flaggedPosts.FlaggedPosts[postID]
	if !exists {
		flaggers = make(map[string]struct{})
	}
	flaggers[flaggingPeer] = struct{}{}
	n.flaggedPosts.FlaggedPosts[postID] = flaggers
}

func (n *node) isPostFlaggedByMe(postID string) bool {
	// check if I flagged this post
	n.flaggedPosts.Lock()
	defer n.flaggedPosts.Unlock()
	flaggers, exists := n.flaggedPosts.FlaggedPosts[postID]
	if !exists {
		return false
	}
	_, flaggedByMe := flaggers[n.GetNodeIDStr()]
	return flaggedByMe
}

func (n *node) getFlagsForPost(postID string) (map[string]struct{}, bool) {
	// get the set of peer who flagged this post
	n.flaggedPosts.Lock()
	defer n.flaggedPosts.Unlock()
	flaggers, exists := n.flaggedPosts.FlaggedPosts[postID]
	return flaggers, exists
}

func (n *node) isPostFlaggedByMyFriends(postID string) bool {
	// check if a threshold of my followings flagged this post
	n.flaggedPosts.Lock()
	defer n.flaggedPosts.Unlock()
	flaggers, exists := n.flaggedPosts.FlaggedPosts[postID]
	if !exists {
		return false
	}
	if len(flaggers) > n.Configuration.FlagThreshold {
		return true
	}
	return false
}

func (n *node) dropReputation(peer string) {
	// decrease the reputation of a peer
	n.nodesReputation.Lock()
	defer n.nodesReputation.Unlock()
	n.nodesReputation.NodesReputation[peer]--
}

func (n *node) getReputation(peer string) int {
	n.nodesReputation.Lock()
	defer n.nodesReputation.Unlock()

	return n.nodesReputation.NodesReputation[peer]
}

func (n *node) createComment(postID, content string) types.Comment {
	return types.Comment{
		CommentID: xid.New().String(),
		Content:   content,
		AuthorID:  n.GetNodeIDStr(),
		PostID:    postID,
		CreatedAt: time.Now(),
	}
}

func (n *node) waitForChallengeReply(requestID, peer string) *types.PoSChallengeReplyMessage {
	cha := make(chan interface{})
	n.chanHandler.ChanAdd(requestID, cha)
	defer n.chanHandler.ChanRem(requestID)

	n.Debug().Msgf("[%s] Waiting for challenge reply...", n.Socket.GetAddress())

	select {
	case chanValue := <-cha:
		challengeReplyMsg, ok := chanValue.(*types.PoSChallengeReplyMessage)
		if !ok {
			_ = logAndReturnError(n.Logger, PError{message: "Could not convert channel value to PoSChallengeReplyMessage!"})
			return nil
		}

		n.Debug().Msgf("[%s] Received challenge reply!", n.Socket.GetAddress())

		mapDelete(n.Challenges, peer, &n.challenges.Mutex)
		return challengeReplyMsg
	case <-time.After(n.ChallengeTimeout):
		n.Debug().Msgf("[%s] Received challenge reply!", n.Socket.GetAddress())
		mapDelete(n.Challenges, peer, &n.challenges.Mutex)
		return nil
	}
}

// Sends a challenge to a node, waiting for a reply with timeout and validating the reply.
func (n *node) validatePeer(peer string, msg *transport.Message) bool {
	// Get the ChordID of that peer.
	peerChordID, err := getChordID(msg)
	if err != nil {
		_ = logAndReturnError(n.Logger, err)
		return false
	}

	n.Debug().Msgf("[%s] Validating peer %s (%s)", n.Socket.GetAddress(), peer, msg.Type)

	// Create and send a challenge to that peer.
	requestID, err := n.sendChallengeRequest(peer, peerChordID)
	if err != nil {
		_ = logAndReturnError(n.Logger, err)
		return false
	}

	challengeReplyMsg := n.waitForChallengeReply(requestID, peer)
	if challengeReplyMsg == nil {
		return false
	}

	validated, err := n.posModule.ValidateChallenge(challengeReplyMsg, true)
	if err != nil {
		_ = logAndReturnError(n.Logger, err)
		return false
	}

	n.Debug().Msgf("[%s] Challenge validated!", n.Socket.GetAddress())

	return validated
}

// updateNodePostKeys updates the encrypted post keys on the storage
func (n *node) updateNodePostKeys() {

	// for each node in followers
	for _, v := range n.GetFollowers() {
		// get followers public key
		n.followers.Lock()
		followerBytes := n.followers.Followers[v]
		var follower = utils.EncodeChordIDToStr(followerBytes)
		n.followers.Unlock()
		var queryStr = fm.Sprintf(PublicKeyFormatStr, follower)
		followerPublicKeyBytes := n.Storage.GetKeysStore().Get(queryStr)
		followerPublicKey, err := n2ccrypto.DecodePublicKey(followerPublicKeyBytes)
		if err != nil {
			n.Debug().Msgf("UpdateNodePostKeys: failed to decode node %s public key", follower)
			continue
		}

		n.Logger.Debug().Msgf("[UpdateNodePostKeys] number of postPPKModules %v", len(n.postPpkModules))
		// generate the encrypted decode key for the node
		for keyIDMinusOne, postPPKMod := range n.postPpkModules {
			var encryptedKey, err = n.ppkModule.Encrypt(postPPKMod.GetKey(),
				followerPublicKey)

			if err != nil {
				n.Debug().Msgf("updateNodePostKeys: failed to encrypt post private key for node %s ",
					follower)
				continue
			}
			key := fm.Sprintf(NodeDecodingKeyFormatStr, n.GetNodeIDStr(), keyIDMinusOne+1, follower)
			n.Storage.GetKeysStore().Set(key, encryptedKey)
		}

	}
}

func (n *node) generateNewPostPPKModule() error {
	if n.Configuration.PostEncryption {

		// Generate new PPK for new
		postPpk, err := n2ccrypto.NewDCypherModule(true,
			n.ppkFilename+fm.Sprintf("_Key_%v", len(n.postPpkModules)+1))
		if err != nil {
			panic("Failed to generate PPK module to encrypt private posts")
		}
		// store it locally
		n.postPpkModules = append(n.postPpkModules, postPpk)
		// Update chord storage
		n.updateNodePostKeys()

	}
	return nil
}

func (n *node) getMostRecentPostPPKModule() (*n2ccrypto.DCypherModule, int) {
	if len(n.postPpkModules) <= 0 {
		return nil, -1
	}
	return n.postPpkModules[len(n.postPpkModules)-1], len(n.postPpkModules)
}

func (n *node) getDecryptionKey(post *types.Post) ([]byte, error) {
	var decryptionKey []byte
	var err error

	if post.AuthorID == n.GetNodeIDStr() {
		decryptionKey = n.postPpkModules[post.KeyID-1].GetKey()
		n.Debug().Msgf("[decryptPostContent] node is author of the post %v", post.PostID)
	} else {
		var queryStr = fm.Sprintf(NodeDecodingKeyFormatStr,
			post.AuthorID, post.KeyID, n.GetNodeIDStr())
		var encryptedDecryptionKey = n.Storage.GetKeysStore().Get(queryStr)

		decryptionKey, err = n.ppkModule.Decrypt(encryptedDecryptionKey, n.ppkModule.PrivateKey)
		if err != nil {
			n.Debug().Msgf("[decryptPostContent] Failed to get decryption key bytes for post %v",
				post.PostID)
			return nil, xerrors.Errorf("[decryptPostContent] Failed to get decryption key bytes ")
		}
	}

	return decryptionKey, nil
}

// attempt to decrypt post content, if decryption fails returns an error
// returns post.Content if post isn't encrypted
func (n *node) decryptPostContent(post *types.Post) ([]byte, error) {
	if n.Configuration.PostEncryption && post.KeyID > 0 {
		// do decryption of post
		// get decryption key
		decryptionKey, err := n.getDecryptionKey(post)
		if err != nil {
			return nil, err
		}

		var decryptionModule = n2ccrypto.DCypherModule{}
		decryptionModule.LoadKeyToModule(decryptionKey)

		decryptedContent, err := decryptionModule.Decrypt(post.Content)
		if err != nil {
			return nil, xerrors.Errorf("[decryptPostContent] Failed to get decryption post content ")
		}
		return decryptedContent, nil
	}

	return post.Content, nil
}

// setupNodeKeys setups post ppk module and add it to chord storage
func (n *node) setupNodeKeys() error {

	if n.Configuration.PostEncryption {

		n.postPpkModules = make([]*n2ccrypto.DCypherModule, 0)
		// Add own public key to key store
		n.Storage.GetKeysStore().Set(fm.Sprintf(PublicKeyFormatStr, n.GetNodeIDStr()),
			n2ccrypto.GetBytesPublicKey(n.ppkModule.PublicKey))

		// Create new post ppk module
		return n.generateNewPostPPKModule()
	}
	return nil
}

func (n *node) isBlocked(authorID string) bool {
	n.blockedPeers.Lock()
	defer n.blockedPeers.Unlock()

	for _, v := range n.BlockedPeers {
		if v == authorID {
			return true
		}
	}

	return false
}

func (n *node) handlePostRequest(chordQueryMsg *types.ChordRequestMessage) error {
	myAddr := n.Socket.GetAddress()

	// store message in local storage
	postBytes, err := json.Marshal(chordQueryMsg.Post)
	if err != nil {
		return xerrors.Errorf("[handlePostRequest] failed to marshal post: %v", err)
	}
	n.Storage.GetPostStore().Set(chordQueryMsg.Post.PostID, postBytes)

	// convert postID str to bigInt
	postIDBigInt, err := utils.StrToBigInt(chordQueryMsg.Post.PostID)
	if err != nil {
		return xerrors.Errorf("[handlePostRequest] failed to convert postID [%s] to bigInt: %v",
			chordQueryMsg.Post.PostID, err)
	}

	n.Debug().Msgf("[handlePostRequest] [%s] Store post with PostID [%d] to PostStore", myAddr, postIDBigInt)
	if chordQueryMsg.IsPrivate {
		err = n.storePrivatePostInCatalog(chordQueryMsg.Post)
		if err != nil {
			return xerrors.Errorf("[handlePostRequest] failed to store private post in catalog: %v", err)
		}
		n.Debug().Msgf("[handlePostRequest] [%s] Store private post with PostID [%d] in catalog",
			myAddr, postIDBigInt)
	} else {
		err = n.storePublicPostInCatalog(chordQueryMsg.Post)
		if err != nil {
			return xerrors.Errorf("[handlePostRequest] failed to store public post in catalog: %v", err)
		}
		n.Debug().Msgf("[handlePostRequest] [%s] Store public post with PostID [%d] in catalog",
			myAddr, postIDBigInt)
	}
	return nil
}

func (n *node) waitForPostMessageAck(postReqMsg types.PostRequestMessage,
	keySuccessorNode types.ChordNode, transpPosReqtMsg transport.Message) (types.Post, error) {
	myAddr := n.Socket.GetAddress()

	// creat a channel to receive the post message ack message
	postMsgChan := make(chan interface{})
	n.chanHandler.ChanAdd(postReqMsg.RequestID, postMsgChan)
	defer n.chanHandler.ChanRem(postReqMsg.RequestID)

	attemptCount := 0
	for attemptCount < n.Configuration.RPCRetry {
		n.Debug().Msgf("[GetPost] [%s] Unicast RequestID [%s] to [%s]",
			myAddr, postReqMsg.RequestID, keySuccessorNode.IPAddr)
		err := n.Unicast(keySuccessorNode.IPAddr, transpPosReqtMsg)
		if err != nil {
			continue
		}

		select {
		case msg := <-postMsgChan:
			postReplyMsg, ok := msg.(*types.PostReplyMessage)
			if !ok {
				return postReplyMsg.Post, xerrors.Errorf("[GetPost] Received wrong message type. " +
					"Expected a postReplyMsg")
			}
			n.Debug().Msgf("[GetPost] [%s] Received post reply message for RequestID [%s]",
				myAddr, postReplyMsg.RequestID)

			if n.Configuration.PostEncryption {

				// decrypt post content
				decryptedContent, err := n.decryptPostContent(&postReplyMsg.Post)
				if err != nil {
					postReplyMsg.Post.Content = append([]byte("ENCRYPTED POST CONTENT>>"), postReplyMsg.Post.Content...)
				} else {
					postReplyMsg.Post.Content = decryptedContent
				}
			}
			return postReplyMsg.Post, nil
		case <-time.After(n.Configuration.AckTimeout):
			//timeout
		}
		attemptCount++
	}

	return types.Post{}, xerrors.Errorf("[GetPost] Max Number of retry reached")
}

func (n *node) AddFollowing(node types.ChordNode) {
	n.following.Following[node.IPAddr] = node.ID
}
