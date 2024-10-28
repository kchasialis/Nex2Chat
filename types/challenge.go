package types

import (
	"fmt"
)

// -----------------------------------------------------------------------------

// PoSChallengeRequestMessage

// NewEmpty implements types.Message.
func (c PoSChallengeRequestMessage) NewEmpty() Message {
	return &PoSChallengeRequestMessage{}
}

// Name implements types.Message.
func (PoSChallengeRequestMessage) Name() string {
	return "PoSChallengeReq"
}

// String implements types.Message.
func (c PoSChallengeRequestMessage) String() string {
	return fmt.Sprintf(`{\n,
		Verification Method: %v,
		RequestedNumberHash: %v,
		DifficultyFactor: %v,
		Nonce: %v,
		TargetNodeID: %v,
		\n}`, c.VerificationMethodID, c.RequestedNumberHash, c.DifficultyFactor, c.Nonce, c.TargetNodeID)
}

// HTML implements types.Message.
func (c PoSChallengeRequestMessage) HTML() string {
	return c.String()
}

// Get verification method associated with given challenge message
func (c PoSChallengeRequestMessage) GetChallengeVerificationMethodID() string {
	return "PoSVerificationModuleV1.0.0"
}

// -----------------------------------------------------------------------------

// PoSChallengeReplyMessage

// NewEmpty implements types.Message.
func (c PoSChallengeReplyMessage) NewEmpty() Message {
	return &PoSChallengeReplyMessage{}
}

// Name implements types.Message.
func (PoSChallengeReplyMessage) Name() string {
	return "PoSChallengeReply"
}

// String implements types.Message.
func (c PoSChallengeReplyMessage) String() string {
	return fmt.Sprintf(`{\n,
		Verification Method: %v,
		RequestedNumber: %v,
		Nonce: %v,
		ReplyingNodeID: %v,
		\n}`, c.VerificationMethodID, c.RequestedNumber, c.Nonce, c.ReplyingNodeID)
}

// HTML implements types.Message.
func (c PoSChallengeReplyMessage) HTML() string {
	return c.String()
}

// Get verification method associated with given challenge message
func (c PoSChallengeReplyMessage) GetChallengeVerificationMethodID() string {
	return "PoSVerificationModuleV1.0.0"
}
