package impl

import (
	"github.com/rs/xid"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"go.dedis.ch/cs438/utils"
	"golang.org/x/xerrors"
	"math/big"
	"time"
)

// *****************************************************************************************
// *************************************** CHORD *******************************************
// *****************************************************************************************

// ask node n to find id’s successor
func (n *node) findSuccessor(id [KeyLen]byte) (types.ChordNode, error) {
	myAddr := n.Socket.GetAddress()
	n.Debug().Msgf("[findSuccessor] [%s] find successor for id [%d]", myAddr, utils.BytesToBigInt(id))

	// n' = find_predecessor(ID)
	predecessor, err := n.findPredecessor(id)
	if err != nil {
		return types.ChordNode{}, xerrors.Errorf("[findSuccessor] [%s] failed to find predecessor "+
			"for id [%d]: %v", myAddr, utils.BytesToBigInt(id), err)
	}
	n.Debug().Msgf("[findSuccessor] [%s] found predecessor for id [%s]: {ID: %d, IPAddr [%s]}",
		myAddr, utils.BytesToBigInt(id), predecessor.ID, predecessor.IPAddr)

	// n'.sucessor
	predecessorSuccessor, err := n.rpcSuccessor(predecessor)
	if err != nil {
		return types.ChordNode{}, xerrors.Errorf("[findSuccessor] [%s] failed to find predecessor [%s]'s successor "+
			"for id [%d]: %v", myAddr, predecessor.IPAddr, utils.BytesToBigInt(id), err)
	}
	n.Debug().Msgf("[findSuccessor] [%s] found predecessor's successor for id [%s]: {ID: %d, IPAddr [%s]}",
		myAddr, utils.BytesToBigInt(id), predecessorSuccessor.ID, predecessorSuccessor.IPAddr)

	// RETURN
	return predecessorSuccessor, nil
}

// ask node n to find id's predecessor
func (n *node) findPredecessor(id [KeyLen]byte) (types.ChordNode, error) {
	myAddr := n.Socket.GetAddress()
	n.Debug().Msgf("[findPredecessor] [%s] find predecessor for id [%d]", myAddr, utils.BytesToBigInt(id))
	var err error

	// n' = n
	remoteChord := types.ChordNode{
		ID:     n.chordID,
		IPAddr: n.Socket.GetAddress(),
	}
	// n'.successor
	remoteChordSuccessor := n.MySuccessor()

	// While ID not In Interval n' and n'.successor
	for !IsBetweenLeftOpenRightClosed(id, remoteChord.ID, remoteChordSuccessor.ID) {
		//mySlice := [][KeyLen]byte{id, remoteChord.ID, remoteChordSuccessor.ID}
		//DisplayRing(mySlice, "id", "n", "ns")
		n.Debug().Msgf("[findPredecessor] [%s] id [%d] does NOT belong to interval (%d, %d]",
			myAddr, utils.BytesToBigInt(id), utils.BytesToBigInt(remoteChord.ID), utils.BytesToBigInt(remoteChordSuccessor.ID))

		// n' = n'.closest_preceding_finger(id)
		remoteChord, err = n.rpcClosestPrecedingFinger(remoteChord, id)
		if err != nil {
			return types.ChordNode{}, xerrors.Errorf("[findPredecessor] [%s] failed to find rpcClosestPrecedingFinger "+
				"from remoteChord [%s] for id [%d]: %v", myAddr, remoteChord.IPAddr, utils.BytesToBigInt(id), err)
		}

		// n'.successor
		remoteChordSuccessor, err = n.rpcSuccessor(remoteChord)
		if err != nil {
			return types.ChordNode{}, xerrors.Errorf("[findPredecessor] [%s] failed to find rpcSuccessor "+
				"from remoteChord [%s]: %v", myAddr, remoteChord.IPAddr, err)
		}
	}
	n.Debug().Msgf("[findPredecessor] [%s] id [%d] belongs to interval (%d, %d]",
		myAddr, utils.BytesToBigInt(id), utils.BytesToBigInt(remoteChord.ID), utils.BytesToBigInt(remoteChordSuccessor.ID))

	return remoteChord, nil
}

// return closest finger preceding id
func (n *node) closestPrecedingFinger(id [KeyLen]byte) types.ChordNode {
	myAddr := n.Socket.GetAddress()

	for i := M - 1; i >= 0; i-- {
		fe, err := n.fingerTable.FingerTableGet(i)
		if err != nil {
			n.Error().Msgf("[closestPrecedingFinger] Error while FingerTableGet %v", err)
		}
		if IsStrictlyBetween(fe.Node.ID, n.chordID, id) {
			n.Debug().Msgf("[closestPrecedingFinger] [%s] return closest finger preceding id [%d]: [%s]",
				myAddr, utils.BytesToBigInt(id), fe.Node.IPAddr)
			return fe.Node
		}
	}
	myChordNode := types.ChordNode{
		ID:     n.chordID,
		IPAddr: n.Socket.GetAddress(),
	}
	n.Debug().Msgf("[closestPrecedingFinger] "+
		"[%s] return closest finger preceding id [%d]: myChordNode: [%s]",
		myAddr, utils.BytesToBigInt(id), myChordNode.IPAddr)
	return myChordNode
}

// ask node n to find id’s successor
func (n *node) MySuccessor() types.ChordNode {
	fe, err := n.fingerTable.FingerTableGet(0)
	if err != nil {
		n.Error().Msgf("[MySuccessor] Error while FingerTableGet: %v", err)
	}
	return fe.Node
}

// ask node n to find id’s Predecessor
func (n *node) MyPredecessor() types.ChordNode {
	return n.predecessorChord
}

func (n *node) GetChordNode() types.ChordNode {
	return types.ChordNode{ID: n.chordID, IPAddr: n.Socket.GetAddress()}
}

// Join the Chord network
// Node n learns its predecessor and figers by asking existingNode, an arbitrary node in the network, to look them up
func (n *node) Join(existingNode types.ChordNode) error {
	myAddr := n.Configuration.Socket.GetAddress()

	// if n is not the only node in the network
	if existingNode.IPAddr != "" {
		n.Debug().Msgf("[Join] [%s] is not the only node in the network", myAddr)
		n.Debug().Msgf("[Join] [%s] call InitFingerTable(%s)", myAddr, existingNode.IPAddr)
		n.Debug().Msgf("[Join] [%s] Adding peer %s to routing table", n.Socket.GetAddress(), existingNode.IPAddr)
		n.AddPeer(existingNode.IPAddr)

		err := n.InitFingerTable(existingNode)
		if err != nil {
			return err
		}

		n.Debug().Msgf("[Join] [%s] call updateOthers()", myAddr)
		err = n.updateOthers()
		if err != nil {
			return err
		}

		n.hasJoinedChordNetwork = true
		return nil
	}

	// if n is the only node in the network
	n.Debug().Msgf("[Join] [%s] is the first node in the network", myAddr)
	for i := 0; i < M; i++ {
		newChordNode := types.ChordNode{
			ID:     n.chordID,
			IPAddr: n.Socket.GetAddress(),
		}
		err := n.fingerTable.FingerTableSetNode(i, newChordNode)

		if err != nil {
			return xerrors.Errorf("[Join] Error while FingerTableSetNode: %v", err)
		}

	}
	//n.Debug().Msgf("[Join] [%s] Finger table: %v", myAddr, n.FingerTableToString())
	n.predecessorChord = types.ChordNode{
		ID:     n.chordID,
		IPAddr: n.Socket.GetAddress(),
	}

	n.hasJoinedChordNetwork = true
	return nil
}

// InitFingerTable initialize finger table using an arbitrary Chord node in network
func (n *node) InitFingerTable(existingNode types.ChordNode) error {
	myAddr := n.Socket.GetAddress()

	n.Debug().Msgf("[initFingerTable] [%s] intialize finger table using existing node [%s]",
		myAddr, existingNode.IPAddr)

	var err error

	fe, _ := n.fingerTable.FingerTableGet(0)
	newChordNode, err := n.rpcFindSuccessor(existingNode, fe.Start)
	n.Debug().Msgf("[initFingerTable] [%s] n.fingerTable[0].node = [%s]",
		myAddr, newChordNode.IPAddr)
	if err != nil {
		return err
	}

	err = n.fingerTable.FingerTableSetNode(0, newChordNode)
	if err != nil {
		return xerrors.Errorf("[InitFingerTable] error while FingerTableSetNode: %v", err)
	}

	n.Debug().Msgf("[initFingerTable] [%s] - start rpc get and set predecessor", myAddr)
	n.predecessorChord, err = n.rpcGetPredecessor(newChordNode)
	n.Debug().Msgf("[initFingerTable] [%s] Update our predecessor to: %s", myAddr, n.predecessorChord.IPAddr)
	if err != nil {
		return err
	}

	err = n.rpcSetPredecessor(newChordNode)
	if err != nil {
		return err
	}
	err = n.rpcSetSuccessor(n.predecessorChord)
	if err != nil {
		return err
	}

	for i := 0; i < M-1; i++ {
		fe1, err := n.fingerTable.FingerTableGet(i + 1)
		if err != nil {
			return xerrors.Errorf("[InitFingerTable] error while FingerTableGet: %v", err)
		}
		fe2, err := n.fingerTable.FingerTableGet(i)
		if err != nil {
			return xerrors.Errorf("[InitFingerTable] error while FingerTableGet: %v", err)
		}

		if IsBetweenLeftClosedRightOpen(fe1.Start.ID, n.chordID, fe2.Node.ID) {
			err := n.fingerTable.FingerTableAssignMinusOne(i + 1)
			if err != nil {
				return xerrors.Errorf("[InitFingerTable] error while FingerTableAssignMinusOne: %v", err)
			}
		} else {
			newChordNode, err := n.rpcFindSuccessor(existingNode, fe1.Start)
			if err != nil {
				return err
			}
			err = n.fingerTable.FingerTableSetNode(i+1, newChordNode)
			if err != nil {
				return xerrors.Errorf("[InitFingerTable] error while FingerTableSetNode: %v", err)
			}
		}
	}
	return nil

}

// update all nodes whose finger tables should refer to n
func (n *node) updateOthers() error {
	myAddr := n.Socket.GetAddress()
	n.Debug().Msgf("[updateOthers] [%s] update other nodes of network", myAddr)

	for i := 0; i < M; i++ {
		//_, _ = os.Stdout.WriteString("\n")
		n.Debug().Msgf("[updateOthers] [%s] [%d/%d]", myAddr, i, M-1)
		bigInt := new(big.Int)
		bigInt = bigInt.Exp(big.NewInt(2), big.NewInt(int64(i)), nil)

		id, err := SubtractIntFromBytes(n.chordID, bigInt)
		if err != nil {
			n.Error().Msgf("[updateOthers] [%d/%d] \n\nERR [1]\n\n", i, M-1)
			return err
		}
		p, err := n.findPredecessor(id)
		if err != nil {
			n.Error().Msgf("[updateOthers] [%d/%d] \n\nERR [2]\n\n", i, M-1)
			return err
		}
		err = n.rpcUpdateFingerTable(p, types.ChordNode{ID: n.chordID, IPAddr: n.Socket.GetAddress()}, uint(i))
		if err != nil {
			n.Error().Msgf("[updateOthers] [%d/%d] \n\nERR [3]\n\n", i, M-1)
			return err
		}
	}
	return nil
}

// if s is the ith finger of n, update n's finger table with s
func (n *node) updateFingerTable(s types.ChordNode, i uint) error {
	fe, err := n.fingerTable.FingerTableGet(int(i))
	if err != nil {
		return xerrors.Errorf("[updateFingerTable] Error while FingerTableGet: %v", err)
	}
	if IsBetweenLeftClosedRightOpen(s.ID, n.chordID, fe.Node.ID) {
		n.Debug().Msgf("[updateFingerTable][IsBetweenLeftClosedRightOpen] [%s] [%s] IS IN[%s - %s)",
			n.Socket.GetAddress(), utils.BytesToBigInt(s.ID), utils.BytesToBigInt(n.chordID),
			utils.BytesToBigInt(fe.Node.ID))
		n.Debug().Msgf("[updateFingerTable] [%s] IPaddr[%s] ID:[%d]",
			n.Socket.GetAddress(), s.IPAddr, utils.BytesToBigInt(s.ID))
		err = n.fingerTable.FingerTableSetNode(int(i), s)
		if err != nil {
			return xerrors.Errorf("[updateFingerTable] Error while FingerTableSetNode: %v", err)
		}

		//n.Debug().Msgf("[updateFingerTable] [%s]\n%s\n", n.Socket.GetAddress(), n.FingerTableToString())
		// update finger table of predecessor of n
		p := n.predecessorChord
		//if s.IPAddr != n.Socket.GetAddress() { // TODO check ??
		err := n.rpcUpdateFingerTable(p, s, i)
		if err != nil {
			return err
		}
		//}
	} else {
		n.Debug().Msgf("[updateFingerTable][IsBetweenLeftClosedRightOpen] [%s] [%s] IS NOT IN[%s - %s)",
			n.Socket.GetAddress(), utils.BytesToBigInt(s.ID), utils.BytesToBigInt(n.chordID),
			utils.BytesToBigInt(fe.Node.ID))
	}
	n.Debug().Msgf("[updateFingerTable] [%s] OUT OF INTERVAL - IPaddr[%s] ID:[%d]",
		n.Socket.GetAddress(), s.IPAddr, utils.BytesToBigInt(s.ID))
	return nil
}

// *****************************************************************************************
// *************************** CHORD REMOTE PROCEDURE CALLS ********************************
// *****************************************************************************************

// if s is the ith finger of remote chord node, update remote chord node's finger table with s
func (n *node) rpcUpdateFingerTable(remoteChordNode types.ChordNode, s types.ChordNode, i uint) error {
	myAddr := n.Socket.GetAddress()

	if s.IPAddr == remoteChordNode.IPAddr {
		n.Debug().Msgf("[rpcUpdateFingerTable] [%s] Base case to avoid infinite loop", myAddr)
		return nil
	}

	/*if remoteChordNode.IPAddr == myAddr {
		n.Debug().Msgf("[rpcUpdateFingerTable] [%s] local node's call shortcut", myAddr)
		return n.updateFingerTable(s, i)
	}

	*/

	msg := types.ChordRequestMessage{Query: types.RPCUpdateFingerTable, ChordNode: s, IthFinger: i,
		RequestID: xid.New().String()}
	transpMsg, _ := n.TypesMsgToTranspMsg(msg)

	// send ChordRequestMessage and wait for ChordReplyMessage
	_, err := n.sendRequestAndWaitReply(remoteChordNode, msg.RequestID, transpMsg, types.ChordQueryStr[msg.Query])
	if err != nil {
		return xerrors.Errorf("[rpcUpdateFingerTable] failed to sendRequestAndWaitReply: %v", err)
	}
	return nil

}

// rpcClosestPrecedingFinger call the remoteNode and return its closest finger preceding id
func (n *node) rpcClosestPrecedingFinger(remoteChordNode types.ChordNode, id [KeyLen]byte) (types.ChordNode, error) {
	myAddr := n.Socket.GetAddress()
	n.Debug().Msgf("[rpcClosestPrecedingFinger] "+
		"[%s] call remoteNode [%s] and return its closest finger preceding id [%d]",
		myAddr, remoteChordNode.IPAddr, utils.BytesToBigInt(id))

	if remoteChordNode.IPAddr == myAddr {
		n.Debug().Msgf("[rpcClosestPrecedingFinger] "+
			"[%s] local node's closest finger preceding id [%d]: [%s]",
			myAddr, utils.BytesToBigInt(id), remoteChordNode.IPAddr)
		return n.closestPrecedingFinger(id), nil
	}

	msg := types.ChordRequestMessage{Query: types.RPCClosestPrecedingFinger,
		ChordNode: types.ChordNode{ID: id, IPAddr: ""}, RequestID: xid.New().String()}
	transpMsg, _ := n.TypesMsgToTranspMsg(msg)

	// send ChordRequestMessage and wait for ChordReplyMessage
	replyMsg, err := n.sendRequestAndWaitReply(remoteChordNode, msg.RequestID, transpMsg, types.ChordQueryStr[msg.Query])
	if err != nil {
		return types.ChordNode{}, xerrors.Errorf("[rpcClosestPrecedingFinger] failed to sendRequestAndWaitReply: %v", err)
	}
	return replyMsg.ChordNode, nil

}

// rpcSuccessor call the remoteNode and return its successor
func (n *node) rpcSuccessor(remoteChordNode types.ChordNode) (types.ChordNode, error) {
	myAddr := n.Socket.GetAddress()
	n.Debug().Msgf("[rpcSuccessor] [%s] call the remoteNode [%s] and return its successor",
		myAddr, remoteChordNode.IPAddr)

	if remoteChordNode.IPAddr == myAddr {
		return n.MySuccessor(), nil
	}

	msg := types.ChordRequestMessage{Query: types.RPCSuccessor, RequestID: xid.New().String()}
	transpMsg, _ := n.TypesMsgToTranspMsg(msg)

	// send ChordRequestMessage and wait for ChordReplyMessage
	replyMsg, err := n.sendRequestAndWaitReply(remoteChordNode, msg.RequestID, transpMsg, types.ChordQueryStr[msg.Query])
	if err != nil {
		return types.ChordNode{}, xerrors.Errorf("[rpcSuccessor] failed to sendRequestAndWaitReply: %v", err)
	}
	return replyMsg.ChordNode, nil

}

func (n *node) rpcGetPredecessor(remoteChordNode types.ChordNode) (types.ChordNode, error) {
	myAddr := n.Socket.GetAddress()
	n.Debug().Msgf("[rpcGetPredecessor] [%s] call node [%s] and return its predecessor",
		myAddr, remoteChordNode.IPAddr)

	// create ChordRequestMessage
	msg := types.ChordRequestMessage{Query: types.RPCGetPredecessor, RequestID: xid.New().String(),
		ChordNode: types.ChordNode{ID: n.chordID, IPAddr: n.Socket.GetAddress()}}
	transpMsg, _ := n.TypesMsgToTranspMsg(msg)

	// send ChordRequestMessage and wait for ChordReplyMessage
	replyMsg, err := n.sendRequestAndWaitReply(remoteChordNode, msg.RequestID, transpMsg, types.ChordQueryStr[msg.Query])
	if err != nil {
		return types.ChordNode{}, xerrors.Errorf("[rpcGetPredecessor] failed to sendRequestAndWaitReply: %v", err)
	}
	return replyMsg.ChordNode, nil

}

func (n *node) rpcSetPredecessor(remoteChordNode types.ChordNode) error {
	myAddr := n.Socket.GetAddress()
	n.Debug().Msgf("[rpcSetPredecessor] [%s] call remoteChordNode [%s] and set ourselves as predecessor",
		myAddr, remoteChordNode.IPAddr)

	msg := types.ChordRequestMessage{Query: types.RPCSetPredecessor, RequestID: xid.New().String(),
		ChordNode: types.ChordNode{ID: n.chordID, IPAddr: n.Socket.GetAddress()}}

	transpMsg, _ := n.TypesMsgToTranspMsg(msg)

	msgChan := make(chan interface{})
	n.chanHandler.ChanAdd(msg.RequestID, msgChan)
	defer n.chanHandler.ChanRem(msg.RequestID)

	// send ChordRequestMessage and wait for ChordReplyMessage
	_, err := n.sendRequestAndWaitReply(remoteChordNode, msg.RequestID, transpMsg, types.ChordQueryStr[msg.Query])
	if err != nil {
		return xerrors.Errorf("[rpcGetPredecessor] failed to sendRequestAndWaitReply: %v", err)
	}
	return nil
}

// rpcSetSuccessor call the remoteNode and set ourselves as the successor of the remote node
func (n *node) rpcSetSuccessor(remoteChordNode types.ChordNode) error {
	myAddr := n.Socket.GetAddress()
	n.Debug().Msgf("[rpcSetSuccessor] [%s] call remoteChordNode [%s] and set ourselves as successor",
		myAddr, remoteChordNode.IPAddr)
	msg := types.ChordRequestMessage{Query: types.RPCSetSuccessor, RequestID: xid.New().String(),
		ChordNode: types.ChordNode{ID: n.chordID, IPAddr: n.Socket.GetAddress()}}
	transpMsg, _ := n.TypesMsgToTranspMsg(msg)

	// send ChordRequestMessage and wait for ChordReplyMessage
	_, err := n.sendRequestAndWaitReply(remoteChordNode, msg.RequestID, transpMsg, types.ChordQueryStr[msg.Query])
	if err != nil {
		return xerrors.Errorf("[rpcGetPredecessor] failed to sendRequestAndWaitReply: %v", err)
	}
	return nil
}

// rpcFindSuccessor call the remoteNode and return the successor of the start of the 1st finger
func (n *node) rpcFindSuccessor(remoteChordNode, callerFirstFinger types.ChordNode) (types.ChordNode, error) {
	myAddr := n.Socket.GetAddress()
	n.Debug().Msgf("[rpcFindSuccessor] [%s] call remoteChordNode [%s] and return the successor of "+
		"the 1st finger's start ID [%d]", myAddr, remoteChordNode.IPAddr, utils.BytesToBigInt(callerFirstFinger.ID))

	msg := types.ChordRequestMessage{
		RequestID: xid.New().String(),
		Query:     types.RPCFindSuccessor,
		ChordNode: callerFirstFinger,
	}
	//n.Debug().Msgf("[rpcFindSuccessor] [%s] create msg [%v]", myAddr, msg)

	transpMsg, _ := n.TypesMsgToTranspMsg(msg)

	// send ChordRequestMessage and wait for ChordReplyMessage
	replyMsg, err := n.sendRequestAndWaitReply(remoteChordNode, msg.RequestID, transpMsg, types.ChordQueryStr[msg.Query])
	if err != nil {
		return types.ChordNode{}, xerrors.Errorf("[rpcFindSuccessor] failed to sendRequestAndWaitReply: %v", err)
	}
	return replyMsg.ChordNode, nil
}

// rpcGetPostCatalog call the remoteNode and return the entire post catalog of that node
func (n *node) rpcGetPostCatalog(remoteChordNode types.ChordNode) (catalog map[string][]byte, err error) {
	myAddr := n.Socket.GetAddress()
	n.Debug().Msgf("[rpcGetPostCatalog] [%s] call remoteChordNode [%s] and return its entire catalog",
		myAddr, remoteChordNode.IPAddr)

	msg := types.CatalogRequestMessage{
		RequestID: xid.New().String(),
	}
	//n.Debug().Msgf("[rpcFindSuccessor] [%s] create msg [%v]", myAddr, msg)

	transpMsg, _ := n.TypesMsgToTranspMsg(msg)

	msgChan := make(chan interface{})
	n.chanHandler.ChanAdd(msg.RequestID, msgChan)
	defer n.chanHandler.ChanRem(msg.RequestID)

	attemptCount := 0
	for attemptCount < n.Configuration.RPCRetry {
		n.Debug().Msgf("[rpcGetPostCatalog] [%s] Unicast RequestID [%s] to [%s]",
			myAddr, msg.RequestID, remoteChordNode.IPAddr)

		err := n.Unicast(remoteChordNode.IPAddr, transpMsg)
		if err != nil {
			continue
		}

		select {
		case msg := <-msgChan:
			replyMsg, ok := msg.(*types.CatalogReplyMessage)
			if !ok {
				return make(map[string][]byte), xerrors.Errorf("[rpcGetPostCatalog] Wrong message type" +
					" expected: CatalogReplyMessage")
			}
			return replyMsg.Catalog, nil
		case <-time.After(n.Configuration.AckTimeout):
			//timeout
		}
		attemptCount++
	}

	return make(map[string][]byte), xerrors.Errorf("[rpcGetPostCatalog] Max Number of retry reached")
}

// ******* Chord utils ******

// sendRequestAndWaitReply send a ChordRequestMessage to a remote
// Chord Node and wait for a ChordReplyMessage on a channel
func (n *node) sendRequestAndWaitReply(remoteChordNode types.ChordNode, requestID string,
	transpMsg transport.Message, msgQuery string) (replyMsg *types.ChordReplyMessage, err error) {

	myAddr := n.Socket.GetAddress()

	n.Debug().Msgf("[sendRequestAndWaitReply] [%s] Add channel with key RequestID [%s] to channel handler",
		myAddr, requestID)

	msgChan := make(chan interface{})
	n.chanHandler.ChanAdd(requestID, msgChan)
	defer n.chanHandler.ChanRem(requestID)

	attemptCount := 0
	for attemptCount < n.Configuration.RPCRetry {
		n.Debug().Msgf("[sendRequestAndWaitReply] [%s] Send RequestID [%s] of type [%s > %s] to [%s] (attempt %d/%d)",
			myAddr, requestID, transpMsg.Type, msgQuery, remoteChordNode.IPAddr, attemptCount+1, n.Configuration.RPCRetry)
		err := n.Unicast(remoteChordNode.IPAddr, transpMsg)
		if err != nil {
			continue
		}

		select {
		case msg := <-msgChan:
			replyMsg, ok := msg.(*types.ChordReplyMessage)
			n.Debug().Msgf("[sendRequestAndWaitReply] [%s] Successfully received reply for RequestID [%s]",
				myAddr, requestID)
			if !ok {
				return &types.ChordReplyMessage{}, xerrors.Errorf("[sendRequestAndWaitReply] Wrong message type. " +
					"Expected: ChordReplyMessage")
			}
			return replyMsg, nil
		case <-time.After(n.Configuration.AckTimeout):
			//timeout
		}
		attemptCount++
	}
	return &types.ChordReplyMessage{}, xerrors.Errorf("[sendRequestAndWaitReply] Max Number of retries reached.")
}

// TypesMsgToTranspMsg
func (n *node) TypesMsgToTranspMsg(msg types.Message) (transport.Message, error) {
	transpMsg, err := n.Configuration.MessageRegistry.MarshalMessage(msg)
	if err != nil {
		return transport.Message{}, err
	}

	return transpMsg, nil
}
