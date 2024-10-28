// Simple CLI to run a node
package main

import (
	"bufio"
	fm "fmt"
	"go.dedis.ch/cs438/storage/file"
	"os"
	"strings"
	"sync"
	"time"

	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/peer/impl"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/types"

	"go.dedis.ch/cs438/transport/udp"
)

const visibilityAll = "all"
const visibilityPublic = "public"
const visibilityPrivate = "private"

func main() {
	// setup peer
	trans := udp.NewUDP()
	sock, err := trans.CreateSocket("127.0.0.1:0")
	if err != nil {
		fm.Printf(impl.Red("failed to create socket %s"), err)
	}
	storage, err := file.NewPersistency("./filestorage/")
	if err != nil {
		fm.Printf(impl.Red("failed to init the storage %s"), err)
	}
	uilock := sync.Mutex{}
	followRequestCh := make(chan string)
	followReplyCh := make(chan string)
	conf := peer.Configuration{
		Socket:              sock,
		MessageRegistry:     standard.NewRegistry(),
		AntiEntropyInterval: 10 * time.Second,
		HeartbeatInterval:   1 * time.Minute,
		AckTimeout:          5 * time.Second,
		ContinueMongering:   0.5,
		ChunkSize:           8192,
		BackoffDataRequest: peer.Backoff{
			Initial: 2 * time.Second,
			Factor:  2,
			Retry:   5,
		},
		Storage: storage,
		// We don’t use paxos so those parameters are irrelevant
		TotalPeers: 1,
		PaxosThreshold: func(u uint) int {
			return 0
		},
		PaxosID:                 0,
		PaxosProposerRetry:      time.Second,
		RequestValidation:       true,
		WithChord:               true,
		ChallengeTimeout:        10 * time.Second,
		FollowRequestUserPrompt: makeFollowRequestUserPrompt(followRequestCh, followReplyCh, &uilock),
		FollowReplyDisplay:      makeFollowReplyDisplay(&uilock),
		PostEncryption:          true,
		FlagThreshold:           10,
		RPCRetry:                5,
	}
	node := impl.NewPeer(conf)
	err = node.Start()
	if err != nil {
		fm.Printf(impl.Red("Failed to start node %s"), err)
		return
	}
	fm.Println(impl.Green("Welcome on Nex2Chat!\n"))
	loop := true
	reader := bufio.NewReader(os.Stdin)
	whoami(node)
	fm.Println("\nType 'help' to see available commands.")
	for loop {
		loop = readUserCommand(node, *reader, followRequestCh, followReplyCh, &uilock)
	}
	fm.Println(impl.Green("Bye!"))
	err = node.Stop()
	if err != nil {
		fm.Printf(impl.Red("Error stopping node %s"), err)
	}
	sock.Close()
}

func makeFollowRequestUserPrompt(
	followRequestCh chan string,
	followReplyCh chan string,
	uilock *sync.Mutex,
) func(string) bool {
	return func(peer string) bool {
		uilock.Lock()
		defer uilock.Unlock()
		fm.Printf(
			impl.Blue("\rAccept follow request from user %s (Y(es) / N(o))? "),
			peer,
		)
		followRequestCh <- peer
		answer := <-followReplyCh
		answer = strings.ToLower(answer)
		accept := answer == "yes" || answer == "y"
		return accept
	}
}

func makeFollowReplyDisplay(uilock *sync.Mutex) func(string) {
	return func(peer string) {
		uilock.Lock()
		defer uilock.Unlock()
		fm.Printf(impl.Green("\rUser %s accepted our follow request!\n"), peer)
		fm.Print("> ")
	}
}

func readUserCommand(node peer.Peer, reader bufio.Reader,
	followRequestCh chan string, followReplyCh chan string, uilock *sync.Mutex) bool {

	fm.Print("> ")

	command, err := reader.ReadString('\n')
	command = strings.TrimSpace(command)
	if err != nil {
		// exit on EOF (Ctrl-D)
		return false
	}

	select {
	case <-followRequestCh:
		followReplyCh <- command
		return true
	default:
	}

	uilock.Lock()
	defer uilock.Unlock()
	quit, err := handleCommand(node, reader, command)
	if err != nil {
		fm.Printf(impl.Red("Error: %s\n"), err)
	}

	return quit
}

func handleCommand(node peer.Peer, reader bufio.Reader, command string) (bool, error) {
	var err error

	switch command {
	case "whoami":
		whoami(node)
	case "add":
		err = addPeer(node, reader)
	case "neighbors":
		showNeighbors(node)
	case "new":
		err = joinChordNetwork(node, reader, true)
	case "join":
		err = joinChordNetwork(node, reader, false)
	case "post":
		err = post(node, reader)
	case "feed":
		err = getFeed(node, reader)
	case "like":
		err = likePost(node, reader)
	case "comment":
		err = commentPost(node, reader)
	case "flag":
		err = flagPost(node, reader)
	case "follow":
		err = follow(node, reader)
	case "unfollow":
		err = unfollow(node, reader)
	case "following":
		following(node)
	case "followers":
		followers(node)
	case "block":
		err = block(node, reader)
	case "quit":
		return false, err
	case "finger":
		showFingerTable(node)
	case "help":
		help()
	case "":
	default:
		fm.Printf(impl.Red("Unknown command '%s'\n"), command)
	}

	return true, err
}

func help() {
	fm.Print(
		`Available commands:
   ===== INFO =====
   whoami - show information about myself
   help   - display this help message

   ===== ROUTING NETWORK =====
   add - add a peer
   neighbors - display your neighbors

   ===== CHORD NETWORK =====
   new  - create a new chord network
   join - join the chord network

   ===== POSTING =====
   post - write a new post
   feed - display your feed
   flag - flag a post as inappropriate
   like - like a post
   comment - add a comment to a post

   ===== FOLLOWING =====
   follow    - follow someone to see their posts
   unfollow  - stop following someone
   following - display the nodes you follow
   followers - display your followers
   block     - block a user (prevents them from following you)

   ===== CLIENT =====
   quit - exit nex2chat

   ===== DEBUG =====
   finger - display your local finger table
`)
}

func whoami(node peer.Peer) {
	fm.Printf("You are connected on %s\n", node.GetAddress())
	fm.Printf("Your chord id is %s\n", node.GetNodeIDStr())
}

func post(node peer.Peer, reader bufio.Reader) error {
	fm.Printf(impl.Blue("Post visibility [%s / %s]: "), visibilityPublic, visibilityPrivate)
	visibility, err := reader.ReadString('\n')
	visibility = strings.ToLower(strings.TrimSpace(visibility))
	if err != nil {
		return err
	}
	if visibility == "" {
		visibility = visibilityPublic
	}
	isPublic := visibility == visibilityPublic
	isPrivate := visibility == visibilityPrivate
	if !(isPublic || isPrivate) {
		return fm.Errorf("invalid post visibility")
	}
	fm.Printf(impl.Blue("Post content (%s): "), visibility)
	message, err := reader.ReadString('\n')
	message = strings.TrimSpace(message)
	if err != nil {
		return err
	}
	post := node.CreatePost([]byte(message))
	if isPublic {
		err = node.StorePublicPost(post)
	} else {
		err = node.StorePrivatePost(post)
	}
	if err != nil {
		return err
	}
	fm.Println(impl.Green("Post created sucessfuly!"))
	fm.Print(post)
	return nil
}

func getFeed(node peer.Peer, reader bufio.Reader) error {
	fm.Printf(impl.Blue("Feed to display [%s / %s / %s]: "), visibilityAll, visibilityPublic, visibilityPrivate)
	visibility, err := reader.ReadString('\n')
	visibility = strings.ToLower(strings.TrimSpace(visibility))
	if err != nil {
		return err
	}
	if visibility == "" {
		visibility = visibilityAll
	}
	showPublic := visibility == "public" || visibility == visibilityAll
	showPrivate := visibility == "private" || visibility == visibilityAll
	if !(showPublic || showPrivate) {
		return fm.Errorf("invalid feed category")
	}
	var feed []types.Post
	if showPublic {
		publicFeed, err := node.GetPublicFeed()
		if err != nil {
			return err
		}
		feed = append(feed, publicFeed...)
	}
	if showPrivate {
		privateFeed, err := node.GetPrivateFeed()
		if err != nil {
			return err
		}
		feed = append(feed, privateFeed...)
	}
	if len(feed) == 0 {
		fm.Println("No feed to display yet.")
	}
	for _, post := range feed {
		fm.Printf("%s\n", post)
	}
	return nil
}

func likePost(node peer.Peer, reader bufio.Reader) error {
	fm.Print(impl.Blue("PostID to like: "))
	post, err := reader.ReadString('\n')
	post = strings.TrimSpace(post)
	if err != nil {
		return err
	}
	err = node.Like(post)
	if err != nil {
		return err
	}
	fm.Printf(impl.Green("Post %s has been liked.\n"), post)
	return nil
}

func commentPost(node peer.Peer, reader bufio.Reader) error {
	fm.Print(impl.Blue("PostID to comment: "))
	post, err := reader.ReadString('\n')
	post = strings.TrimSpace(post)
	if err != nil {
		return err
	}
	fm.Print(impl.Blue("Your comment: "))
	comment, err := reader.ReadString('\n')
	comment = strings.TrimSpace(comment)
	if err != nil {
		return err
	}
	err = node.Comment(post, comment)
	if err != nil {
		return err
	}
	fm.Printf(impl.Green("Comment added on post %s.\n"), post)
	return nil
}

func flagPost(node peer.Peer, reader bufio.Reader) error {
	fm.Print(impl.Blue("PostID to flag: "))
	post, err := reader.ReadString('\n')
	post = strings.TrimSpace(post)
	if err != nil {
		return err
	}
	err = node.FlagPost(post)
	if err != nil {
		return err
	}
	fm.Printf(impl.Green("Post %s has been flagged.\n"), post)
	return nil
}

func follow(node peer.Peer, reader bufio.Reader) error {
	fm.Print(impl.Blue("Peer to follow: "))
	peer, err := reader.ReadString('\n')
	peer = strings.TrimSpace(peer)
	if err != nil {
		return err
	}
	err = node.FollowRequest(peer)
	if err != nil {
		return err
	}
	fm.Printf(impl.Green("Follow request sent to %s\n"), peer)
	return nil
}

func unfollow(node peer.Peer, reader bufio.Reader) error {
	fm.Print(impl.Blue("Peer to unfollow: "))
	peer, err := reader.ReadString('\n')
	peer = strings.TrimSpace(peer)
	if err != nil {
		return err
	}
	node.StopFollowing(peer)
	fm.Printf(impl.Green("You are not following %s anymore.\n"), peer)
	return nil
}

func block(node peer.Peer, reader bufio.Reader) error {
	fm.Print(impl.Blue("Peer address of the peer to block: "))
	peerAddr, err := reader.ReadString('\n')
	peerAddr = strings.TrimSpace(peerAddr)
	if err != nil {
		return err
	}

	fm.Print(impl.Blue("Node ID of the peer to block: "))
	nodeID, err := reader.ReadString('\n')
	nodeID = strings.TrimSpace(nodeID)
	if err != nil {
		return err
	}
	return node.Block(peerAddr, nodeID)
}

func following(node peer.Peer) {
	following := node.GetFollowing()
	impl.PrintList("You are following:", "You are not following anyone yet.", following)
}

func followers(node peer.Peer) {
	followers := node.GetFollowers()
	impl.PrintList("Your followers are:", "You have no followers yet.", followers)
}

func addPeer(node peer.Peer, reader bufio.Reader) error {
	var address string
	fm.Print(impl.Blue("Peer address: "))
	address, err := reader.ReadString('\n')
	address = strings.TrimSpace(address)
	if err != nil {
		return err
	}
	node.AddPeer(address)
	fm.Printf(impl.Green("Peer %s added as neighbor.\n"), address)
	return nil
}
func joinChordNetwork(node peer.Peer, reader bufio.Reader, newNetwork bool) error {
	var address string
	if newNetwork {
		address = ""
	} else {
		var err error
		fm.Print(impl.Blue("Peer address: "))
		address, err = reader.ReadString('\n')
		address = strings.TrimSpace(address)
		if err != nil {
			return err
		}
	}
	chordNode := types.ChordNode{
		IPAddr: address,
	}
	err := node.Join(chordNode)
	if err != nil {
		return err
	}
	if newNetwork {
		fm.Print(impl.Green("Chord network created sucessfully.\n"))
	} else {
		fm.Print(impl.Green("Chord network joined sucessfully.\n"))
	}
	return nil
}

func showFingerTable(node peer.Peer) {
	fm.Print(node.FingerTableToString())
}

func showNeighbors(node peer.Peer) {
	neighbors := node.GetNeighbors()
	impl.PrintList("You neighbors are:", "You have no neighbors yet.", neighbors)
}
