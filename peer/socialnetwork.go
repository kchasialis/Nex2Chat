package peer

import (
	"go.dedis.ch/cs438/n2ccrypto"
	"go.dedis.ch/cs438/types"
)

const PublicPostPk = "0"
const KeyLen = n2ccrypto.DefaultKeyLen

type ContentSharing interface {

	// CreatePost from content
	CreatePost(content []byte) types.Post

	// StorePrivatePost stores a private post it
	StorePrivatePost(post types.Post) error

	// StorePublicPost creates a post and stores it
	StorePublicPost(post types.Post) error

	// UpdatePostCatalog function sends a message to all followed peers
	//to get their post catalog and update ours with missing postIDs
	UpdatePostCatalog() error

	// GetPost get a post from storage
	GetPost(postID string) (types.Post, error)

	// DeletePost removes post from storage
	//DeletePost(postID string) error

	// GetPrivateFeed retrieves all posts from Following
	GetPrivateFeed() ([]types.Post, error)

	// GetPublicFeed retrieves all posts from Following
	GetPublicFeed() ([]types.Post, error)

	// SearchByTopic retrieves all public posts related to a regexp
	//SearchByTopic(reg regexp.Regexp) ([]types.Post, error)

	// Like adds a like to a post
	Like(postID string) error

	// Comment adds a comment to a post
	Comment(postID, content string) error

	// FlagPost requests to flag a post
	// if a majority of node approves, the node’s reputation is decreased
	FlagPost(postID string) error
}

type UserInteracting interface {

	// FollowRequest requests to follow a peer
	FollowRequest(peer string) error

	// StopFollowing stop following a peer
	StopFollowing(peer string)

	// Block prevent a peer from following you
	Block(peer string, nodeID string) error

	// GetFollowing returns the nodes that the peer is following
	GetFollowing() []string

	// GetFollowers returns the nodes that follow the peer
	GetFollowers() []string

	// GetNodeID returns the ID of the peer
	GetNodeID() [KeyLen]byte

	// GetNodeIDStr returns the ID of the peer encoded as a string
	GetNodeIDStr() string

	// GetReputation returns the reputation of the nodes
	GetReputation() map[string]int
}

type ChordInterface interface {
	// Join makes a node join a Chord Network
	Join(existingNode types.ChordNode) error

	// GetChordNode returns your Chord Node
	GetChordNode() types.ChordNode

	// InitFingerTable initialize finger table using an arbitrary Chord node
	InitFingerTable(existingNode types.ChordNode) error

	// FingerTableToString outputs a formatted string containing the finger table of the node
	FingerTableToString() string

	// MySuccessor return the current successor according to the fingertable
	MySuccessor() types.ChordNode

	// MyPredecessor return the current predecessor stored in the node
	MyPredecessor() types.ChordNode

	// DisplayRing visualizes big integers on a circular ring within the specified range and radius
	DisplayRing(chordIDs [][KeyLen]byte, labels ...string) error

	// PredAndSuccToString outputs a formatted string containing the predecessor and successor of current node
	PredAndSuccToString() string

	// PostCatalogToString outputs a formatted string containing the of the PostCatalog of the node
	PostCatalogToString() string
}

/*type FingerTableInterface interface {
	// TODO ADD COMMENT
	// FingerTableGet
	FingerTableGet(index int) (FingerEntry, error)

	// FingerTableSetStart
	FingerTableSetStart(index int, chordNode types.ChordNode) error

	// FingerTableSetNode
	FingerTableSetNode(index int, chordNode types.ChordNode) error

	// FingerTableAssignMinusOne
	FingerTableAssignMinusOne(index int) error
}

*/

type TempoInterface interface {
	AddFollowing(addr types.ChordNode)
}

type FingerEntry struct {
	Start types.ChordNode // identifier of the Start of the interval
	Node  types.ChordNode // struct containing the ID and IP address of the successor Node
}
