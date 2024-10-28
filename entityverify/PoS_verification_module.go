package entityverify

/*
	Proof of Storage (PoS) based entity verification
*/

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.dedis.ch/cs438/utils"

	"go.dedis.ch/cs438/n2ccrypto"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

const (
	RangePerDifficultyFactor uint = 4   // number of hashes per difficulty factor
	MaxDifficultyFactor      uint = 100 // maximum difficulty factor that will be generated
)

type PoSChallengeData struct {
	NodeID         []byte
	Nonce          []byte
	RandomN        uint
	RandomNHash    []byte
	GenerationTime time.Time
}

func (data *PoSChallengeData) ID() string {
	return fmt.Sprintf("%v-%v", data.NodeID, data.Nonce)
}

type PoSModule struct {
	// currently calculated difficulty factor bound (currently cached difficulty factor + 1)
	currentDifficultyFactorBound uint
	// hashtable of salted number hashes to corresponding number
	mapHashToNumber map[string]uint
	// ChordID associated to PoSModule (salt)
	nodeID              string
	generatedChallenges map[string]*PoSChallengeData
	moduleLock          sync.RWMutex
}

func (module *PoSModule) GetNodeID() string {
	return module.nodeID
}

func NewPoSModule(generateToDifficultyFactor uint, nodeID []byte) *PoSModule {
	var module = &PoSModule{currentDifficultyFactorBound: 0,
		mapHashToNumber:     make(map[string]uint),
		nodeID:              string(nodeID[:]),
		generatedChallenges: make(map[string]*PoSChallengeData),
	}
	// pregenerate table
	module.GenerateHashesToDifficultyFactor(generateToDifficultyFactor)
	return module
}

// generates the table up to given difficulty factor
func (module *PoSModule) GenerateHashesToDifficultyFactor(difficultyFactor uint) {
	module.moduleLock.Lock()
	// for each difficulty factor
	for df := module.currentDifficultyFactorBound; df <= difficultyFactor && df <= MaxDifficultyFactor; df++ {
		// generate numbers in the range for that difficulty factor
		for pos := uint(0); pos < RangePerDifficultyFactor; pos++ {
			var number = pos + df*RangePerDifficultyFactor
			var numberStr = fmt.Sprintf("%v", number)
			var numberHash = n2ccrypto.StrongHashSalted([]byte(numberStr), []byte(module.nodeID))
			module.mapHashToNumber[string(numberHash)] = number
		}
		module.currentDifficultyFactorBound = df
	}
	module.moduleLock.Unlock()
}

// GenerateChallenge generates the challenge for a given ChordID, nonce and difficulty factor,
// and saves it in the generated challenges table
func (module *PoSModule) GenerateChallenge(NodeID []byte, nonce []byte, difficultyFactor uint) (
	*types.PoSChallengeRequestMessage, error) {

	// generate random number in difficulty interval
	var randN = randomUintInRange(0, (difficultyFactor+1)*RangePerDifficultyFactor-1)
	// hash it with ChordID as salt
	var randNHash = n2ccrypto.StrongHashSalted([]byte(fmt.Sprintf("%v", randN)), NodeID)

	// save challenge to table
	var challengeDataPtr = &PoSChallengeData{
		NodeID:         NodeID,
		Nonce:          nonce,
		RandomN:        randN,
		RandomNHash:    randNHash,
		GenerationTime: time.Now(),
	}

	module.moduleLock.Lock()
	module.generatedChallenges[challengeDataPtr.ID()] = challengeDataPtr
	module.moduleLock.Unlock()

	// construct PoSChallengeRequestMessage
	var msgPtr = &types.PoSChallengeRequestMessage{
		RequestID:            string(nonce[:]),
		VerificationMethodID: module.GetVerificationMethodID(),
		RequestedNumberHash:  randNHash,
		DifficultyFactor:     difficultyFactor,
		Nonce:                nonce,
		TargetNodeID:         NodeID,
	}
	return msgPtr, nil
}

// SolveChallenge solve the challenge request and generate a reply
// returns error if the message is not compatible,
// or if the challenge request is malformated (potential malicious behaviour)
func (module *PoSModule) SolveChallenge(request *types.PoSChallengeRequestMessage) (
	*types.PoSChallengeReplyMessage, error) {

	// assert that Challenge message is of expected type
	if !module.IsVerificationMethodIDCompatible(request.GetChallengeVerificationMethodID()) {
		return nil, xerrors.Errorf("SolveChallenge: ChallengeMessage isnt compatible with this PoS Module")
	}

	// generate table if not generated before
	module.moduleLock.RLock()
	currentDifficulty := module.currentDifficultyFactorBound
	module.moduleLock.RUnlock()
	if currentDifficulty > request.DifficultyFactor+1 {
		return nil, xerrors.Errorf("SolveChallenge: requested Difficulty factor is above maximum cap")
	}
	module.GenerateHashesToDifficultyFactor(request.DifficultyFactor)

	// query map for the respective number
	module.moduleLock.RLock()
	var number, numberExists = module.mapHashToNumber[string(request.RequestedNumberHash)]
	module.moduleLock.RUnlock()
	if !numberExists {
		return nil, xerrors.Errorf(`SolveChallenge: hash doesnt exist on map for difficulty
		 factor %v and ChordID %v , cant retrieve number`, request.DifficultyFactor,
			utils.BytesToBigInt([types.KeyLen]byte([]byte(module.nodeID))))
	}

	// generate reply message
	module.moduleLock.RLock()
	var replyMsg = &types.PoSChallengeReplyMessage{
		RequestID:            request.RequestID,
		VerificationMethodID: module.GetVerificationMethodID(),
		RequestedNumber:      number,
		Nonce:                request.Nonce,
		ReplyingNodeID:       []byte(module.nodeID),
	}
	module.moduleLock.RUnlock()

	return replyMsg, nil
}

// validate the challenge, and check if the challenge is in the generated challenges table
// Assumptions:
//   - Verification that ChordID belongs to whoever sent the PoSChallengeReplyMessage was done before
func (module *PoSModule) ValidateChallenge(reply *types.PoSChallengeReplyMessage,
	deleteFromGeneratedTable bool) (bool, error) {

	// assert that Challenge message is of expected type
	if !module.IsVerificationMethodIDCompatible(reply.GetChallengeVerificationMethodID()) {
		return false, xerrors.Errorf("ValidateChallenge: ChallengeMessage isnt compatible with this PoS Module")
	}

	// check if Challenge is in generated challenges table
	module.moduleLock.RLock()
	var queryChallengeData = PoSChallengeData{NodeID: reply.ReplyingNodeID, Nonce: reply.Nonce}
	var challengeData, challengeExists = module.generatedChallenges[queryChallengeData.ID()]
	module.moduleLock.RUnlock()

	// validate replied number
	if !(challengeExists && challengeData.RandomN == reply.RequestedNumber) {
		return false, nil
	}

	// delete from generated challenges table if requested
	if deleteFromGeneratedTable {
		module.DeleteChallengeFromTable(queryChallengeData.NodeID, queryChallengeData.Nonce)
	}
	return true, nil
}

// deletes a challenge from the table, if it exists
func (module *PoSModule) DeleteChallengeFromTable(NodeID []byte, nonce []byte) {
	module.moduleLock.Lock()
	var tmpChallengeData = PoSChallengeData{NodeID: NodeID, Nonce: nonce}
	// delete from table
	delete(module.generatedChallenges, tmpChallengeData.ID())
	module.moduleLock.Unlock()
}

func (module *PoSModule) IsVerificationMethodIDCompatible(ID string) bool {
	var ownID = module.GetVerificationMethodID()
	var isCompatible = ownID == ID
	return isCompatible
}

func (module *PoSModule) GetVerificationMethodID() string {
	return "PoSVerificationModuleV1.0.0"
}

func randomUintInRange(min, max uint) uint {
	// swap order if min is bigger than max
	if min >= max {
		var tmp = max
		max = min
		min = tmp
	}
	// seed generator
	return min + uint(rand.Uint64()%uint64(max-min+1))
}
