//go:build performance
// +build performance

package perf

import (
	"fmt"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/utils"
	"golang.org/x/xerrors"
	"os"
	"strconv"
	"testing"
	"time"
)

func Test_N2C_Chord_Load_Balance(t *testing.T) {
	resetStorage()

	transp := channelFac()

	// Record the start time
	startTimeGenNodes := time.Now()

	// generate nodes
	nbNodes := 5
	nodesArray, _, chordIDsArray, err := GenerateChordNodes(t, nbNodes, transp)
	require.NoError(t, err)

	// Record the end time
	endTimeGenNodes := time.Now()

	// Calculate the duration between start and end time
	durationGenNodes := endTimeGenNodes.Sub(startTimeGenNodes)

	// log the duration
	myLogger().Info().Msgf("[Test_N2C_Chord_Load_Balance] Generating %d nodes and joining them to the network took %s",
		nbNodes, durationGenNodes)

	// print finger tables and pred/succ for each node
	//for i := range nodesArray {
	//	myLogger().Info().Msgf("[Test_N2C_Chord_Load_Balance] [%s] \n %s", nodesArray[i].GetAddr(), nodesArray[i].FingerTableToString())
	//	myLogger().Info().Msgf("[Test_N2C_Chord_Load_Balance] [%s] \n%s", nodesArray[i].GetAddr(), nodesArray[i].PredAndSuccToString())
	//
	//}

	var nodeIDlabels []string

	var postLabels []string
	// labels of nodes before posts
	for i := range chordIDsArray {
		nodeIDlabels = append(nodeIDlabels, fmt.Sprintf("n%d", i+1))
	}

	startTimeGenPosts := time.Now()
	// Generate and Store multiple posts from node 1
	nbPosts := 50
	for i := 0; i < nbPosts; i++ {
		post, err := GenerateAndStorePost(t, nodesArray[0], i+1)
		require.NoError(t, err)

		// prepare labels for ring
		// convert postID str to bigIng
		postIDBytes, err := utils.DecodeChordIDFromStr(post.PostID)
		require.NoError(t, err)

		// add post IDs and labels
		chordIDsArray = append(chordIDsArray, postIDBytes)
		postLabels = append(postLabels, "p"+strconv.Itoa(i+1))
	}

	// Record the end time
	endTimeGenPosts := time.Now()

	// Calculate the duration between start and end time
	durationGenPosts := endTimeGenPosts.Sub(startTimeGenPosts)

	// log the duration
	myLogger().Info().Msgf("[Test_N2C_Chord_Load_Balance] Generating %d posts and storing them in the network took %s",
		nbPosts, durationGenPosts)

	//var nodeAndPostlabels []string
	//nodeAndPostlabels = append(nodeIDlabels, postLabels...)

	// display ring with nodes and posts
	//ring, err := impl.DisplayRing(chordIDsArray, nodeAndPostlabels...)
	//myLogger().Info().Msgf(ring)

	var data []int
	for i := range nodesArray {
		postStore := nodesArray[i].GetStorage().GetPostStore()
		myLogger().Info().Msgf("[Test_N2C_Chord_Load_Balance] [%s] postStore length: %d", nodesArray[i].GetAddr(), postStore.Len())
		data = append(data, postStore.Len())
	}

	myLogger().Info().Msgf("[Test_N2C_Chord_Load_Balance] data: %v", data)

	title := "Chord Load Balance"
	subtitle := fmt.Sprintf("%d posts distributed across %d nodes in a %d byte(s) Chord ID space",
		nbPosts, nbNodes, KeyLen)
	seriesName := "Posts"
	savedFilePath := "./charts/chord_load_balance_chart.html"
	xAxisLabel := "Nodes"
	yAxisLabel := "Number of Posts"
	err = createBarChart(data, nodeIDlabels, xAxisLabel, yAxisLabel, title, subtitle, seriesName, savedFilePath)
	if err != nil {
		t.Fatal(err)
	}
}

func createBarChart(data []int, xTicksLabels []string, xAxisLabel string, yAxisLabel, title string, subtitle string, seriesName string, savedFilePath string) error {
	myLogger().Info().Msgf("[createBarChart] Generating %s chart", title)
	// Create a new bar instance
	bar := charts.NewBar()

	// Set some global options
	bar.SetGlobalOptions(
		charts.WithGridOpts(opts.Grid{
			Left:   "30%", // Increase to reduce plot size
			Right:  "30%", // Increase to reduce plot size
			Top:    "20%", // Increase to reduce plot size
			Bottom: "20%", // Increase to reduce plot size
		}),
		charts.WithTitleOpts(opts.Title{
			Title:    title,
			Subtitle: subtitle,
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name:         xAxisLabel,
			NameLocation: "center",
			AxisLabel: &opts.AxisLabel{
				Margin: 3,
				Show:   true,
			},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name:         yAxisLabel,
			NameLocation: "center",
			AxisLabel: &opts.AxisLabel{
				Margin: -2,
				Show:   true,
			},
		}),
	)

	// Add data sets to the bar chart
	bar.SetXAxis(xTicksLabels).AddSeries(seriesName, generateBarItems(data))

	// Where to save the file
	f, err := os.Create(savedFilePath)
	if err != nil {
		return xerrors.Errorf("[createBarChart] Failed to create the file: %v", err)
	}
	defer f.Close()

	// Render the chart
	err = bar.Render(f)
	if err != nil {
		return xerrors.Errorf("[createBarChart] failed to render the chart: %v", err)
	}

	//openInBrowser(savedFilePath)
	return nil
}

// generateBarItems generates and returns bar items from the provided data
func generateBarItems(data []int) []opts.BarData {
	items := make([]opts.BarData, 0)
	for _, value := range data {
		items = append(items, opts.BarData{Value: float64(value)})
	}
	return items
}

//
//func openInBrowser(filepath string) {
//	var cmd *exec.Cmd
//
//	switch runtime.GOOS {
//	case "linux":
//		cmd = exec.Command("xdg-open", filepath)
//	case "windows":
//		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", filepath)
//	case "darwin": // MacOS
//		cmd = exec.Command("open", "./"+filepath)
//	default:
//		fmt.Println("Unsupported platform")
//		return
//	}
//
//	// Run the command and capture any error
//	err := cmd.Start()
//	if err != nil {
//		fmt.Printf("Failed to open the file in a browser: %v\n", err)
//	} else {
//		fmt.Println("File should now be open in your default browser.")
//	}
//}
