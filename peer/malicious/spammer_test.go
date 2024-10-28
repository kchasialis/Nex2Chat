package malicious

import (
	"github.com/stretchr/testify/require"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/transport/udp"
	"go.dedis.ch/cs438/types"
	"testing"
	"time"
)

var udpFac transport.Factory = udp.NewUDP

func Test_N2C_Malicious_Spammer_Start(t *testing.T) {
	opts := []z.Option{
		z.WithChord(true),
		z.WithAutostart(false),
	}

	transp := udpFac()
	spamFac := GetSpammerFac(0, 0, 100*time.Millisecond)

	node1 := z.NewTestNode(t, spamFac, transp, "127.0.0.1:0", opts...)
	err := node1.Join(types.ChordNode{})
	require.NoError(t, err)

	err = node1.Start()
	if err != nil {
		myLogger().Error().Msgf("[Test] Start of Spammer failed")
	}
	require.NoError(t, err)
	node1.Stop()

	publicFeed, err := node1.GetPublicFeed()
	require.Len(t, publicFeed, 0)
}

func Test_N2C_Malicious_Spammer_Spam_Public(t *testing.T) {
	opts := []z.Option{
		z.WithChord(true),
		z.WithAutostart(false),
	}

	transp := udpFac()
	spamFac := GetSpammerFac(10, 0, 100*time.Millisecond)

	node1 := z.NewTestNode(t, spamFac, transp, "127.0.0.1:0", opts...)
	err := node1.Join(types.ChordNode{})
	require.NoError(t, err)

	err = node1.Start()
	if err != nil {
		myLogger().Error().Msgf("[Test] Start of Spammer failed")
	}
	time.Sleep(10 * time.Second)
	require.NoError(t, err)
	node1.Stop()

	publicFeed, err := node1.GetPublicFeed()
	require.Len(t, publicFeed, 10)
}

func Test_N2C_Malicious_Spammer_Spam_Private(t *testing.T) {
	opts := []z.Option{
		z.WithChord(true),
		z.WithAutostart(false),
	}

	transp := udpFac()
	spamFac := GetSpammerFac(0, 10, 100*time.Millisecond)

	node1 := z.NewTestNode(t, spamFac, transp, "127.0.0.1:0", opts...)
	err := node1.Join(types.ChordNode{})
	require.NoError(t, err)

	err = node1.Start()
	if err != nil {
		myLogger().Error().Msgf("[Test] Start of Spammer failed")
	}
	time.Sleep(10 * time.Second)
	require.NoError(t, err)
	node1.Stop()

	privateFeed, err := node1.GetPrivateFeed()
	require.Len(t, privateFeed, 10)
}

func Test_N2C_Malicious_Spammer_Spam_Both(t *testing.T) {
	opts := []z.Option{
		z.WithChord(true),
		z.WithAutostart(false),
	}

	transp := udpFac()
	spamFac := GetSpammerFac(10, 10, 100*time.Millisecond)

	node1 := z.NewTestNode(t, spamFac, transp, "127.0.0.1:0", opts...)
	err := node1.Join(types.ChordNode{})
	require.NoError(t, err)

	err = node1.Start()
	if err != nil {
		myLogger().Error().Msgf("[Test] Start of Spammer failed")
	}
	time.Sleep(10 * time.Second)
	require.NoError(t, err)
	node1.Stop()

	publicFeed, err := node1.GetPublicFeed()
	privateFeed, err := node1.GetPrivateFeed()
	require.Equal(t, 20, len(publicFeed)+len(privateFeed))
}

func Test_N2C_Malicious_Spammer_Spam_Stop(t *testing.T) {
	opts := []z.Option{
		z.WithChord(true),
		z.WithAutostart(false),
	}

	transp := udpFac()
	spamFac := GetSpammerFac(50, 50, 100*time.Millisecond)

	node1 := z.NewTestNode(t, spamFac, transp, "127.0.0.1:0", opts...)
	err := node1.Join(types.ChordNode{})
	require.NoError(t, err)

	err = node1.Start()
	if err != nil {
		myLogger().Error().Msgf("[Test] Start of Spammer failed")
	}
	time.Sleep(10 * time.Second)
	require.NoError(t, err)
	node1.Stop()

	//after stopping, node should not post anything else
	publicFeed1, err := node1.GetPublicFeed()
	postsPosted := len(publicFeed1)
	time.Sleep(1 * time.Second)
	publicFeed2, err := node1.GetPublicFeed()
	require.Equal(t, postsPosted, len(publicFeed2))
}
