package impl

import (
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
	"math/big"
	"sync"
)

type SafeFingerTable struct {
	mutex       sync.RWMutex
	fingerTable []peer.FingerEntry
	size        int
}

func NewSafeFingerTable(size int, chordID [KeyLen]byte) (SafeFingerTable, error) {
	fingerTable := make([]peer.FingerEntry, size)
	var err error

	for i := 1; i <= size; i++ {
		bigInt := new(big.Int)
		bigInt = bigInt.Exp(big.NewInt(2), big.NewInt(int64(i-1)), nil)
		fingerTable[i-1].Start.ID, err = AddIntToBytes(chordID, bigInt)
		if err != nil {
			return SafeFingerTable{}, xerrors.Errorf("[NewSafeFingerTable] Failed to generate finger table: %v", err)
		}
	}
	return SafeFingerTable{sync.RWMutex{}, fingerTable, size}, nil
}

func (sft *SafeFingerTable) FingerTableGet(index int) (peer.FingerEntry, error) {
	sft.mutex.RLock()
	defer sft.mutex.RUnlock()
	if index >= sft.size {
		return peer.FingerEntry{}, xerrors.Errorf("[FingerTableSetStart] Out of bound Index. Attempted "+
			"to reach [%d] in an array of size [%d]", index, sft.size)
	}
	return sft.fingerTable[index], nil
}

func (sft *SafeFingerTable) FingerTableSetStart(index int, chordNode types.ChordNode) error {
	sft.mutex.Lock()
	defer sft.mutex.Unlock()
	if index >= sft.size {
		return xerrors.Errorf("[FingerTableSetStart] Out of bound Index. Attempted to reach [%d] in an array "+
			"of size [%d]", index, sft.size)
	}
	sft.fingerTable[index].Start = chordNode
	return nil
}

func (sft *SafeFingerTable) FingerTableSetNode(index int, chordNode types.ChordNode) error {
	sft.mutex.Lock()
	defer sft.mutex.Unlock()
	if index >= sft.size {
		return xerrors.Errorf("[FingerTableSetNode] Out of bound Index. Attempted to reach [%d] in an array "+
			"of size [%d]", index, sft.size)
	}
	sft.fingerTable[index].Node = chordNode
	return nil
}

func (sft *SafeFingerTable) FingerTableAssignMinusOne(index int) error {
	sft.mutex.Lock()
	defer sft.mutex.Unlock()
	if index >= sft.size {
		return xerrors.Errorf("[FingerTableSetNode] Out of bound Index. Attempted to reach [%d] in an array "+
			"of size [%d]", index, sft.size)
	}

	sft.fingerTable[index].Node = sft.fingerTable[index-1].Node
	return nil
}
