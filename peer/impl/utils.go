package impl

import (
	"bytes"
	"crypto"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/n2ccrypto"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/storage"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"go.dedis.ch/cs438/utils"
)

type RetryError struct {
	ID uint
}

func (e RetryError) Error() string {
	return "RetryError"
}

type PError struct {
	message string
}

func (e PError) Error() string {
	return e.message
}

func isTimeout(err error) bool {
	var netErr net.Error
	return (errors.As(err, &netErr) && netErr.Timeout()) || (errors.Is(err, transport.TimeoutError(0)))
}

// Thread-safe map operations using RWMutex
func mapAddRW[K comparable, V any](m map[K]V, key K, val V, rwmut *sync.RWMutex) {
	rwmut.Lock()
	defer rwmut.Unlock()

	m[key] = val
}

func mapReadRW[K comparable, V any](m map[K]V, key K, rwmut *sync.RWMutex) (V, bool) {
	rwmut.RLock()
	defer rwmut.RUnlock()

	val, exists := m[key]

	return val, exists
}

func mapDeleteRW[K comparable, V any](m map[K]V, key K, rwmut *sync.RWMutex) bool {
	rwmut.Lock()
	defer rwmut.Unlock()

	_, exists := m[key]
	if exists {
		delete(m, key)
	}

	return exists
}

func mapCopy[K comparable, V any](m map[K]V, mut *sync.Mutex) map[K]V {
	mut.Lock()
	defer mut.Unlock()

	// Create a copy first.
	mCopy := make(map[K]V)
	for k, v := range m {
		mCopy[k] = v
	}

	return mCopy
}

func mapRead[K comparable, V any](m map[K]V, key K, mut *sync.Mutex) (V, bool) {
	mut.Lock()
	defer mut.Unlock()

	val, exists := m[key]

	return val, exists
}

func mapAdd[K comparable, V any](m map[K]V, key K, val V, mut *sync.Mutex) {
	mut.Lock()
	defer mut.Unlock()

	m[key] = val
}

func mapDelete[K comparable, V any](m map[K]V, key K, mut *sync.Mutex) bool {
	mut.Lock()
	defer mut.Unlock()

	_, exists := m[key]
	if exists {
		delete(m, key)
	}

	return exists
}

// Returns the keys as a slice (Useful when the map is used a set).
func mapKeys[K comparable, V any](m map[K]V, mut *sync.Mutex) []K {
	mut.Lock()
	defer mut.Unlock()

	ret := make([]K, len(m))
	i := 0
	for p := range m {
		ret[i] = p
		i++
	}

	return ret
}

func containsElement(arr []string, element string) bool {
	for _, item := range arr {
		if item == element {
			return true
		}
	}
	return false
}

// NoNeighborsError is a soft error, and it is treated different from others.
type NoNeighborsError struct{}

func (e NoNeighborsError) Error() string {
	return "Could not find any neighbor to send message to!"
}

func logAndReturnError(logger zerolog.Logger, err error) error {
	if err != nil && !isTimeout(err) {
		logger.Err(err).Msg(err.Error())
	}

	return err
}

func storeChunk(blobStorage storage.Store, chunk []byte) string {
	hash := crypto.SHA256.New()
	hash.Write(chunk)
	metaHash := hex.EncodeToString(hash.Sum(nil))
	chuckCopy := make([]byte, len(chunk))
	if copy(chuckCopy, chunk) != len(chunk) {
		panic("copy() failed!")
	}
	blobStorage.Set(metaHash, chuckCopy)

	return metaHash
}

func distributeBudget(budget uint, neighbors uint) []uint {
	var i uint
	mod := budget % neighbors
	budgets := make([]uint, neighbors)
	for ; i < mod; i++ {
		budgets[i] = (budget / neighbors) + 1
	}
	for ; i < neighbors; i++ {
		budgets[i] = budget / neighbors
	}

	return budgets
}

func checkDataIntegrity(chunk []byte, metahashExpected string) bool {
	if len(chunk) == 0 {
		return false
	}

	hash := crypto.SHA256.New()
	hash.Write(chunk)
	metaHash := hex.EncodeToString(hash.Sum(nil))

	return metaHash == metahashExpected
}

func hasAllChunks(chunks [][]byte) bool {
	for _, chunk := range chunks {
		if chunk == nil {
			return false
		}
	}

	return true
}

func getNameFirst(blobStorage storage.Store, key string, val []byte) (string, bool) {
	chunk := blobStorage.Get(string(val))
	if chunk != nil {
		metaHashes := strings.Split(string(chunk), peer.MetafileSep)
		chunks := make([][]byte, len(metaHashes))
		for i, metaHash := range metaHashes {
			chunks[i] = blobStorage.Get(metaHash)
		}
		b := hasAllChunks(chunks)
		if b {
			return key, false
		}
	}

	return "", true
}

func createNodeID() (string, *n2ccrypto.PPKModule, [KeyLen]byte) {
	// generate PPKModule to set ID for chord
	var ppkFileName = xid.New().String()
	ppkModulePtr, err := n2ccrypto.NewPKKModule(true, KeyStoragePath+ppkFileName)
	if err != nil {
		panic("Failed to generate PPK Module, likely file RW permissions")
	}

	// nodeID/chordID will be hash of PPK's public key
	var tmp [KeyLen]byte
	copy(tmp[:], n2ccrypto.Hash(n2ccrypto.GetBytesPublicKey(ppkModulePtr.PublicKey)))

	return ppkFileName, ppkModulePtr, tmp
}

// createPostID returns a chord identifier for a post determined by its author and date posted
func createPostID(authorID [KeyLen]byte, date time.Time) (chordID [KeyLen]byte) {

	// Using MarshalBinary
	binaryTime, err := date.MarshalBinary()
	if err != nil {
		panic(fmt.Sprintf("[createPostID] [failed to marshal date: %s", err))
	}

	// Create a byte slice from authorID byte array
	authorIDBytes := authorID[:]

	// Combine authorID and date
	combinedBytes := append(authorIDBytes, binaryTime...)

	// Hash the combined data
	hashedData := n2ccrypto.Hash(combinedBytes)

	// Fit hash data into a [KeyLen]byte
	copy(chordID[:], hashedData)

	return chordID
}

// IsStrictlyBetween returns true if target is strictly between lower and upper: (lower, upper)
func IsStrictlyBetween(target, lower, upper [KeyLen]byte) bool {
	// Inner interval
	if bytes.Compare(lower[:], upper[:]) < 0 {
		return bytes.Compare(target[:], lower[:]) > 0 && bytes.Compare(target[:], upper[:]) < 0
	}
	// Outer interval
	return bytes.Compare(target[:], lower[:]) > 0 || bytes.Compare(target[:], upper[:]) < 0
}

// IsBetweenLeftOpenRightClosed returns true if target is between lower and upper, upper included: (lower, upper]
func IsBetweenLeftOpenRightClosed(target, lower, upper [KeyLen]byte) bool {
	// Inner interval
	if bytes.Compare(lower[:], upper[:]) < 0 {
		return bytes.Compare(target[:], lower[:]) > 0 && bytes.Compare(target[:], upper[:]) <= 0
	}
	// Outer interval
	return bytes.Compare(target[:], lower[:]) > 0 || bytes.Compare(target[:], upper[:]) <= 0
}

// IsBetweenLeftClosedRightOpen returns true if target is between lower and upper, lower included: [lower, upper)
func IsBetweenLeftClosedRightOpen(target, lower, upper [KeyLen]byte) bool {
	// Inner interval
	if bytes.Compare(lower[:], upper[:]) < 0 {
		return bytes.Compare(target[:], lower[:]) >= 0 && bytes.Compare(target[:], upper[:]) < 0
	}
	// Outer interval
	return bytes.Compare(target[:], lower[:]) >= 0 || bytes.Compare(target[:], upper[:]) < 0
}

func AddIntToBytes(array [KeyLen]byte, valueToAdd *big.Int) ([KeyLen]byte, error) {
	// Convert input to Big.Int
	bigIntFromArray := utils.BytesToBigInt(array)
	// Add the other Big.Int
	newValue := new(big.Int).Add(bigIntFromArray, valueToAdd)

	modulus := new(big.Int).Exp(big.NewInt(2), big.NewInt(KeyLen*8), nil)

	moddedValue := new(big.Int).Mod(newValue, modulus)
	// Transform back to [KeyLen]byte
	newByteArray, err := utils.BigIntToBytes(moddedValue)

	return newByteArray, err
}

// SubtractIntFromBytes returns a new byte array given a byte array subtracted by an Int
func SubtractIntFromBytes(array [KeyLen]byte, valueToSub *big.Int) ([KeyLen]byte, error) {
	// Convert input to Big.Int
	bigIntFromArray := utils.BytesToBigInt(array)
	// Sub the other Big.Int
	newValue := new(big.Int).Sub(bigIntFromArray, valueToSub)

	modulus := new(big.Int).Exp(big.NewInt(2), big.NewInt(KeyLen*8), nil)

	moddedValue := new(big.Int).Mod(newValue, modulus)

	// Transform back to [KeyLen]byte
	newByteArray, err := utils.BigIntToBytes(moddedValue)

	return newByteArray, err
}

func shouldValidateMsg(msg *transport.Message) bool {
	return msg.Type == "datarequest" || msg.Type == "searchrequest" || msg.Type == "followrequest"
}

func getChordIDDataRequestMsg(msg *transport.Message) ([KeyLen]byte, error) {
	var dataRequestMessage types.DataRequestMessage
	var peerNodeID [KeyLen]byte

	err := json.Unmarshal(msg.Payload, &dataRequestMessage)
	if err != nil {
		return peerNodeID, err
	}

	return dataRequestMessage.ChordID, nil
}

func getChordIDSearchRequestMsg(msg *transport.Message) ([KeyLen]byte, error) {
	var searchRequestMessage types.SearchRequestMessage
	var peerNodeID [KeyLen]byte

	err := json.Unmarshal(msg.Payload, &searchRequestMessage)
	if err != nil {
		return peerNodeID, err
	}

	return searchRequestMessage.ChordID, nil
}

func getChordIDFollowRequestMsg(msg *transport.Message) ([KeyLen]byte, error) {
	var followRequestMessage types.FollowRequestMessage
	var peerNodeID [KeyLen]byte

	err := json.Unmarshal(msg.Payload, &followRequestMessage)
	if err != nil {
		return peerNodeID, err
	}

	return followRequestMessage.ChordID, nil
}

func getChordIDQueryChordMessage(msg *transport.Message) ([KeyLen]byte, error) {
	var queryChordMessage types.ChordRequestMessage
	var peerNodeID [KeyLen]byte

	err := json.Unmarshal(msg.Payload, &queryChordMessage)
	if err != nil {
		return peerNodeID, err
	}

	return queryChordMessage.ChordNode.ID, nil
}

func getChordID(msg *transport.Message) ([KeyLen]byte, error) {
	if msg.Type == "datarequest" {
		return getChordIDDataRequestMsg(msg)
	} else if msg.Type == "searchrequest" {
		return getChordIDSearchRequestMsg(msg)
	} else if msg.Type == "followrequest" {
		return getChordIDFollowRequestMsg(msg)
	} else if msg.Type == "ChordRequestMessage" {
		return getChordIDQueryChordMessage(msg)
	}

	return [KeyLen]byte{}, nil
}
