package pipelines

import (
	"context"
	"sync"
)

// GenerateStream takes a function that generates data and returns a channel of type T
func GenerateStream[T any](ctx context.Context, fn func(context.Context) T) <-chan T {
	stream := make(chan T)

	go func() {
		defer close(stream)

		for {
			select {
			case <-ctx.Done():
				return
			case stream <- fn(ctx):
			}
		}
	}()

	return stream
}

// StreamSlice takes a slice of type []T and returns a channel of type T
func StreamSlice[T any](ctx context.Context, data []T) <-chan T {
	stream := make(chan T)

	go func() {
		defer close(stream)

		for i := range data {
			select {
			case <-ctx.Done():
				return
			case stream <- data[i]:
			}
		}
	}()

	return stream
}

// StreamMap takes a map of type map[T]H and returns a channel of type H
func StreamMap[T comparable, H comparable](ctx context.Context, data map[T]H) <-chan H {
	stream := make(chan H)

	go func() {
		defer close(stream)

		for _, val := range data {
			select {
			case <-ctx.Done():
				return
			case stream <- val:
			}
		}
	}()

	return stream
}

// FanOut controls concurrent processing of data from the input channel
func FanOut[T any, H any](ctx context.Context, inputStream <-chan T, fn func(context.Context, T) H, numFan int) []<-chan H {
	process := func() <-chan H {
		stream := make(chan H)

		go func() {
			defer close(stream)

			for value := range inputStream {
				select {
				case <-ctx.Done():
					return
				default:
					// process data with supplied function
					stream <- fn(ctx, value)
				}
			}
		}()

		return stream
	}

	fanOutChannels := make([]<-chan H, numFan)

	for i := 0; i < numFan; i++ {
		fanOutChannels[i] = process()
	}

	return fanOutChannels
}

// FanIn takes any number of readonly channels and returns a fanned in channel
func FanIn[T any](ctx context.Context, channels ...<-chan T) <-chan T {
	var wg sync.WaitGroup
	fannedInStream := make(chan T)

	transfer := func(c <-chan T) {
		defer wg.Done()

		for msg := range c {
			select {
			case <-ctx.Done():
				return
			case fannedInStream <- msg:
			}
		}
	}

	for i := range channels {
		wg.Add(1)
		go transfer(channels[i])
	}

	go func() {
		wg.Wait()
		close(fannedInStream)
	}()

	return fannedInStream
}
