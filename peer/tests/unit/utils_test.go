package unit

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/peer/impl"
	"go.dedis.ch/cs438/utils"
	"math/big"
	"testing"
)

func Test_N2C_Utils_SubtractFromByteArraySimple(t *testing.T) {
	bArray := [KeyLen]byte(make([]byte, KeyLen))
	bArrayBis := [KeyLen]byte(make([]byte, KeyLen))
	bArrayBis[KeyLen-1] = 1
	bArray[KeyLen-1] = 255
	myLogger().Info().Msgf("my array: %v", bArrayBis)

	mynumb := new(big.Int)
	mynumb.SetUint64(254)
	newBArray, _ := impl.AddIntToBytes(bArrayBis, mynumb)
	myLogger().Info().Msgf("my array(expected): %v", bArray)
	myLogger().Info().Msgf("my array(gotten): %v", newBArray)
	require.Equal(t, bArray, newBArray)
}

func Test_N2C_Utils_ConvertFromByteArrayComplex(t *testing.T) {
	// Prepare expected result
	expectedByteArray := [KeyLen]byte(make([]byte, KeyLen))
	expectedByteArray[KeyLen-1] = 255
	fmt.Printf("expectedByteArray: %08b\n", expectedByteArray)
	// ======

	// Prepare input
	inputByteArray := [KeyLen]byte(make([]byte, KeyLen))
	inputByteArray[KeyLen-1] = 1
	fmt.Printf("inputByteArray: %08b\n", inputByteArray)
	// ======

	// Added Number
	mynumb := new(big.Int)
	mynumb.SetString("254", 10)

	// Convert input to Big.Int
	inputBigInt := utils.BytesToBigInt(inputByteArray)
	fmt.Printf("inputBigInt: %d\n", inputBigInt)

	// Add the other Big.Int
	added := new(big.Int).Add(inputBigInt, mynumb)
	fmt.Printf("added: %d\n", added)

	// Transform back to [KeyLen]byte
	newBArray, _ := utils.BigIntToBytes(added)
	fmt.Printf("newBArray: %d\n", newBArray)

	myLogger().Info().Msgf("my array(expected): %v", expectedByteArray)
	myLogger().Info().Msgf("my array(gotten): %v", newBArray)
	require.Equal(t, expectedByteArray, newBArray)
}

func Test_N2C_Utils_AddFromByteArrayComplex(t *testing.T) {
	// Prepare expected result
	expectedByteArray := [KeyLen]byte(make([]byte, KeyLen))
	expectedByteArray[KeyLen-1] = 0
	// ======

	// Prepare input
	inputByteArray := [KeyLen]byte(make([]byte, KeyLen))
	inputByteArray[KeyLen-1] = 1
	// ======

	// Added Number
	addedBigInt := new(big.Int)
	addedBigInt.SetString("255", 10)

	newBArray, _ := impl.AddIntToBytes(inputByteArray, addedBigInt)

	myLogger().Info().Msgf("my array(expected): %v", expectedByteArray)
	myLogger().Info().Msgf("my array(gotten): %v", newBArray)
	require.Equal(t, expectedByteArray, newBArray)
}

func Test_N2C_Utils_SubFromByteArrayComplex(t *testing.T) {
	// Prepare input
	expectedByteArray := [KeyLen]byte(make([]byte, KeyLen))
	expectedByteArray[KeyLen-1] = 255
	// ======

	// Prepare expected result
	inputByteArray := [KeyLen]byte(make([]byte, KeyLen))
	inputByteArray[KeyLen-1] = 1
	// ======

	// Subtracted Number
	mynumb := new(big.Int)
	mynumb.SetString("254", 10)

	newBArray, _ := impl.SubtractIntFromBytes(expectedByteArray, mynumb)

	myLogger().Info().Msgf("my array(expected): %v", inputByteArray)
	myLogger().Info().Msgf("my array(gotten): %v", newBArray)
	require.Equal(t, inputByteArray, newBArray)
}

func Test_N2C_Utils_AddFromByteArrayComplexOverflow(t *testing.T) {
	// Prepare expected result
	expectedByteArray := [KeyLen]byte(make([]byte, KeyLen))
	// ======

	// Prepare input
	inputByteArray := [KeyLen]byte(make([]byte, KeyLen))
	for i := range inputByteArray {
		inputByteArray[i] = 255
	}
	// ======

	// Added Number
	mynumb := big.NewInt(1)

	newBArray, _ := impl.AddIntToBytes(inputByteArray, mynumb)

	myLogger().Info().Msgf("my array(Initial): %v", inputByteArray)
	myLogger().Info().Msgf("my array(expected): %v", expectedByteArray)
	myLogger().Info().Msgf("my array(gotten): %v", newBArray)
	require.Equal(t, expectedByteArray, newBArray)
}

func Test_N2C_Utils_SubFromByteArrayComplexOverflow(t *testing.T) {
	// Prepare input
	inputByteArray := [KeyLen]byte(make([]byte, KeyLen))
	// ======

	// Prepare expected result
	expectedByteArray := [KeyLen]byte(make([]byte, KeyLen))
	for i := range expectedByteArray {
		expectedByteArray[i] = 255
	}
	// ======

	// Added Number
	mynumb := big.NewInt(1)

	newBArray, _ := impl.SubtractIntFromBytes(inputByteArray, mynumb)

	myLogger().Info().Msgf("my array(Initial): %v", inputByteArray)
	myLogger().Info().Msgf("my array(expected): %v", expectedByteArray)
	myLogger().Info().Msgf("my array(gotten): %v", newBArray)
	require.Equal(t, expectedByteArray, newBArray)
}

// WE ASSUME WORKING MOD len([KeyLen]byte)
func Test_N2C_Utils_IsStrictlyBetween(t *testing.T) {
	// Test case 1 - is 4 strictly between 2 and 5 - expected True
	numb1 := big.NewInt(2)
	numb2 := big.NewInt(5)

	x1 := big.NewInt(4)

	b32_1, _ := utils.BigIntToBytes(x1)
	b32_2, _ := utils.BigIntToBytes(numb1)
	b32_3, _ := utils.BigIntToBytes(numb2)
	res := impl.IsStrictlyBetween(b32_1, b32_2, b32_3)

	require.True(t, res)

	// Test case 2 - is 6 strictly between 2 and 5 - expected False
	numb1 = big.NewInt(2)
	numb2 = big.NewInt(5)

	x1 = big.NewInt(6)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsStrictlyBetween(b32_1, b32_2, b32_3)

	require.False(t, res)

	// Test case 3 - is 6 strictly between 5 and 2 - expected True
	numb1 = big.NewInt(5)
	numb2 = big.NewInt(2)

	x1 = big.NewInt(6)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsStrictlyBetween(b32_1, b32_2, b32_3)

	require.True(t, res)

	// Test case 4 - is 4 strictly between 5 and 2 - expected False
	numb1 = big.NewInt(5)
	numb2 = big.NewInt(2)

	x1 = big.NewInt(4)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsStrictlyBetween(b32_1, b32_2, b32_3)

	require.False(t, res)

	// Test case 5 - is 4 strictly between 5 and 2 - expected False
	numb1 = big.NewInt(5)
	numb2 = big.NewInt(2)

	x1 = big.NewInt(5)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsStrictlyBetween(b32_1, b32_2, b32_3)

	require.False(t, res)

	// Test case 6 - is 4 strictly between 5 and 2 - expected False
	numb1 = big.NewInt(5)
	numb2 = big.NewInt(2)

	x1 = big.NewInt(2)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsStrictlyBetween(b32_1, b32_2, b32_3)

	require.False(t, res)
}

func Test_N2C_Utils_IsBetweenLeftClosedRightOpen(t *testing.T) {
	// Test case 1 - is 4 between 2 and 5 - expected True
	numb1 := big.NewInt(2)
	numb2 := big.NewInt(5)

	x1 := big.NewInt(4)

	b32_1, _ := utils.BigIntToBytes(x1)
	b32_2, _ := utils.BigIntToBytes(numb1)
	b32_3, _ := utils.BigIntToBytes(numb2)
	res := impl.IsBetweenLeftClosedRightOpen(b32_1, b32_2, b32_3)

	require.True(t, res)

	// Test case 2 - is 6 between 2 and 5 - expected False
	numb1 = big.NewInt(2)
	numb2 = big.NewInt(5)

	x1 = big.NewInt(6)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsBetweenLeftClosedRightOpen(b32_1, b32_2, b32_3)

	require.False(t, res)

	// Test case 3 - is 6 between 5 and 2 - expected True
	numb1 = big.NewInt(5)
	numb2 = big.NewInt(2)

	x1 = big.NewInt(6)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsBetweenLeftClosedRightOpen(b32_1, b32_2, b32_3)

	require.True(t, res)

	// Test case 4 - is 4 between 5 and 2 - expected False
	numb1 = big.NewInt(5)
	numb2 = big.NewInt(2)

	x1 = big.NewInt(4)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsBetweenLeftClosedRightOpen(b32_1, b32_2, b32_3)

	require.False(t, res)

	// Test case 5 - is 2 between 2 and 5 - expected True
	numb1 = big.NewInt(2)
	numb2 = big.NewInt(5)

	x1 = big.NewInt(2)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsBetweenLeftClosedRightOpen(b32_1, b32_2, b32_3)

	require.True(t, res)

	// Test case 6 - is 5 between 2 and 5 - expected False
	numb1 = big.NewInt(2)
	numb2 = big.NewInt(5)

	x1 = big.NewInt(5)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsBetweenLeftClosedRightOpen(b32_1, b32_2, b32_3)

	require.False(t, res)

	// Test case 7 - is 5 between 5 and 2 - expected True
	numb1 = big.NewInt(5)
	numb2 = big.NewInt(2)

	x1 = big.NewInt(5)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsBetweenLeftClosedRightOpen(b32_1, b32_2, b32_3)

	require.True(t, res)

	// Test case 8 - is 2 between 5 and 2 - expected False
	numb1 = big.NewInt(5)
	numb2 = big.NewInt(2)

	x1 = big.NewInt(2)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsBetweenLeftClosedRightOpen(b32_1, b32_2, b32_3)

	require.False(t, res)

}

func Test_N2C_Utils_IsBetweenLeftOpenRightClosed(t *testing.T) {
	// Test case 1 - is 4 between 2 and 5 - expected True
	numb1 := big.NewInt(2)
	numb2 := big.NewInt(5)

	x1 := big.NewInt(4)

	b32_1, _ := utils.BigIntToBytes(x1)
	b32_2, _ := utils.BigIntToBytes(numb1)
	b32_3, _ := utils.BigIntToBytes(numb2)
	res := impl.IsBetweenLeftOpenRightClosed(b32_1, b32_2, b32_3)

	require.True(t, res)

	// Test case 2 - is 6 between 2 and 5 - expected False
	numb1 = big.NewInt(2)
	numb2 = big.NewInt(5)

	x1 = big.NewInt(6)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsBetweenLeftOpenRightClosed(b32_1, b32_2, b32_3)

	require.False(t, res)

	// Test case 3 - is 6 between 5 and 2 - expected True
	numb1 = big.NewInt(5)
	numb2 = big.NewInt(2)

	x1 = big.NewInt(6)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsBetweenLeftOpenRightClosed(b32_1, b32_2, b32_3)

	require.True(t, res)

	// Test case 4 - is 4 between 5 and 2 - expected False
	numb1 = big.NewInt(5)
	numb2 = big.NewInt(2)

	x1 = big.NewInt(4)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsBetweenLeftOpenRightClosed(b32_1, b32_2, b32_3)

	require.False(t, res)

	// Test case 5 - is 2 between 2 and 5 - expected False
	numb1 = big.NewInt(2)
	numb2 = big.NewInt(5)

	x1 = big.NewInt(2)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsBetweenLeftOpenRightClosed(b32_1, b32_2, b32_3)

	require.False(t, res)

	// Test case 6 - is 5 between 2 and 5 - expected True
	numb1 = big.NewInt(2)
	numb2 = big.NewInt(5)

	x1 = big.NewInt(5)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsBetweenLeftOpenRightClosed(b32_1, b32_2, b32_3)

	require.True(t, res)

	// Test case 7 - is 5 between 5 and 2 - expected False
	numb1 = big.NewInt(5)
	numb2 = big.NewInt(2)

	x1 = big.NewInt(5)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsBetweenLeftOpenRightClosed(b32_1, b32_2, b32_3)

	require.False(t, res)

	// Test case 8 - is 2 between 5 and 2 - expected True
	numb1 = big.NewInt(5)
	numb2 = big.NewInt(2)

	x1 = big.NewInt(2)

	b32_1, _ = utils.BigIntToBytes(x1)
	b32_2, _ = utils.BigIntToBytes(numb1)
	b32_3, _ = utils.BigIntToBytes(numb2)
	res = impl.IsBetweenLeftOpenRightClosed(b32_1, b32_2, b32_3)

	require.True(t, res)
}
