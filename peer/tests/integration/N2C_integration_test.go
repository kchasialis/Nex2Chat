package integration

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
	"math"
	"math/rand"
	"os"
	"testing"
	"time"

	z "go.dedis.ch/cs438/internal/testing"
)

// state holds all variables needed across stages.
type stateN2C struct {
	t              *testing.T
	nodes          []*z.TestNode
	chordEntryNode types.ChordNode
	friends        int
	publicPosts    []types.Post
	privatePosts   []types.Post

	opts []z.Option
}

// stage represents a new step in the integration test
type stageN2C func(*stateN2C) *stateN2C

// Test_N2C_Perf_Follow_10s is used to calculate the percentage of runs that successfully
// pass the following stage with a wait time of 10sec between the sending of the request
// and the checking of the number of followers
func Test_N2C_Perf_Follow_10s(t *testing.T) {
	scenario := func(transport transport.Transport, X uint, Y uint) func(*testing.T) {
		return func(t *testing.T) {
			networkSetup := setupAccounts(transport, X)
			becomingFollowers := followMesh(Y)

			stages := []stageN2C{
				networkSetup,
				becomingFollowers,
			}

			t.Run("stages 1-7", func(t *testing.T) {
				s := &stateN2C{t: t}
				defer stopState(s)

				for k := 0; k < len(stages); k++ {
					stages[k](s)
				}
			})
		}
	}

	// Node 1 ------ Node 2
	//   |             |
	// Node 3 ---------

	attempts := 20
	for i := 0; i < attempts; i++ {
		t.Run("undisrupted transport - 3 nodes - 3 friends", scenario(udpFac(), 3, 3))
	}
}

// X nodes create an account, Y nodes are friends between them,
// all nodes post something and all nodes get their feed.
// Check that all X nodes have the public posts and additionally
// the Y nodes have each other's private posts. User1 blocks user2.
// user 2 and user1 should no longer have each other's posts (except user1's public)
// A new node joins the network, it should have access to all public
// posts. The new node follows user1, it should have access to
// one private post. Then K nodes stop. Check if the other nodes
// can access the public posts
func Test_N2C_Integration_No_Malicious(t *testing.T) {
	scenario := func(transport transport.Transport, X uint, Y uint, K uint) func(*testing.T) {
		return func(t *testing.T) {
			networkSetup := setupAccounts(transport, X)
			becomingFollowers := followMesh(Y)
			crashingNodes := stopNodes(K)

			stages := []stageN2C{
				networkSetup,
				becomingFollowers,
				postGetPublic,
				postGetPrivate,
				blockNode,
				newNode,
				crashingNodes,
			}

			t.Run("stages 1-7", func(t *testing.T) {
				s := &stateN2C{t: t}
				defer stopState(s)

				for k := 0; k < len(stages); k++ {
					stages[k](s)
				}
			})
		}
	}

	// Node 1 ------ Node 2
	//   |             |
	// Node 3 ---------
	//
	// Node 4   Node 5   Node 6

	t.Run("undisrupted transport - 6 nodes - 3 friends - 3 crash", scenario(udpFac(), 6, 3, 3))
}

// X nodes create an account, and they are all friends between them.
// A node joins the network and starts spamming. While this node is
// spamming (posting 1 post every 100 milliseconds), the other nodes
// are checking every 1 second for new spam and flagging it. As the
// spamming node's reputation drops, its ability to post fast decreases
func Test_N2C_Integration_Malicious_Spammer(t *testing.T) {
	scenario := func(transport transport.Transport, X uint) func(*testing.T) {
		return func(t *testing.T) {
			networkSetup := setupAccounts(transport, X)
			becomingFollowers := followMesh(X)

			stages := []stageN2C{
				networkSetup,
				becomingFollowers,
				addSpammer,
				flaggingSpam,
			}

			t.Run("stages 1-4", func(t *testing.T) {
				s := &stateN2C{t: t}
				defer stopState(s)

				for k := 0; k < len(stages); k++ {
					stages[k](s)
				}
			})
		}
	}

	// Spammer
	//   |
	// Node 1-----Nodes 2-15

	t.Run("undisrupted transport - 5 nodes and 1 spammer", scenario(udpFac(), 5))
}

func setupAccounts(transport transport.Transport, X uint) stageN2C {
	return func(s *stateN2C) *stateN2C {
		s.t.Log("~~ stage 1 <> setup ~~")

		require.GreaterOrEqual(s.t, int(X), 2)
		opts := []z.Option{
			z.WithRequestValidation(true),
			z.WithChord(true),
			z.WithTotalPeers(X),
			z.WithContinueMongering(0.5),
			z.WithHeartbeat(time.Second * 30),
			z.WithAntiEntropy(time.Second * 5),
			z.WithPostEncryption(true),
		}

		var sampleNode types.ChordNode
		//X Non-Malicious Nodes
		for i := uint(0); i < X; i++ {
			node := z.NewTestNode(s.t, studentFac, transport, "127.0.0.1:0", opts...)
			if i == 0 {
				err := node.Join(types.ChordNode{})
				require.NoError(s.t, err)
				sampleNode = node.GetChordNode()
				s.chordEntryNode = sampleNode
			} else {
				err := node.Join(sampleNode)
				require.NoError(s.t, err)
			}
			s.nodes = append(s.nodes, &node)
		}
		// everyone is everyone's peer
		for _, node := range s.nodes {
			for _, other := range s.nodes {
				node.AddPeer(other.GetAddr())
			}
		}

		s.opts = opts

		return s
	}
}

func followMesh(Y uint) stageN2C {
	return func(s *stateN2C) *stateN2C {
		s.t.Log("~~ stage 2 <> becoming followers ~~")

		s.friends = int(Y)

		for i, user := range s.nodes {
			// the nodes that are not in the follow mesh should only follow themselves
			if i >= s.friends {
				err := user.FollowRequest(user.GetAddr())
				require.NoError(s.t, err)
				continue
			}
			//the nodes in the follow mesh should follow each other
			for i, follow := range s.nodes {
				if i >= s.friends {
					break
				}
				err := user.FollowRequest(follow.GetAddr())
				require.NoError(s.t, err)
			}
		}

		//the time it takes for everyone to accept the follow is unpredictable but 10s seems good most times
		time.Sleep(time.Second * 10)

		// check if first Y nodes are following each other and other nodes are not
		for i, user := range s.nodes {
			if i >= s.friends {
				followers := user.GetFollowers()
				require.Len(s.t, followers, 1)
				require.Equal(s.t, followers[0], user.GetAddr())
				following := user.GetFollowing()
				require.Len(s.t, following, 1)
				require.Equal(s.t, following[0], user.GetAddr())
				continue
			}
			followers := user.GetFollowers()
			require.Len(s.t, followers, s.friends)
			following := user.GetFollowing()
			require.Len(s.t, following, s.friends)

			for i, follow := range s.nodes {
				if i >= s.friends {
					break
				}
				require.Contains(s.t, following, follow.GetAddr())
			}
		}

		return s
	}
}

func addSpammer(s *stateN2C) *stateN2C {
	s.t.Log("~~ stage 3 <> adding spammer node ~~")

	spammer := z.NewTestNode(s.t, studentFac, udpFac(), "127.0.0.1:0", s.opts...)
	err := spammer.Join(s.chordEntryNode)
	require.NoError(s.t, err)
	s.nodes = append(s.nodes, &spammer)
	for _, other := range s.nodes {
		spammer.AddPeer(other.GetAddr())
	}
	//after the node follows someone, they will be updated on the catalog
	err = spammer.FollowRequest(s.nodes[0].GetAddr())
	require.NoError(s.t, err)

	go func(spammer z.TestNode, s *stateN2C) {
		for i := 0; i < 1000; i++ {
			content := []byte("spam")
			post := spammer.CreatePost(content)
			err := spammer.StorePublicPost(post)
			if err != nil {
				s.t.Log("stopping spam")
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}(spammer, s)

	return s
}

func flaggingSpam(s *stateN2C) *stateN2C {
	s.t.Log("~~ stage 4 <> flagging spam ~~")

	//for 10 seconds, 1 second at a time, they check for spam
	for t := 0; t < 100; t++ {
		time.Sleep(time.Second * 1)
		s.t.Log("looking for spam")
		for _, user := range s.nodes[:len(s.nodes)-1] {
			publicFeed, err := user.GetPublicFeed()
			require.NoError(s.t, err)
			for _, publicPost := range publicFeed {
				if string(publicPost.Content) == string([]byte("spam")) {
					err := user.FlagPost(publicPost.PostID)
					require.NoError(s.t, err)
				}
			}
		}
		spammerID := s.nodes[len(s.nodes)-1].GetNodeIDStr()
		s.t.Log(s.nodes[0].GetReputation()[spammerID])
	}

	return s
}

func postGetPublic(s *stateN2C) *stateN2C {
	s.t.Log("~~ stage 3 <> post and get public feed ~~")

	// every user posts 1 public post
	for _, user := range s.nodes {
		content := []byte("hello world, i am user " + user.GetAddr())
		post := user.CreatePost(content)
		err := user.StorePublicPost(post)
		require.NoError(s.t, err)
		s.publicPosts = append(s.publicPosts, post)
		time.Sleep(time.Second * 1)
	}

	time.Sleep(time.Second * 10)

	for i, user := range s.nodes {
		publicFeed, err := user.GetPublicFeed()
		require.NoError(s.t, err)
		contain := 0
		for _, postPublic := range s.publicPosts {
			for _, postFeed := range publicFeed {
				if postPublic.Equal(postFeed) {
					contain++
					break
				}
			}
		}
		// every post in the feed should be equal to some public post
		require.Equal(s.t, len(publicFeed), contain)
		// what percentage of the total posts posted can the node get in their feed
		percentage := float64(len(publicFeed)) / float64(len(s.publicPosts)) * 100
		s.t.Log("user", i, "can reach", math.Floor(percentage), "% of the public posts")
	}

	return s
}

func postGetPrivate(s *stateN2C) *stateN2C {
	s.t.Log("~~ stage 4 <> post and get private feed ~~")

	// every user posts 1 private post
	for _, user := range s.nodes {
		content := []byte("hello friends, i am user " + user.GetAddr())
		post := user.CreatePost(content)
		err := user.StorePrivatePost(post)
		require.NoError(s.t, err)
		s.privatePosts = append(s.privatePosts, post)
		time.Sleep(time.Second * 1)
	}

	time.Sleep(time.Second * 10)

	for i, user := range s.nodes {
		privateFeed, err := user.GetPrivateFeed()
		require.NoError(s.t, err)

		if i >= s.friends {
			// if the node is not in the mesh of friends, they should only have their own private post
			require.Len(s.t, privateFeed, 1)
			//require.True(s.t, privateFeed[0].Equal(s.privatePosts[i]))
			continue
		}

		// if the node is in the mesh of friends, it should have its friends' private posts and no other private post
		percentage := float64(len(privateFeed)) / float64(s.friends) * 100
		s.t.Log("user", i, "can reach", math.Floor(percentage), "% of the private posts")
	}

	return s
}

func blockNode(s *stateN2C) *stateN2C {
	s.t.Log("~~ stage 5 <> user 0 blocking user 1 ~~")

	user0 := s.nodes[0]
	user1 := s.nodes[1]
	err := user0.Block(user1.GetAddr(), user1.GetNodeIDStr())
	require.NoError(s.t, err)

	time.Sleep(time.Second * 10)

	followers := user0.GetFollowers()
	require.Len(s.t, followers, s.friends-1)
	following := user0.GetFollowing()
	require.Len(s.t, following, s.friends-1)

	//user 1 is not notified that it was blocked
	followers = user1.GetFollowers()
	require.Len(s.t, followers, s.friends)
	following = user1.GetFollowing()
	require.Len(s.t, following, s.friends)

	//user 0 should no longer be getting user 1's public posts
	publicFeed, err := user0.GetPublicFeed()
	require.NoError(s.t, err)
	contain := 0
	for _, postFeed := range publicFeed {
		if postFeed.Equal(s.publicPosts[1]) {
			contain++
			break
		}
	}
	require.Equal(s.t, 0, contain)

	//user 0 should no longer be getting user 1's private posts
	privateFeed, err := user0.GetPrivateFeed()
	require.NoError(s.t, err)
	contain = 0
	for _, postFeed := range privateFeed {
		if postFeed.Equal(s.privatePosts[1]) {
			contain++
			break
		}
	}
	require.Equal(s.t, 0, contain)

	//user 1 should no longer be getting user 0's new private posts
	content := []byte("private")
	newPost := user0.CreatePost(content)
	err = user0.StorePrivatePost(newPost)
	require.NoError(s.t, err)
	time.Sleep(time.Second * 5)
	post, _ := user1.GetPost(newPost.PostID)
	require.False(s.t, post.Equal(newPost))

	return s
}

func newNode(s *stateN2C) *stateN2C {
	s.t.Log("~~ stage 6 <> add one user ~~")

	newUser := z.NewTestNode(s.t, studentFac, udpFac(), "127.0.0.1:0", s.opts...)
	err := newUser.Join(s.chordEntryNode)
	require.NoError(s.t, err)
	s.nodes = append(s.nodes, &newUser)
	for _, other := range s.nodes {
		newUser.AddPeer(other.GetAddr())
	}
	//after the node follows someone, they will be updated on the catalog
	err = newUser.FollowRequest(s.nodes[0].GetAddr())
	require.NoError(s.t, err)

	time.Sleep(time.Second * 10)

	publicFeed, err := newUser.GetPublicFeed()
	require.NoError(s.t, err)

	percentage := float64(len(publicFeed)) / float64(len(s.publicPosts)) * 100
	s.t.Log("new user can reach", math.Floor(percentage), "% of the public posts")

	return s
}

func contains(slice []int, value int) bool {
	for _, number := range slice {
		if number == value {
			return true
		}
	}
	return false
}

func stopNodes(K uint) stageN2C {
	return func(s *stateN2C) *stateN2C {
		s.t.Log("~~ stage 7 <> crashing nodes ~~")

		require.GreaterOrEqual(s.t, len(s.nodes), int(K))

		var allNodes []*z.TestNode
		allNodes = append(allNodes, s.nodes...)
		var crashed []int

		s.t.Log("reachability before crashing\n")
		var reachability [][]error
		for _, user := range allNodes {
			var userReachability []error
			for j := range allNodes[:len(allNodes)-1] {
				_, err := user.GetPost(s.publicPosts[j].PostID)
				userReachability = append(userReachability, err)
			}
			reachability = append(reachability, userReachability)
		}
		z.DisplayReachability(s.t, os.Stdout, reachability, crashed)

		for i := 0; i < int(K); i++ {
			//crash a random node
			r := rand.Intn(len(s.nodes))
			err := s.nodes[r].Stop()
			for k, user := range allNodes {
				if user.GetAddr() == s.nodes[r].GetAddr() {
					crashed = append(crashed, k)
				}
			}
			require.NoError(s.t, err)
			s.nodes = append(s.nodes[:r], s.nodes[r+1:]...)

			//only print reachability for <= 10 total nodes
			if len(allNodes) > 10 {
				continue
			}
			s.t.Log("reachability after", i+1, "crashes\n")
			var reachability [][]error
			for userNumber, user := range allNodes {
				var userReachability []error
				if contains(crashed, userNumber) {
					reachability = append(reachability, userReachability)
					continue
				}
				for j := range allNodes[:len(allNodes)-1] {
					// to not block, after 1*Seconds of trying to get a post, move on
					ch := make(chan error)
					go func(s *stateN2C, user *z.TestNode, j int) {
						_, err := user.GetPost(s.publicPosts[j].PostID)
						ch <- err
					}(s, user, j)
					select {
					case <-time.After(time.Second * 1):
						userReachability = append(userReachability, xerrors.Errorf("timeout"))
					case err := <-ch:
						userReachability = append(userReachability, err)
					}
				}
				reachability = append(reachability, userReachability)
			}
			z.DisplayReachability(s.t, os.Stdout, reachability, crashed)
		}

		s.nodes = allNodes
		return s
	}
}

// stop stops all nodes.
func stopState(s *stateN2C) *stateN2C {
	s.t.Log("~~ stop ~~")

	for _, node := range s.nodes {
		n, ok := node.Peer.(z.Terminable)
		if ok {
			err := n.Terminate()
			require.NoError(s.t, err)
		} else {
			err := node.Stop()
			require.NoError(s.t, err)
		}
	}

	return s
}
