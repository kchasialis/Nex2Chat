package types

import (
	"time"

	"go.dedis.ch/cs438/n2ccrypto"
)

const KeyLen = n2ccrypto.DefaultKeyLen

type Post struct {
	PostID    string    // Unique identifier for the post.
	AuthorID  string    // Identifier of the node who created the post.
	Content   []byte    // The main content of the post (text, images, etc.).
	CreatedAt time.Time // Timestamp of when the post was created.
	UpdatedAt time.Time // Timestamp of when the post was last updated.
	Likes     int       // Number of likes the post has received.
	Comments  []Comment // Slice of Comments on the post.
	KeyID     int       // KeyID for the encryption of the Content (0 = No encryption, 1...N = Last N keys)
}

type Comment struct {
	CommentID string    // Unique identifier for the comment.
	PostID    string    // Identifier of the post to which this comment is attached.
	AuthorID  string    // Identifier of the node who created the comment.
	Content   string    // The main content of the comment (text, images, etc.)
	CreatedAt time.Time // Timestamp of when the comment was created.
	KeyID     int       // KeyID for the encryption of the Content (0 = No encryption, 1...N = Last N keys)
}

// FollowRequestMessage is sent to request to follow.
// - implements types.Message
type FollowRequestMessage struct {
	RequestID string
	// ChordID of the peer that sends the request.
	ChordID [KeyLen]byte
}

// FollowReplyMessage is sent as a reply to a follow request.
// - implements types.Message
type FollowReplyMessage struct {
	RequestID string
	// ChordID of the peer that initiated the request.
	ChordID [KeyLen]byte
}

// BlockMessage is sent when a block is initiated.
// - implements types.Message
type BlockMessage struct {
	// NewChordID that will replace the old one.
	NewChordID [KeyLen]byte
}

// FlagPostMessage is sent to flag a post
// - implements types.Message
type FlagPostMessage struct {
	PostID    string
	FlaggerID string
}

type ChordQuery uint

const (
	RPCSuccessor              ChordQuery = 0
	RPCClosestPrecedingFinger ChordQuery = 1
	RPCFindSuccessor          ChordQuery = 2
	RPCGetPredecessor         ChordQuery = 3
	RPCSetPredecessor         ChordQuery = 4
	RPCSetSuccessor           ChordQuery = 5
	RPCUpdateFingerTable      ChordQuery = 6
	PostRequest               ChordQuery = 7
)

var ChordQueryStr = map[ChordQuery]string{
	RPCSuccessor:              "RPCSuccessor",
	RPCClosestPrecedingFinger: "RPCClosestPrecedingFinger",
	RPCFindSuccessor:          "RPCFindSuccessor",
	RPCGetPredecessor:         "RPCGetPredecessor",
	RPCSetPredecessor:         "RPCSetPredecessor",
	RPCSetSuccessor:           "RPCSetSuccessor",
	RPCUpdateFingerTable:      "RPCUpdateFingerTable",
	PostRequest:               "PostRequest",
}

type ChordNode struct {
	ID     [KeyLen]byte
	IPAddr string
}

// ChordRequestMessage is a message sent to request from another nodes.
// - implements types.Message
type ChordRequestMessage struct {
	RequestID string
	ChordNode ChordNode
	Query     ChordQuery
	IthFinger uint

	// PostMessage
	Post      Post
	IsPrivate bool
}

// ChordReplyMessage is a message sent to reply to Chord Request from another nodes.
// - implements types.Message
type ChordReplyMessage struct {
	RequestID string
	ChordNode ChordNode
	Query     ChordQuery

	// PostMessageAck
	PostID string
}

// PostRequestMessage is a message sent to request a Post.
// - implements types.Message
type PostRequestMessage struct {
	RequestID string
	PostID    string
}

// PostReplyMessage is a message containing a Post returned after receiving a PostRequestMessage.
// - implements types.Message
type PostReplyMessage struct {
	RequestID string
	Post      Post
}

type CatalogRequestMessage struct {
	RequestID string
}

type CatalogReplyMessage struct {
	RequestID string
	Catalog   map[string][]byte
}
