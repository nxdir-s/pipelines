package pipelines

import (
	"context"
	"os"
	"os/signal"
	"testing"
)

const (
	TestInt int = 1
)

func TestGenerateStreamInt(t *testing.T) {
	cases := []struct {
		in   func(context.Context) int
		want int
	}{
		{
			in:   func(ctx context.Context) int { return TestInt },
			want: TestInt,
		},
	}

	for _, tc := range cases {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

		stream := GenerateStream(ctx, tc.in)

		select {
		case <-ctx.Done():
			continue
		case out := <-stream:
			if out != tc.want {
				t.Errorf("GenerateStream: %d, want %d", out, tc.want)
			}
		}

		cancel()
	}
}

func TestStreamSliceInt(t *testing.T) {
	cases := []struct {
		in   []int
		want int
	}{
		{
			in:   []int{TestInt, TestInt, TestInt, TestInt, TestInt},
			want: TestInt,
		},
		{
			in:   []int{TestInt},
			want: TestInt,
		},
		{
			in:   []int{},
			want: TestInt,
		},
	}

	for _, tc := range cases {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

		stream := StreamSlice(ctx, tc.in)

		count := 0
		for out := range stream {
			count++

			if out != tc.want {
				t.Errorf("StreamSlice: %d, want %d", out, tc.want)
			}
		}

		if count != len(tc.in) {
			t.Errorf("StreamSlice: missing data, len() %d, found %d", len(tc.in), count)
		}

		cancel()
	}
}

func TestStreamMapInt(t *testing.T) {
	cases := []struct {
		in   map[int]int
		want int
	}{
		{
			in:   map[int]int{1: TestInt, 2: TestInt, 3: TestInt},
			want: TestInt,
		},
		{
			in:   map[int]int{1: TestInt},
			want: TestInt,
		},
		{
			in:   map[int]int{},
			want: TestInt,
		},
	}

	for _, tc := range cases {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

		stream := StreamMap(ctx, tc.in)

		count := 0
		for out := range stream {
			count++

			if out != tc.want {
				t.Errorf("StreamMap: %d, want %d", out, tc.want)
			}
		}

		if count != len(tc.in) {
			t.Errorf("StreamMap: missing data, len() %d, found %d", len(tc.in), count)
		}

		cancel()
	}
}

func TestFanOutInt(t *testing.T) {
	cases := []struct {
		in     chan int
		fn     func(context.Context, int) int
		numIn  int
		numFan int
		want   int
	}{
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			numIn:  1,
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			numIn:  1,
			numFan: 1,
			want:   TestInt,
		},
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			numIn:  10,
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			numIn:  10,
			numFan: 1,
			want:   TestInt,
		},
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			numIn:  0,
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			numIn:  0,
			numFan: 1,
			want:   TestInt,
		},
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			numIn:  100,
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			numIn:  100,
			numFan: 1,
			want:   TestInt,
		},
	}

	for _, tc := range cases {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

		go func() {
			defer close(tc.in)

			for range tc.numIn {
				select {
				case <-ctx.Done():
					return
				default:
					tc.in <- TestInt
				}
			}
		}()

		channels := FanOut(ctx, tc.in, tc.fn, tc.numFan)

		if len(channels) != tc.numFan {
			t.Errorf("FanOut: number of channels %d, numFan %d", len(channels), tc.numFan)
		}

		for i := range channels {
			go func() {
				for out := range channels[i] {
					select {
					case <-ctx.Done():
						return
					default:
						if out != tc.want {
							t.Errorf("FanOut: %d, want %d", out, tc.want)
						}
					}
				}
			}()
		}

		cancel()
	}
}

func TestFanInIntSlice(t *testing.T) {
	cases := []struct {
		in     []int
		fn     func(context.Context, int) int
		numFan int
		want   int
	}{
		{
			in:     []int{TestInt, TestInt, TestInt, TestInt, TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     []int{TestInt, TestInt, TestInt, TestInt, TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			numFan: 1,
			want:   TestInt,
		},
		{
			in:     []int{TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     []int{TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			numFan: 1,
			want:   TestInt,
		},
		{
			in:     []int{},
			fn:     func(ctx context.Context, data int) int { return data },
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     []int{},
			fn:     func(ctx context.Context, data int) int { return data },
			numFan: 1,
			want:   TestInt,
		},
	}

	for _, tc := range cases {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

		stream := StreamSlice(ctx, tc.in)
		fanOut := FanOut(ctx, stream, tc.fn, tc.numFan)
		outStream := FanIn(ctx, fanOut...)

		count := 0
		for out := range outStream {
			count++

			if out != tc.want {
				t.Errorf("FanIn: %v, want %v", out, tc.want)
			}
		}

		if count != len(tc.in) {
			t.Errorf("FanIn: missing data, len() %v, found %v", len(tc.in), count)
		}

		cancel()
	}
}

func TestFanInIntMap(t *testing.T) {
	cases := []struct {
		in     map[int]int
		fn     func(context.Context, int) int
		numFan int
		want   int
	}{
		{
			in:     map[int]int{1: TestInt, 2: TestInt, 3: TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     map[int]int{1: TestInt, 2: TestInt, 3: TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			numFan: 1,
			want:   TestInt,
		},
		{
			in:     map[int]int{1: TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     map[int]int{1: TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			numFan: 1,
			want:   TestInt,
		},
		{
			in:     map[int]int{},
			fn:     func(ctx context.Context, data int) int { return data },
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     map[int]int{},
			fn:     func(ctx context.Context, data int) int { return data },
			numFan: 1,
			want:   TestInt,
		},
	}

	for _, tc := range cases {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

		stream := StreamMap(ctx, tc.in)
		fanOut := FanOut(ctx, stream, tc.fn, tc.numFan)
		outStream := FanIn(ctx, fanOut...)

		count := 0
		for out := range outStream {
			count++

			if out != tc.want {
				t.Errorf("FanIn: %v, want %v", out, tc.want)
			}
		}

		if count != len(tc.in) {
			t.Errorf("FanIn: missing data, len() %v, found %v", len(tc.in), count)
		}

		cancel()
	}
}
