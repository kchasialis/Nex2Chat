package entityverify

import "go.dedis.ch/cs438/types"

/*
	Entity verification module interface definition
*/

// EntityVerification represents the entity verification itnerface
type EntityVerificationInterface interface {
	GenerateChallenge(NodeID []byte, nonce []byte, difficultyFactor uint) (*types.PoSChallengeRequestMessage, error)
	SolveChallenge(request *types.PoSChallengeRequestMessage) (*types.PoSChallengeReplyMessage, error)
	ValidateChallenge(reply *types.PoSChallengeReplyMessage, deleteFromGeneratedTable bool) (bool, error)
	GetVerificationMethodID() string
	DeleteChallengeFromTable(NodeID []byte, nonce []byte)
}
