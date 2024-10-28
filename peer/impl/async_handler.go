package impl

import (
	"github.com/rs/zerolog/log"
	"sync"
)

type AsyncHandler struct {
	cha map[string]chan interface{}
	sync.Mutex
}

func NewAsyncHandler() AsyncHandler {
	return AsyncHandler{make(map[string]chan interface{}), sync.Mutex{}}
}

func (l *AsyncHandler) SendToChan(key string, v interface{}) bool {
	l.Lock()
	defer l.Unlock()
	channel, ok := l.cha[key]
	if !ok {
		log.Error().Msgf("[SendToChan] Channel %s not found", key)
		return false
	}

	select {
	case channel <- v:
		return true
	default:
		return false
	}
}

func (l *AsyncHandler) ChanRem(key string) {
	l.Lock()
	defer l.Unlock()

	delete(l.cha, key)
}

func (l *AsyncHandler) ChanAdd(key string, ch chan interface{}) {
	l.Lock()
	defer l.Unlock()

	l.cha[key] = ch
}
