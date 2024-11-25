package pipelines

import (
	"context"
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
		ctx, cancel := context.WithCancel(context.Background())

		stream := GenerateStream(ctx, tc.in)

		select {
		case <-ctx.Done():
			continue
		case out := <-stream:
			if out != tc.want {
				t.Errorf("GenerateStream: %v, want %v", out, tc.want)
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
	}

	for _, tc := range cases {
		ctx, cancel := context.WithCancel(context.Background())

		stream := StreamSlice(ctx, tc.in)

		count := 0
		for out := range stream {
			count++

			if out != tc.want {
				t.Errorf("StreamSlice: %v, want %v", out, tc.want)
			}
		}

		if count != len(tc.in) {
			t.Errorf("StreamSlice: missing data, len() %v, found %v", len(tc.in), count)
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
	}

	for _, tc := range cases {
		ctx, cancel := context.WithCancel(context.Background())

		stream := StreamMap(ctx, tc.in)

		count := 0
		for out := range stream {
			count++

			if out != tc.want {
				t.Errorf("StreamMap: %v, want %v", out, tc.want)
			}
		}

		if count != len(tc.in) {
			t.Errorf("StreamMap: missing data, len() %v, found %v", len(tc.in), count)
		}

		cancel()
	}
}

func TestFanOutInt(t *testing.T) {
	cases := []struct {
		in     chan int
		fn     func(context.Context, int) int
		numFan int
		want   int
	}{
		{
			in:     make(chan int, 1),
			fn:     func(ctx context.Context, data int) int { return data },
			numFan: 3,
			want:   TestInt,
		},
	}

	for _, tc := range cases {
		ctx, cancel := context.WithCancel(context.Background())

		tc.in <- TestInt
		close(tc.in)

		channels := FanOut(ctx, tc.in, tc.fn, tc.numFan)

		if len(channels) != tc.numFan {
			t.Errorf("FanOut: number of channels %v, numFan %v", len(channels), tc.numFan)
		}

		for i := range channels {
			for out := range channels[i] {
				if out != tc.want {
					t.Errorf("FanOut: %v, want %v", out, tc.want)
				}
			}
		}

		cancel()
	}
}
