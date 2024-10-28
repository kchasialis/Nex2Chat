package impl

import (
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/types"
	"sync"
)

type RumorTable map[string][]types.Rumor

type rumorTable struct {
	RumorTable
	sync.Mutex
}

type routingTable struct {
	peer.RoutingTable
	sync.Mutex
}

type SyncTable map[string]chan any

type syncTable struct {
	SyncTable
	sync.RWMutex
}

type Requests map[string]struct{}

type requests struct {
	Requests
	sync.RWMutex
}

type SearchReplies map[string]map[string]bool

type searchReplies struct {
	SearchReplies
	sync.Mutex
}

type catalog struct {
	peer.Catalog
	sync.Mutex
}

type PendingRequests map[string]struct{}

type pendingRequests struct {
	PendingRequests
	sync.Mutex
}

// Following maps a peer address to a nodeID that a peer follows.
type Following map[string][KeyLen]byte

type following struct {
	Following
	sync.Mutex
}

// Followers maps a peer address to nodeID for peers that follow a peer.
type Followers map[string][KeyLen]byte

type followers struct {
	Followers
	sync.Mutex
}

// FlaggedPosts maps PostID to a set of peers.
type FlaggedPosts map[string]map[string]struct{}

type flaggedPosts struct {
	FlaggedPosts
	sync.Mutex
}

// NodesReputation maps peer to reputation.
type NodesReputation map[string]int

type nodesReputation struct {
	NodesReputation
	sync.Mutex
}

type BlockedPeers map[string]string

type blockedPeers struct {
	BlockedPeers
	sync.Mutex
}

// Challenges maps peer address to challenge.
type Challenges map[string]*types.PoSChallengeRequestMessage

type challenges struct {
	Challenges
	sync.Mutex
}

// Likes contains a map of posts that he peer liked
type Likes map[string]struct{}

type likes struct {
	Likes
	sync.Mutex
}
