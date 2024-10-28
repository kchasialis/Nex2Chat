package utils

import (
	"encoding/hex"
	"encoding/json"
	"go.dedis.ch/cs438/n2ccrypto"
	"golang.org/x/xerrors"
	"math/big"
)

const KeyLen = n2ccrypto.DefaultKeyLen

func BytesToBigInt(array [KeyLen]byte) *big.Int {
	return new(big.Int).SetBytes(array[:])
}

func BigIntToBytes(number *big.Int) ([KeyLen]byte, error) {
	newByteArray := number.Bytes()
	arraylen := len(newByteArray)
	if arraylen > KeyLen {
		return [KeyLen]byte{}, xerrors.Errorf("[BigIntToBytes] Max [KeyLen]Byte size exceeded")
	}

	var byteArray [KeyLen]byte
	copy(byteArray[KeyLen-arraylen:], newByteArray)
	return byteArray, nil
}

func EncodeChordIDToStr(postID [KeyLen]byte) string {
	return hex.EncodeToString(postID[:])
}

func DecodeChordIDFromStr(s string) ([KeyLen]byte, error) {
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return [KeyLen]byte{}, xerrors.Errorf("[DecodeChordIDFromStr] failed to decode string: %v", err)
	}

	var postID [KeyLen]byte
	copy(postID[:], bytes[:KeyLen])
	return postID, nil
}

func StrToBigInt(s string) (*big.Int, error) {
	bytes, err := DecodeChordIDFromStr(s)
	if err != nil {
		return new(big.Int), xerrors.Errorf("[DecodeChordIDFromStr] failed to decode string: %v", err)
	}
	return BytesToBigInt(bytes), nil
}

func ByteArrayToMapStringStruct(array []byte) (map[string]struct{}, error) {
	var mapStringStruct = make(map[string]struct{})
	if array != nil {
		err := json.Unmarshal(array, &mapStringStruct)
		if err != nil {
			return nil, xerrors.Errorf("[ByteArrayToMapStringStruct] Error while unmarshal json: %v", err)
		}
	}
	return mapStringStruct, nil
}
