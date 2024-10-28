package impl

import (
	"encoding/json"
	"fmt"
	"go.dedis.ch/cs438/utils"
	"math"
	"math/big"
	"strings"
)

var (
	MIN = big.NewInt(0)                                              // Lower bound of the chord space
	MAX = new(big.Int).Exp(big.NewInt(2), big.NewInt(KeyLen*8), nil) // Upper bound: 2^KeyLen
)

// FingerTableToString outputs a formatted string containing the finger table of the node
func (n *node) FingerTableToString() string {
	out := new(strings.Builder)
	out.WriteString("\n")
	out.WriteString(fmt.Sprintf("Finger Table of node [%s], ", n.Socket.GetAddress()))

	chordIDBigInt := utils.BytesToBigInt(n.chordID)
	out.WriteString(fmt.Sprintf("ChordID: (BigInt)%s (hex)%x (normalized: %.2f):\n",
		chordIDBigInt, n.chordID, normalize(chordIDBigInt, MIN, MAX)))

	idSpaceBigInt := new(big.Int)
	idSpaceBigInt.Exp(big.NewInt(2), big.NewInt(KeyLen*8), nil)
	out.WriteString(fmt.Sprintf("(ID space of 2^%d = %d)\n\n", KeyLen*8, idSpaceBigInt))
	out.WriteString(fmt.Sprintf("%6s | %15s | %8s | %15s | %8s | %12s\n",
		"i", "start.ID", "norm.", "node.ID", "norm.", "node.IPAddr"))
	out.WriteString(fmt.Sprintf("%6s | %15s | %8s | %15s | %8s | %12s\n",
		"----", "--------------", "------", "--------------", "------", "--------------"))
	for i := 0; i < M; i++ {
		fe, err := n.fingerTable.FingerTableGet(i)
		if err != nil {
			outError := new(strings.Builder)
			outError.WriteString(fmt.Sprintf("[FingerTableToString] Error while FingerTableGet: %v", err))
			return outError.String()
		}
		startBigInt := utils.BytesToBigInt(fe.Start.ID)
		nodeBigInt := utils.BytesToBigInt(fe.Node.ID)
		out.WriteString(fmt.Sprintf("%6d | 0x%x ~ %8d | %8.2f | 0x%x ~ %8d | %8.2f | %12s\n",
			i, fe.Start.ID, startBigInt, normalize(startBigInt, MIN, MAX), fe.Node.ID, nodeBigInt,
			normalize(nodeBigInt, MIN, MAX), fe.Node.IPAddr))
	}
	return out.String()
}

// PredAndSuccToString outputs a formatted string containing the predecessor and successor of current node
func (n *node) PredAndSuccToString() string {
	chordIDBigInt := utils.BytesToBigInt(n.chordID)
	predecessorBigInt := utils.BytesToBigInt(n.predecessorChord.ID)
	successorBigInt := utils.BytesToBigInt(n.MySuccessor().ID)

	out := new(strings.Builder)
	out.WriteString("\n")
	out.WriteString(fmt.Sprintf("%12s | %12s | %12s | %12s\n",
		"", "myPredessor", "myChordID", "MySuccessor"))
	out.WriteString(fmt.Sprintf("%12s | %12s | %12s | %12s\n",
		"------------", "------------", "------------", "------------"))
	out.WriteString(fmt.Sprintf("%12s | %12s | %12s | %12s\n",
		"IPAddr", n.predecessorChord.IPAddr, n.Socket.GetAddress(), n.MySuccessor().IPAddr))
	out.WriteString(fmt.Sprintf("%12s | %12s | %12s | %12s\n",
		"ChordID", predecessorBigInt, chordIDBigInt, successorBigInt))
	out.WriteString(fmt.Sprintf("%12s | %12x | %12x | %12x\n",
		"ChordIDHex", n.predecessorChord.ID, n.chordID, n.MySuccessor().ID))
	out.WriteString(fmt.Sprintf("%12s | %12f | %12f | %12f\n",
		"Normalized", normalize(predecessorBigInt, MIN, MAX),
		normalize(chordIDBigInt, MIN, MAX), normalize(successorBigInt, MIN, MAX)))
	out.WriteString("\n")
	return out.String()
}

// DisplayRing visualizes nodeIDs on a circular ring within the specified range and radius.
// (implemented with the help of ChatGPT)
// func DisplayRing(radius int, bigInts []*big.Int) error {
func DisplayRing(chordIDs [][KeyLen]byte, labels ...string) (string, error) {
	out := new(strings.Builder)

	var radius = 5 // radius of the circle in the terminal
	var bigInts []*big.Int

	idSpaceBigInt := new(big.Int)
	idSpaceBigInt.Exp(big.NewInt(2), big.NewInt(KeyLen*8), nil)
	out.WriteString(fmt.Sprintf("\nChord Ring (ID space of 2^%d = %d):\n\n", KeyLen*8, idSpaceBigInt))

	// Create a slice of big.Ints from the input
	for _, chordID := range chordIDs {
		chordIDbytes := utils.BytesToBigInt(chordID)
		bigInts = append(bigInts, chordIDbytes)
	}

	// Check if any value is out of range
	for _, n := range bigInts {
		if n.Cmp(MAX) >= 0 || n.Cmp(MIN) < 0 {
			return "", fmt.Errorf("[DisplayRing] value out of range: %v", n)
		}
	}

	// Use provided labels or default to "n1", "n2", etc.
	useDefaultLabels := len(labels) == 0
	if useDefaultLabels {
		for i := range bigInts {
			labels = append(labels, fmt.Sprintf("n%d", i+1))
		}
	}

	// Pre-calculate the circle points and their corresponding angles
	circlePoints := make(map[[2]int]float64)
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			if onCircle(x, y, radius) {
				angle := CalcAngle(y, x)
				circlePoints[[2]int{x, y}] = angle
			}
		}
	}

	// Find the closest point on the circle for each label
	closestPoints := findClosestPointsOnRing(bigInts, circlePoints)

	// Display the ring with labels
	ringWithLabelsString := ringWithLabelsToString(radius, closestPoints, labels)
	out.WriteString(ringWithLabelsString)

	out.WriteString("\nChordIDs:\n")
	for i, bigInt := range bigInts {
		out.WriteString(fmt.Sprintf("Node %5s: %8s (0x%x) (normalized: %.2f)\n",
			labels[i], bigInt.String(), chordIDs[i], normalize(bigInt, MIN, MAX)))

	}
	out.WriteString("\n")
	return out.String(), nil
}

// findClosestPointsOnRing Return a map that stores the closest point for each label
func findClosestPointsOnRing(bigInts []*big.Int, circlePoints map[[2]int]float64) map[int][2]int {
	closestPoints := make(map[int][2]int)
	for i, n := range bigInts {
		normalizedValue := normalize(n, MIN, MAX)
		labelAngle := normalizedValue * 2.0 * math.Pi
		var closestPoint [2]int
		smallestDiff := math.MaxFloat64
		for point, angle := range circlePoints {
			diff := math.Abs(angle - labelAngle)
			if diff > math.Pi {
				diff = 2*math.Pi - diff
			}
			if diff < smallestDiff {
				smallestDiff = diff
				closestPoint = point
			}
		}
		closestPoints[i] = closestPoint
	}
	return closestPoints
}

func ringWithLabelsToString(radius int, closestPoints map[int][2]int, labels []string) string {
	out := new(strings.Builder)
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			if onCircle(x, y, radius) {
				labeled := false
				for i, point := range closestPoints {
					if point == [2]int{x, y} {
						label := labels[i] // Use the appropriate label
						out.WriteString(fmt.Sprintf("%s ", label))
						labeled = true
						break
					}
				}
				if !labeled {
					out.WriteString(". ")
				}
			} else {
				out.WriteString("  ")
			}
		}
		out.WriteString("\n")
	}
	return out.String()
}

func CalcAngle(y int, x int) float64 {
	angle := math.Atan2(float64(y), float64(x)) + math.Pi/2
	if angle < 0 {
		angle += 2 * math.Pi
	} else if angle > 2*math.Pi {
		angle -= 2 * math.Pi
	}
	return angle
}

// onCircle determines if a given point (x, y) lies on the circumference of a circle with a specified radius.
// (implemented with the help of ChatGPT)
func onCircle(x, y, radius int) bool {
	return x*x+y*y >= radius*radius-1 && x*x+y*y <= radius*radius+1
}

// normalize converts a big.Int value to a float64 representing its normalized position within a given range.
// (implemented with the help of ChatGPT)
func normalize(n, min, max *big.Int) float64 {
	// Convert big.Int to big.Float
	nFloat := new(big.Float).SetInt(n)
	minFloat := new(big.Float).SetInt(min)
	maxFloat := new(big.Float).SetInt(max)

	// Calculate the normalized value
	rangeFloat := new(big.Float).Sub(maxFloat, minFloat)
	normalizedFloat := new(big.Float).Sub(nFloat, minFloat)
	normalizedFloat.Quo(normalizedFloat, rangeFloat)

	// Convert to float64
	normalizedValue, _ := normalizedFloat.Float64()
	return normalizedValue
}

// PostCatalogToString outputs a formatted string containing the of the PostCatalog of the node
func (n *node) PostCatalogToString() string {
	out := new(strings.Builder)
	out.WriteString("\n\n")
	out.WriteString(fmt.Sprintf("Post Catalog of node [%s]:\n\n", n.Socket.GetAddress()))
	out.WriteString(fmt.Sprintf("%12s | %s \n", "AuthorID", "PostIDs"))
	out.WriteString(fmt.Sprintf("%12s | %s \n", "------------", "------------------------"))
	n.Storage.GetPostCatalogStore().ForEach(func(authorID string, postIDsByAuthorBytes []byte) bool {
		var postIDsByAuthor map[string]struct{}
		if postIDsByAuthorBytes == nil {
			postIDsByAuthor = make(map[string]struct{}, 0)
		} else {
			err := json.Unmarshal(postIDsByAuthorBytes, &postIDsByAuthor)
			if err != nil {
				n.Error().Msgf("[PostCatalogToString] failed to unmarshal postIDsByAuthorBytes: %v", err)
			}
		}
		out.WriteString(fmt.Sprintf("%12s | ", authorID))
		for postIDbyAuthor := range postIDsByAuthor {
			out.WriteString(fmt.Sprintf("%s, ", postIDbyAuthor))
		}
		out.WriteString("\n")
		return true
	})
	out.WriteString("\n")
	return out.String()
}
