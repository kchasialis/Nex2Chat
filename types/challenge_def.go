package types

// challenge message interface
//type ChallengeMessage interface {
//	GetChallengeVerificationMethodID() string
//}

//// ChallengeRequestMessage is a message sent to a node requesting it to solve a challenge
//type ChallengeRequestMessage struct {
//	VerificationMethodID string // name/ID of verification method expected
//}
//
//// ChallengeReplyMessage is a message sent to a node with the solution to a challenge
//type ChallengeReplyMessage struct {
//	VerificationMethodID string // name/ID of verification method expected
//}

// PoSChallengeRequestMessage is a message sent to a node requesting it to solve a
// proof of storage challenge
type PoSChallengeRequestMessage struct {
	RequestID            string
	VerificationMethodID string
	RequestedNumberHash  []byte
	DifficultyFactor     uint
	Nonce                []byte
	TargetNodeID         []byte
}

// PoSChallengeReplyMessage is a message sent to a node with the solution to a
// proof of storage challenge
type PoSChallengeReplyMessage struct {
	RequestID            string
	VerificationMethodID string
	RequestedNumber      uint
	Nonce                []byte
	ReplyingNodeID       []byte
}
