package malicious

import (
	"fmt"
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/peer/impl"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	// logout is the logger configuration
	logout = zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		FormatCaller: func(i interface{}) string {
			return filepath.Base(fmt.Sprintf("%s", i))
		},
	}
)

func myLogger() *zerolog.Logger {
	mylogger := zerolog.New(logout).
		Level(zerolog.TraceLevel).
		With().Timestamp().Logger().
		With().Caller().Logger().
		With().Str("role", "test").Logger()
	return &mylogger
}

// spammer spams new posts public/private in every delayPost time, since the moment it starts
type SpammerNode struct {
	peer.Peer
	newPostsPublic  int
	newPostsPrivate int
	delayPost       time.Duration
	wg              sync.WaitGroup
	stop            chan struct{}
	sync.RWMutex
}

func GetSpammerFac(newPostsPublic int, newPostsPrivate int, delayPost time.Duration) peer.Factory {
	if delayPost == 0 {
		delayPost = time.Second
	}
	return func(conf peer.Configuration) peer.Peer {
		node := impl.NewPeer(conf)
		return &SpammerNode{
			Peer:            node,
			newPostsPublic:  newPostsPublic,
			newPostsPrivate: newPostsPrivate,
			delayPost:       delayPost,
			stop:            make(chan struct{}),
		}
	}
}

func (n *SpammerNode) Start() error {
	n.Lock()
	n.wg.Add(n.newPostsPrivate + n.newPostsPublic)
	n.Unlock()

	go func() {
		n.createPublicPosts()
		n.createPrivatePosts()
	}()

	return nil
}

func (n *SpammerNode) createPrivatePosts() {
	for i := 0; i < n.newPostsPrivate; i++ {
		go func() {
			defer n.wg.Done()
			select {
			case <-n.stop:
				return
			default:
				post := n.CreatePost(make([]byte, 16))
				err := n.StorePrivatePost(post)
				if err != nil {
					myLogger().Info().Msgf("[Spammer] PostPrivate has failed")
				}
			}
		}()
		time.Sleep(n.delayPost)
	}
}

func (n *SpammerNode) createPublicPosts() {
	for i := 0; i < n.newPostsPublic; i++ {
		go func() {
			defer n.wg.Done()
			select {
			case <-n.stop:
				return
			default:
				post := n.CreatePost(make([]byte, 16))
				err := n.StorePublicPost(post)
				if err != nil {
					myLogger().Info().Msgf("[Spammer] PostPublic has failed")
				}
			}
		}()
		time.Sleep(n.delayPost)
	}
}

func (n *SpammerNode) Stop() error {
	//to avoid stop while wg.Add()
	n.RLock()
	defer n.RUnlock()

	close(n.stop)
	n.wg.Wait()
	return nil
}
