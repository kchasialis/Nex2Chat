package unit

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/peer/impl"
	"go.dedis.ch/cs438/utils"
	"math/big"
	"testing"
)

// Test_N2C_UI tests the display of big integers on a Chord ring with a predefined range.
func Test_N2C_UI(t *testing.T) {
	// Big integers to be placed on the ring
	n1 := new(big.Int)
	n1.Exp(big.NewInt(2), big.NewInt(1), nil)

	n2 := new(big.Int)
	n2.Exp(big.NewInt(2), big.NewInt(KeyLen*3), nil)
	n2.Mul(n2, big.NewInt(7))

	var arrayOfBytes [][KeyLen]byte
	b32_1, err := utils.BigIntToBytes(n1)
	require.NoError(t, err)
	b32_2, err := utils.BigIntToBytes(n2)
	require.NoError(t, err)

	// make a slice of [32]byte with the two byteArrays
	arrayOfBytes = append(arrayOfBytes, b32_1)
	arrayOfBytes = append(arrayOfBytes, b32_2)

	//fmt.Println(arrayOfBytes)

	ring, err := impl.DisplayRing(arrayOfBytes)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	myLogger().Info().Msgf(ring)

}
