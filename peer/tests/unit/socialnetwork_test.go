package unit

import (
	"encoding/json"
	"math/rand"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/peer/impl"
	"go.dedis.ch/cs438/storage/file"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

func Test_N2C_Social_PostUI(t *testing.T) {
	resetStorage()

	transp := channelFac()

	store, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage has failed")
	}

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	node1.Join(types.ChordNode{})

	postContent1 := []byte("This is my first post")

	// Create post 1 from node 1
	post1ByNode1 := node1.CreatePost(postContent1)
	err = node1.StorePrivatePost(post1ByNode1)
	if err != nil {
		myLogger().Error().Msgf("[Test_N2C_Social_StorePostToChord] [%s] StorePrivatePost has failed", node1.GetAddr())

	}
	myLogger().Info().Msgf("[Test_N2C_Social_StorePostToChord] [%s] post1ByNode1: %v", node1.GetAddr(), post1ByNode1)
}

// Testing the creation of a post and get post functions, as well as equal function
func Test_N2C_Social_CreatePostAndRetrieve(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	store2, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(2) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node2 has failed")
	}

	transp := channelFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store2), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()
	defer node2.Stop()

	node1.Join(types.ChordNode{})
	node2.Join(node1.GetChordNode())

	msg := []byte("helloworld")
	msg2 := []byte("helloworld2")

	// Create post 1 from n1
	post1ByNode1 := node1.CreatePost(msg)
	err = node1.StorePrivatePost(post1ByNode1)
	require.NoError(t, err)

	// Create post 2 from n1
	post2ByNode1 := node1.CreatePost(msg2)
	err = node1.StorePrivatePost(post2ByNode1)
	require.NoError(t, err)

	// Retrieve post 1 from n2
	post1ByNode2, err := node2.GetPost(post1ByNode1.PostID)
	require.NoError(t, err)

	// Post 1 from n1 should be the same as post 1 from n2
	require.True(t, post1ByNode1.Equal(post1ByNode2))
	// Post 2 from n1 should not be the same as post 1 from n2
	require.False(t, post2ByNode1.Equal(post1ByNode2))
}

// Testing the post equal function
func Test_N2C_Social_PostEqualFunctionBasic(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	transp := channelFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	node1.Join(types.ChordNode{})

	msg := []byte("helloworld")

	// Create post 1 from n1
	post1ByNode1 := node1.CreatePost(msg)
	err = node1.StorePrivatePost(post1ByNode1)
	require.NoError(t, err)

	// Retrieve post 1 from n2
	post2ByNode1 := node1.CreatePost(msg)
	err = node1.StorePrivatePost(post2ByNode1)
	require.NoError(t, err)

	// Post 1 from n1 should be equal to post 1 from n2
	require.False(t, post2ByNode1.Equal(post1ByNode1))
}

// Testing the post equal function
func Test_N2C_Social_GetFeedSimple(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	store2, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(2) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node2 has failed")
	}

	transp := channelFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store2), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()
	defer node2.Stop()

	node1.Join(types.ChordNode{})
	node2.Join(node1.GetChordNode())

	msg := []byte("helloworld")

	// Create post 1 from n1
	post1ByNode1 := node1.CreatePost(msg)
	err = node1.StorePrivatePost(post1ByNode1)
	require.NoError(t, err)

	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)
	// Retrieve post 2 from n1

	post2ByNode1 := node1.CreatePost(msg)
	err = node1.StorePrivatePost(post2ByNode1)
	require.NoError(t, err)

	node2.AddFollowing(node1.GetChordNode())
	posts, _ := node2.GetPrivateFeed()

	localPost := make([]types.Post, 2)
	localPost[0] = post1ByNode1
	localPost[1] = post2ByNode1

	sort.Slice(localPost, func(i, j int) bool {
		return localPost[i].PostID < localPost[j].PostID
	})

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].PostID < posts[j].PostID
	})

	myLogger().Info().Msgf("post[0] [%v] v.s. post1ByNode1 [%v]", posts[0], localPost[0])
	myLogger().Info().Msgf("post[1] [%v] v.s. post1ByNode1 [%v]", posts[1], localPost[1])
	require.True(t, posts[0].Equal(localPost[0]))
	require.True(t, posts[1].Equal(localPost[1]))
}

// node gets a not existent post, should return an error
func Test_N2C_Social_Get_No_Post(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	transp := channelFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	node1.Join(types.ChordNode{})
	_, err = node1.GetPost(xid.New().String())

	require.Error(t, err)
}

// node gets a post that was set to the storage
func Test_N2C_Social_Get_Post(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	transp := channelFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	node1.Join(types.ChordNode{})

	post := node1.CreatePost([]byte("ThisIsAPost"))
	postID := post.PostID
	postMarshalled, err := json.Marshal(post)
	node1.GetStorage().GetPostStore().Set(postID, postMarshalled)

	myPost, err := node1.GetPost(postID)
	require.NoError(t, err)
	require.True(t, post.Equal(myPost))
}

// node posts an empty post, should return an error and not post
func Test_N2C_Social_Post_No_Post(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	transp := channelFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	node1.Join(types.ChordNode{})

	var msg []byte
	emptyPost1ByNode1 := node1.CreatePost(msg)
	err = node1.StorePrivatePost(emptyPost1ByNode1)
	require.Error(t, err)
	require.Equal(t, 0, node1.GetStorage().GetPostStore().Len())
}

// node posts a post, the post should be in the storage
func Test_N2C_Social_Post_Correct_Post(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	transp := channelFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	node1.Join(types.ChordNode{})

	content := make([]byte, 16)

	post1ByNode1 := node1.CreatePost(content)
	err = node1.StorePrivatePost(post1ByNode1)
	require.NoError(t, err)
	require.Equal(t, 1, node1.GetStorage().GetPostStore().Len())

	myPostBytes := node1.GetStorage().GetPostStore().Get(post1ByNode1.PostID)
	var myPost types.Post
	err = json.Unmarshal(myPostBytes, &myPost)
	require.NoError(t, err)
	require.True(t, post1ByNode1.Equal(myPost))
}

// node posts a post and gets it, should get the post
func Test_N2C_Social_Post_And_Get(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	transp := channelFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	node1.Join(types.ChordNode{})

	content := make([]byte, 16)
	post1ByNode1 := node1.CreatePost(content)
	err = node1.StorePrivatePost(post1ByNode1)
	require.NoError(t, err)
	require.Equal(t, 1, node1.GetStorage().GetPostStore().Len())

	myPost, err := node1.GetPost(post1ByNode1.PostID)
	require.NoError(t, err)
	require.True(t, post1ByNode1.Equal(myPost))
}

// node gets feed when there are no posts, should not give error, should give empty feed
func Test_N2C_Social_Get_Feed_No_Post(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	transp := channelFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	node1.Join(types.ChordNode{})

	feed, err := node1.GetPrivateFeed()
	require.NoError(t, err)
	require.Len(t, feed, 0)
}

// node gets feed with 1 post: its own, should get the post
func Test_N2C_Social_Get_Feed_Local_Post(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	transp := channelFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	node1.Join(types.ChordNode{})

	msg := []byte("helloworld") //make([]byte, 16)
	_, err = rand.Read(msg)
	require.NoError(t, err)

	post1ByNode1 := node1.CreatePost(msg)
	err = node1.StorePrivatePost(post1ByNode1)
	require.NoError(t, err)
	require.Equal(t, 1, node1.GetStorage().GetPostStore().Len())

	feed, err := node1.GetPrivateFeed()
	require.NoError(t, err)
	require.Len(t, feed, 1)

	postExported := feed[0]
	require.True(t, postExported.Equal(post1ByNode1))

}

// node gets feed with 1 private post: someone else's, should not get the post cause private and not following
func Test_N2C_Social_Get_Feed_Remote_Post_Private(t *testing.T) {
	resetStorage()

	store, err := file.NewPersistency(impl.StoragePath)
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage has failed")
	}

	transp := udpFac()

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()
	defer node2.Stop()

	node1.Join(types.ChordNode{})
	node2.Join(node1.GetChordNode())

	feed, err := node2.GetPrivateFeed()
	require.NoError(t, err)
	require.Len(t, feed, 0)

	msg := make([]byte, 16)
	_, err = rand.Read(msg)
	require.NoError(t, err)

	post1ByNode1 := node1.CreatePost(msg)
	err = node1.StorePrivatePost(post1ByNode1)
	require.NoError(t, err)

	require.Equal(t, 1, node1.GetStorage().GetPostStore().Len())
	require.Equal(t, 1, node2.GetStorage().GetPostStore().Len())

	// remote post so no access
	feed, err = node2.GetPrivateFeed()
	require.NoError(t, err)
	require.Len(t, feed, 0)

	// Own post so has access
	feed, err = node1.GetPrivateFeed()
	require.NoError(t, err)
	require.Len(t, feed, 1)
}

// node gets feed with 1 private post: someone else's, should get the post cause private and following
func Test_N2C_Social_Get_Feed_Remote_Post_Private_Following(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	store2, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(2) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node2 has failed")
	}

	transp := channelFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store2), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()
	defer node2.Stop()

	node1.Join(types.ChordNode{})
	node2.Join(node1.GetChordNode())

	feed, err := node2.GetPrivateFeed()
	require.NoError(t, err)
	require.Len(t, feed, 0)

	content := make([]byte, 16)
	_, err = rand.Read(content)
	require.NoError(t, err)

	post1ByNode1 := node1.CreatePost(content)
	err = node1.StorePrivatePost(post1ByNode1)
	require.NoError(t, err)

	feedN1, err := node1.GetPrivateFeed()
	feedN2, err := node2.GetPrivateFeed()

	require.Len(t, feedN1, 1)
	require.Len(t, feedN2, 0)

	node2.AddFollowing(node1.GetChordNode())

	// remote post so no access
	feedN2, err = node2.GetPrivateFeed()
	require.NoError(t, err)
	require.Len(t, feedN2, 1)
	require.True(t, true, feedN2[0].Equal(post1ByNode1))

	// Own post so has access
	feedN1, err = node1.GetPrivateFeed()
	require.NoError(t, err)
	require.Len(t, feedN1, 1)
	require.True(t, true, feedN1[0].Equal(post1ByNode1))
}

// node gets feed with 1 post public: someone else's, should get the post
func Test_N2C_Social_Get_Feed_Remote_Post_Public(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	store2, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(2) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node2 has failed")
	}

	transp := channelFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store2), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()
	defer node2.Stop()

	node1.Join(types.ChordNode{})
	node2.Join(node1.GetChordNode())

	node1.AddFollowing(node2.GetChordNode())
	node2.AddFollowing(node1.GetChordNode())

	feed, err := node2.GetPublicFeed()
	require.NoError(t, err)
	require.Len(t, feed, 0)

	postMsg := make([]byte, 16)
	_, err = rand.Read(postMsg)
	require.NoError(t, err)

	post := node1.CreatePost(postMsg)
	err = node1.StorePublicPost(post)
	require.NoError(t, err)

	// Remote public post
	feed, err = node2.GetPublicFeed()
	require.NoError(t, err)
	require.Len(t, feed, 1)
	require.True(t, true, feed[0].Equal(post))

	// Own public post
	feed, err = node1.GetPublicFeed()
	require.NoError(t, err)
	require.Len(t, feed, 1)
	require.True(t, true, feed[0].Equal(post))
}

// node sends a follow request but does not get a reply, should not add to following.
func Test_N2C_Social_Follow_Request_No_Response(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	transp := channelFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	node1.Join(types.ChordNode{})

	following, err := transp.CreateSocket("127.0.0.1:0")
	require.NoError(t, err)

	node1.AddPeer(following.GetAddress())

	err = node1.FollowRequest(following.GetAddress())
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	//should have sent a follow request and not got a follow reply
	ins := node1.GetIns()
	require.Len(t, ins, 0)
	outs := node1.GetOuts()
	require.Len(t, outs, 1)
	require.Equal(t, "followrequest", outs[0].Msg.Type)

	//the reply should have been sent to the follower and with itself as source
	require.Equal(t, following.GetAddress(), outs[0].Header.Destination)
	require.Equal(t, node1.GetAddr(), outs[0].Header.RelayedBy)
	require.Equal(t, node1.GetAddr(), outs[0].Header.Source)

	require.Len(t, node1.GetFollowing(), 0)
}

// node gets a follow request, should reply even if they do not know each other (when getting follow request, add peer)
func Test_N2C_Social_Follow_Reply(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	transp := udpFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	node1.Join(types.ChordNode{})

	follower, err := z.NewSenderSocket(transp, "127.0.0.1:0")

	node1.AddPeer(follower.GetAddress())

	id := xid.New().String()
	followMsg := types.FollowRequestMessage{RequestID: id}

	transpMsg, err := node1.GetRegistry().MarshalMessage(&followMsg)
	require.NoError(t, err)
	header := transport.NewHeader(follower.GetAddress(), follower.GetAddress(), node1.GetAddr(), 0)
	packet := transport.Packet{
		Header: &header,
		Msg:    &transpMsg,
	}

	err = follower.Send(node1.GetAddr(), packet, 0)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	//should have got a follow request and sent a follow reply
	ins := node1.GetIns()
	require.Len(t, ins, 1)
	require.Equal(t, "followrequest", ins[0].Msg.Type)
	outs := node1.GetOuts()
	require.Len(t, outs, 1)
	require.Equal(t, "followreply", outs[0].Msg.Type)

	//the reply should have been sent to the follower and with itself as source
	require.Equal(t, follower.GetAddress(), outs[0].Header.Destination)
	require.Equal(t, node1.GetAddr(), outs[0].Header.RelayedBy)
	require.Equal(t, node1.GetAddr(), outs[0].Header.Source)

	//the reply should have the same id as the request
	followReplyMsg := z.GetFollowReply(t, outs[0].Msg)
	followRequestMsg := z.GetFollowRequest(t, ins[0].Msg)
	require.Equal(t, followRequestMsg.RequestID, followRequestMsg.RequestID)
	require.Equal(t, id, followReplyMsg.RequestID)

	require.Len(t, node1.GetFollowers(), 1)
	require.Equal(t, follower.GetAddress(), node1.GetFollowers()[0]) //not sure if it s address or public key
}

// two nodes exchange follow request and reply and become follower/followed
func Test_N2C_Social_Follow_Request_And_Reply(t *testing.T) {
	transp := channelFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0")
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0")
	defer node1.Stop()
	defer node2.Stop()

	node1.AddPeer(node2.GetAddr())
	node2.AddPeer(node1.GetAddr())

	err := node1.FollowRequest(node2.GetAddr())
	require.NoError(t, err)

	time.Sleep(1000 * time.Millisecond)

	//node 1 should have sent one request and got one reply
	outs := node1.GetOuts()
	require.Len(t, outs, 1)
	require.Equal(t, "followrequest", outs[0].Msg.Type)
	ins := node1.GetIns()
	require.Len(t, ins, 1)
	require.Equal(t, "followreply", ins[0].Msg.Type)

	//node 2 should have got one request and sent one reply
	outs = node2.GetOuts()
	require.Len(t, outs, 1)
	require.Equal(t, "followreply", outs[0].Msg.Type)
	ins = node2.GetIns()
	require.Len(t, ins, 1)
	require.Equal(t, "followrequest", ins[0].Msg.Type)

	require.Len(t, node1.GetFollowing(), 1)
	require.Len(t, node2.GetFollowers(), 1)

	require.Len(t, node2.GetFollowing(), 0)
	require.Len(t, node1.GetFollowers(), 0)
}

func Test_N2C_Social_Flag_Non_Existent_Post(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	transp := udpFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	node1.Join(types.ChordNode{})

	err = node1.FlagPost(xid.New().String())
	require.Error(t, err)
}

func Test_N2C_Social_Flag_No_Majority(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	transp := udpFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	node1.Join(types.ChordNode{})

	content := make([]byte, 16)
	post1ByNode1 := node1.CreatePost(content)
	err = node1.StorePrivatePost(post1ByNode1)
	require.NoError(t, err)

	myPost, err := node1.GetPost(post1ByNode1.PostID)
	require.NoError(t, err)
	require.True(t, post1ByNode1.Equal(myPost))

	err = node1.FlagPost(post1ByNode1.PostID)
	require.NoError(t, err)

	//reputation should drop
	time.Sleep(1 * time.Second)
	reputations := node1.GetReputation()
	chordID := node1.GetNodeIDStr()
	require.Equal(t, -1, reputations[chordID])
}

func Test_N2C_Social_Flag_Majority(t *testing.T) {
	resetStorage()

	store1, err := file.NewPersistency(impl.StoragePath + strconv.Itoa(1) + "/")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage for node1 has failed")
	}

	transp := udpFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store1), z.WithAckTimeout(5*time.Second), z.WithChord(true))
	defer node1.Stop()

	node1.Join(types.ChordNode{})

	content := make([]byte, 16)
	post1ByNode1 := node1.CreatePost(content)
	err = node1.StorePrivatePost(post1ByNode1)
	require.NoError(t, err)

	myPost, err := node1.GetPost(post1ByNode1.PostID)
	require.NoError(t, err)
	require.True(t, post1ByNode1.Equal(myPost))

	err = node1.FlagPost(post1ByNode1.PostID)
	require.NoError(t, err)

	//reputation should drop
	time.Sleep(1 * time.Second)
	reputations := node1.GetReputation()
	chordID := node1.GetNodeIDStr()
	require.Equal(t, -1, reputations[chordID])
}

// Peer sends a follow request to another peer and is required to solve a challenge. It does it successfully.
func Test_N2C_Social_ValidateRequest(t *testing.T) {
	transp := udpFac()

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithRequestValidation(true), z.WithChord(true))
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithRequestValidation(true), z.WithChord(true))
	defer node1.Stop()
	defer node2.Stop()

	node2.AddPeer(node1.GetAddr())
	node1.AddPeer(node2.GetAddr())

	err := node2.FollowRequest(node1.GetAddr())
	require.NoError(t, err)

	time.Sleep(15 * time.Second)

	// node2 should have sent a follow request and a challenge reply.
	outs := node2.GetOuts()
	require.Len(t, outs, 2)
	require.Equal(t, outs[0].Msg.Type, "followrequest")
	require.Equal(t, outs[1].Msg.Type, "PoSChallengeReply")

	// node2 should have received a challenge request and a follow reply.
	ins := node2.GetIns()
	require.Len(t, ins, 2)
	require.Equal(t, ins[0].Msg.Type, "PoSChallengeReq")
	require.Equal(t, ins[1].Msg.Type, "followreply")

	// node1 should have sent a challenge request and a follow reply.
	outs = node1.GetOuts()
	require.Len(t, outs, 2)
	require.Equal(t, outs[0].Msg.Type, "PoSChallengeReq")
	require.Equal(t, outs[1].Msg.Type, "followreply")

	// node1 should have received a follow request and a challenge reply.
	ins = node1.GetIns()
	require.Len(t, ins, 2)
	require.Equal(t, ins[0].Msg.Type, "followrequest")
	require.Equal(t, ins[1].Msg.Type, "PoSChallengeReply")

	// Check followers/following lists.
	require.Len(t, node1.GetFollowers(), 1)
	require.Len(t, node2.GetFollowing(), 1)
	require.Len(t, node1.GetFollowing(), 0)
	require.Len(t, node2.GetFollowers(), 0)
}

//func Test_N2C_Social_ValidateRequest_No_Success(t *testing.T) {
//	transp := udpFac()
//
//	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithRequestValidation(true), z.WithChord(true))
//	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithRequestValidation(true), z.WithChord(true))
//	defer node1.Stop()
//	defer node2.Stop()
//
//	node2.AddPeer(node1.GetAddr())
//	node1.AddPeer(node2.GetAddr())
//
//	err := node2.FollowRequest(node1.GetAddr())
//	require.NoError(t, err)
//
//	time.Sleep(2 * time.Second)
//
//	// node2 should have sent a follow request and a challenge reply.
//	outs := node2.GetOuts()
//	require.Len(t, outs, 2)
//	require.Equal(t, outs[0].Msg.Type, "followrequest")
//	require.Equal(t, outs[1].Msg.Type, "PoSChallengeReply")
//
//	// node2 should have received a challenge request and a follow reply.
//	ins := node2.GetIns()
//	require.Len(t, ins, 1)
//	require.Equal(t, ins[0].Msg.Type, "PoSChallengeReq")
//
//	outs = node1.GetOuts()
//	require.Len(t, outs, 1)
//	require.Equal(t, outs[0].Msg.Type, "PoSChallengeReq")
//
//	// node1 should have received a follow request and a challenge reply.
//	ins = node1.GetIns()
//	require.Len(t, ins, 2)
//	require.Equal(t, ins[0].Msg.Type, "followrequest")
//	require.Equal(t, ins[1].Msg.Type, "PoSChallengeReply")
//
//	// Check followers/following lists.
//	require.Len(t, node1.GetFollowers(), 0)
//	require.Len(t, node2.GetFollowing(), 0)
//	require.Len(t, node1.GetFollowing(), 0)
//	require.Len(t, node2.GetFollowers(), 0)
//}

// Two peers join the network, follow each other.
// Node1 posts and Node2 should be able to view the post of node1, non-encrypted.
func Test_N2C_Social_Post_Non_Encrypted(t *testing.T) {
	resetStorage()

	store, err := file.NewPersistency(impl.StoragePath + "/2")
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage has failed")
	}

	transp := channelFac()

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithChord(true), z.WithPostEncryption(true))
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithChord(true), z.WithPostEncryption(true))
	defer node1.Stop()
	defer node2.Stop()

	node2.AddPeer(node1.GetAddr())
	node1.AddPeer(node2.GetAddr())

	// first node to enter the network
	err = node1.Join(types.ChordNode{})
	require.NoError(t, err)

	err = node2.Join(node1.GetChordNode())
	require.NoError(t, err)

	// Node1 follows node2.
	err = node1.FollowRequest(node2.GetAddr())
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	// Check if follow succeeded.
	require.Len(t, node1.GetFollowing(), 1)
	require.Len(t, node2.GetFollowers(), 1)

	// node2 posts.
	post := node2.CreatePost([]byte("asdf"))
	err = node2.StorePrivatePost(post)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	posts, err := node1.GetPrivateFeed()
	require.Len(t, posts, 1)
	require.Equal(t, posts[0].Content, []byte("asdf"))
}

// Two peers join the network, follow each other.
// Node1 posts and Node2 should be able to view the post of node1, non-encrypted.
// Then, node2 blocks node1. Node1 should no longer be able to decrypt the posts.
func Test_N2C_Social_Block(t *testing.T) {
	resetStorage()

	store, err := file.NewPersistency(impl.StoragePath)
	if err != nil {
		myLogger().Error().Msgf("[Test] NewPersistency storage has failed")
	}

	transp := channelFac()

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithChord(true), z.WithPostEncryption(true))
	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithStorage(store), z.WithChord(true), z.WithPostEncryption(true))
	defer node1.Stop()
	defer node2.Stop()

	node2.AddPeer(node1.GetAddr())
	node1.AddPeer(node2.GetAddr())

	// first node to enter the network
	err = node1.Join(types.ChordNode{})
	require.NoError(t, err)

	err = node2.Join(node1.GetChordNode())
	require.NoError(t, err)

	// Node1 follows node2.
	err = node1.FollowRequest(node2.GetAddr())
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	// Check if follow succeeded.
	require.Len(t, node1.GetFollowing(), 1)
	require.Len(t, node2.GetFollowers(), 1)

	// node2 posts.
	post := node2.CreatePost([]byte("post1"))
	err = node2.StorePrivatePost(post)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	posts, err := node1.GetPrivateFeed()
	require.Len(t, posts, 1)
	require.Equal(t, posts[0].Content, []byte("post1"))

	// Node2 blocks node1.
	err = node2.Block(node1.GetAddr(), node1.GetNodeIDStr())
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	// node2 posts.
	post2 := node2.CreatePost([]byte("post2"))
	err = node2.StorePrivatePost(post2)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	// Post retrieved, but should be encrypted.
	posts, err = node1.GetPrivateFeed()
	require.Len(t, posts, 2)
	// we assume keys for older posts / older posts might have been leaked
	// so node1 can get private posts before being blocked
	var post1Decrypted = string(posts[0].Content) == "post1" || string(posts[1].Content) == "post1"
	var post2Encrypted = string(posts[0].Content) != "post2" && string(posts[1].Content) != "post2"
	require.True(t, post1Decrypted)
	require.True(t, post2Encrypted)
}
