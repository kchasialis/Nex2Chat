package entityverify

import (
	"testing"
)

// Same ChordID, same Nonce, correct value answered
func TestPoSVerificationModule_Creation_Validation_1(t *testing.T) {

	tests := []struct {
		name string
	}{
		{
			name: "Correct module creation, Same ChordID, same Nonce, correct value answered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var module1 = NewPoSModule(1, []byte("Node1"))
			var module2 = NewPoSModule(1, []byte("Node2"))

			var challengeReq, err = module1.GenerateChallenge([]byte("Node2"), []byte("Nonce123"), 1)
			if err != nil {
				t.Fatalf("TestPoSVerificationModule_Creation_Validation_1: failed to generate challenge")
			}

			var challengeReply, err2 = module2.SolveChallenge(challengeReq)
			if err2 != nil {
				t.Fatalf("TestPoSVerificationModule_Creation_Validation_1: failed to solve challenge")
			}

			var validated, err3 = module1.ValidateChallenge(challengeReply, true)
			if err3 != nil {
				t.Fatalf("TestPoSVerificationModule_Creation_Validation_1: failed to validate challenge")
			}
			if !validated {

				t.Fatalf("TestPoSVerificationModule_Creation_Validation_1: failed to validate challenge")
			}

		})
	}
}

// different ChordID,same  Nonce, correct value answered
func TestPoSVerificationModule_Creation_Validation_2(t *testing.T) {

	tests := []struct {
		name string
	}{
		{
			name: "Correct module creation, different ChordID,same  Nonce, correct value answered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var module1 = NewPoSModule(1, []byte("Node1"))
			var module2 = NewPoSModule(1, []byte("Node2"))

			var challengeReq, err = module1.GenerateChallenge([]byte("Node3"), []byte("Nonce123"), 1)
			if err != nil {
				t.Fatalf("TestPoSVerificationModule_Creation_Validation_2: failed to generate challenge")
			}

			var _, err2 = module2.SolveChallenge(challengeReq)
			if err2 == nil {
				t.Fatalf("TestPoSVerificationModule_Creation_Validation_2: shouldnt be able to solve challenge")
			}

		})
	}
}

// Same ChordID, different Nonce, correct value answered
func TestPoSVerificationModule_Creation_Validation_3(t *testing.T) {

	tests := []struct {
		name string
	}{
		{
			name: "Correct module creation, Same ChordID, different Nonce, correct value answered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var module1 = NewPoSModule(1, []byte("Node1"))
			var module2 = NewPoSModule(1, []byte("Node2"))

			var challengeReq, err = module1.GenerateChallenge([]byte("Node2"), []byte("Nonce123"), 1)
			if err != nil {
				t.Fatalf("TestPoSVerificationModule_Creation_Validation_1: failed to generate challenge")
			}

			var challengeReply, err2 = module2.SolveChallenge(challengeReq)
			if err2 != nil {
				t.Fatalf("TestPoSVerificationModule_Creation_Validation_1: failed to solve challenge")
			}
			challengeReply.Nonce = []byte("NotSameNonce")

			var validated, err3 = module1.ValidateChallenge(challengeReply, true)
			if err3 != nil || validated == true {
				t.Fatalf("TestPoSVerificationModule_Creation_Validation_2: should fail to validate challenge")
			}

		})
	}
}

// Same ChordID, same Nonce, incorrect value answered
func TestPoSVerificationModule_Creation_Validation_4(t *testing.T) {

	tests := []struct {
		name string
	}{
		{
			name: "Correct module creation, Same ChordID, same Nonce, incorrect value answered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var module1 = NewPoSModule(1, []byte("Node1"))
			var module2 = NewPoSModule(1, []byte("Node2"))

			var challengeReq, err = module1.GenerateChallenge([]byte("Node2"), []byte("Nonce123"), 1)
			if err != nil {
				t.Fatalf("TestPoSVerificationModule_Creation_Validation_4: failed to generate challenge")
			}

			var challengeReply, err2 = module2.SolveChallenge(challengeReq)
			if err2 != nil {
				t.Fatalf("TestPoSVerificationModule_Creation_Validation_4: failed to solve challenge")
			}
			challengeReply.RequestedNumber = MaxDifficultyFactor * RangePerDifficultyFactor * 2

			var validated, err3 = module1.ValidateChallenge(challengeReply, true)
			if err3 != nil || validated == true {
				t.Fatalf("TestPoSVerificationModule_Creation_Validation_4: should fail to validate challenge")
			}

		})
	}
}
