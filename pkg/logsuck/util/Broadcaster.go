package util

type Broadcaster[T any] struct {
	channels []chan T
}

func (b *Broadcaster[T]) Broadcast(v T) {
	go func() {
		for _, c := range b.channels {
			c <- v
		}
	}()
}

func (b *Broadcaster[T]) Subscribe() <-chan T {
	ret := make(chan T)
	b.channels = append(b.channels, ret)
	return ret
}
