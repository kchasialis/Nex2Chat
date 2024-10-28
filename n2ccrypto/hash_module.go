package n2ccrypto

/*
	Hash generation module, with variable space/compute complexity
*/
import (
	"crypto"
	"encoding/base64"

	"golang.org/x/crypto/argon2"
)

const (
	// Parameters for Argon2
	DefaultTimeCost   = 1         // the amount of computation realized and the execution time
	DefaultMemoryCost = 64 * 1024 // the size of the memory (in kilobytes)
	DefaultThreads    = 1         // the number of threads to use
	DefaultKeyLen     = 1         // the length of the desired key (in bytes)
	DefaultStrongHash = false
)

func Hash(data []byte) []byte {
	if DefaultStrongHash {

		return StrongHashWithConfig(data, make([]byte, 0),
			DefaultTimeCost, DefaultMemoryCost, DefaultThreads, DefaultKeyLen, false)
	}
	return ShaHashSalted(data, make([]byte, 0))[:DefaultKeyLen]
}

func HashSalted(data []byte, salt []byte) []byte {
	if DefaultStrongHash {

		return StrongHashWithConfig(data, salt,
			DefaultTimeCost, DefaultMemoryCost, DefaultThreads, DefaultKeyLen, false)
	}
	return ShaHashSalted(data, salt)[:DefaultKeyLen]

}

// hashes using crypto sha3_512
func ShaHashSalted(data []byte, salt []byte) []byte {
	hasher := crypto.SHA512.New()
	hasher.Write(data)
	hasher.Write(salt)
	return hasher.Sum(nil)
}

// hashes using argon2
func StrongHash(data []byte) []byte {
	return StrongHashWithConfig(data, make([]byte, 0), DefaultTimeCost,
		DefaultMemoryCost, DefaultThreads, DefaultKeyLen, false)
}

// hashes using argon2
func StrongHashSalted(data []byte, salt []byte) []byte {

	return StrongHashWithConfig(data, salt, DefaultTimeCost,
		DefaultMemoryCost, DefaultThreads, DefaultKeyLen, false)
}

/*
Hash the input data with given salt
timeCost:  number of algorithm passes over memory
memoryCost: memory cost for algorithm, in kilobytes
threads: number of threads used/ parallelism degree
keyLen: hash ouput length, in bytes
*/
func StrongHashWithConfig(input []byte, salt []byte, timeCost uint32,
	memoryCost uint32, threads uint8, keyLen uint32, base64Encode bool) []byte {

	if timeCost < 1 {
		timeCost = 1
	}

	if threads < 1 {
		threads = 1
	}

	if salt == nil {
		salt = make([]byte, 0)
	}
	if input == nil {
		input = make([]byte, 0)
	}

	// Use Argon2 IDKey to hash the input
	hashed := argon2.IDKey(input, salt, timeCost, memoryCost, threads, keyLen)

	if base64Encode {
		// Encode the hash in base64
		base64Encoded := make([]byte, base64.StdEncoding.EncodedLen(len(hashed)))
		base64.StdEncoding.Encode(base64Encoded, hashed)
		return base64Encoded
	}
	return hashed
}

//// Secure comparison of hashes, to prevent timing attacks
//func secureCompare(a, b []byte) bool {
//	// Use ConstantTimeCompare to compare slices in constant time
//	return subtle.ConstantTimeCompare(a, b) == 1
//}
