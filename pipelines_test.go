package pipelines

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"testing"
)

var benchSizes = []int{100, 1_000, 10_000}

var benchFans = []int{1, 4, 8, 16}

var benchBuffers = []int{1, 64, 256}

func benchID(ctx context.Context, x int) int {
	return x
}

func makeSlice(n int) []int {
	data := make([]int, n)

	for i := range data {
		data[i] = TestInt
	}

	return data
}

func makeMap(n int) map[int]int {
	data := make(map[int]int, n)

	for i := 0; i < n; i++ {
		data[i] = TestInt
	}

	return data
}

func BenchmarkGenerateStream(b *testing.B) {
	gen := func(ctx context.Context) int { return TestInt }

	for i := range benchSizes {
		b.Run(fmt.Sprintf("size=%d", benchSizes[i]), func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				ctx, cancel := context.WithCancel(context.Background())

				stream := GenerateStream(ctx, gen)

				for range benchSizes[i] {
					<-stream
				}

				cancel()
			}
		})
	}
}

func BenchmarkStreamSlice(b *testing.B) {
	for i := range benchSizes {
		data := makeSlice(benchSizes[i])

		b.Run(fmt.Sprintf("size=%d", benchSizes[i]), func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				ctx, cancel := context.WithCancel(context.Background())

				for range StreamSlice(ctx, data) {
				}

				cancel()
			}
		})
	}
}

func BenchmarkStreamMap(b *testing.B) {
	for i := range benchSizes {
		data := makeMap(benchSizes[i])

		b.Run(fmt.Sprintf("size=%d", benchSizes[i]), func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				ctx, cancel := context.WithCancel(context.Background())

				for range StreamMap(ctx, data) {
				}

				cancel()
			}
		})
	}
}

func BenchmarkFanOut(b *testing.B) {
	for i := range benchSizes {
		data := makeSlice(benchSizes[i])

		b.Run(fmt.Sprintf("size=%d", benchSizes[i]), func(b *testing.B) {
			for _, fan := range benchFans {
				b.Run(fmt.Sprintf("fan=%d", fan), func(b *testing.B) {
					b.ReportAllocs()

					for b.Loop() {
						ctx, cancel := context.WithCancel(context.Background())

						stream := StreamSlice(ctx, data)
						channels := FanOut(ctx, stream, benchID, fan)

						// drain every fan channel so producers complete each iteration
						var wg sync.WaitGroup
						for i := range channels {
							wg.Add(1)
							go func(c <-chan int) {
								defer wg.Done()
								for range c {
								}
							}(channels[i])
						}

						wg.Wait()
						cancel()
					}
				})
			}
		})
	}
}

func BenchmarkFanIn(b *testing.B) {
	for i := range benchSizes {
		data := makeSlice(benchSizes[i])

		b.Run(fmt.Sprintf("size=%d", benchSizes[i]), func(b *testing.B) {
			for _, fan := range benchFans {
				b.Run(fmt.Sprintf("fan=%d", fan), func(b *testing.B) {
					b.ReportAllocs()

					for b.Loop() {
						ctx, cancel := context.WithCancel(context.Background())

						stream := StreamSlice(ctx, data)
						fanOut := FanOut(ctx, stream, benchID, fan)

						for range FanIn(ctx, fanOut...) {
						}

						cancel()
					}
				})
			}
		})
	}
}

func BenchmarkPipeline(b *testing.B) {
	for i := range benchSizes {
		data := makeSlice(benchSizes[i])

		b.Run(fmt.Sprintf("size=%d", benchSizes[i]), func(b *testing.B) {
			for _, fan := range benchFans {
				b.Run(fmt.Sprintf("fan=%d", fan), func(b *testing.B) {
					b.ReportAllocs()

					for b.Loop() {
						ctx, cancel := context.WithCancel(context.Background())

						stream := StreamSlice(ctx, data)
						fanOut := FanOut(ctx, stream, benchID, fan)
						merged := FanIn(ctx, fanOut...)

						for range merged {
						}

						cancel()
					}
				})
			}
		})
	}
}

func BenchmarkPipelineBuffer(b *testing.B) {
	for i := range benchSizes {
		data := makeSlice(benchSizes[i])

		b.Run(fmt.Sprintf("size=%d", benchSizes[i]), func(b *testing.B) {
			for _, fan := range benchFans {
				b.Run(fmt.Sprintf("fan=%d", fan), func(b *testing.B) {
					for n := range benchBuffers {
						b.Run(fmt.Sprintf("buf=%d", benchBuffers[n]), func(b *testing.B) {
							b.ReportAllocs()

							for b.Loop() {
								ctx, cancel := context.WithCancel(context.Background())

								stream := StreamSlice(ctx, data)
								fanOut := FanOut(ctx, stream, benchID, fan)
								merged := FanInBuffer(ctx, benchBuffers[n], fanOut...)

								for range merged {
								}

								cancel()
							}
						})
					}
				})
			}
		})
	}
}

const (
	TestInt int = 1
)

func TestGenerateStreamInt(t *testing.T) {
	cases := []struct {
		in      func(context.Context) int
		want    int
		wantOut int
	}{
		{
			in:      func(ctx context.Context) int { return TestInt },
			want:    TestInt,
			wantOut: 0,
		},
		{
			in:      func(ctx context.Context) int { return TestInt },
			want:    TestInt,
			wantOut: 1,
		},
		{
			in:      func(ctx context.Context) int { return TestInt },
			want:    TestInt,
			wantOut: 10,
		},
		{
			in:      func(ctx context.Context) int { return TestInt },
			want:    TestInt,
			wantOut: 100,
		},
	}

	for _, tc := range cases {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

		stream := GenerateStream(ctx, tc.in)

		count := 0
		for range tc.wantOut {
			select {
			case <-ctx.Done():
				continue
			case out := <-stream:
				count++

				if out != tc.want {
					t.Errorf("GenerateStream: %d, want %d", out, tc.want)
				}
			}
		}

		if count != tc.wantOut {
			t.Errorf("GenerateStream: missing data, wantOut %d, gotOut %d", tc.wantOut, count)
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
					tc.in <- tc.want
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
				t.Errorf("FanIn: %d, want %d", out, tc.want)
			}
		}

		if count != len(tc.in) {
			t.Errorf("FanIn: missing data, len() %d, found %d", len(tc.in), count)
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
				t.Errorf("FanIn: %d, want %d", out, tc.want)
			}
		}

		if count != len(tc.in) {
			t.Errorf("FanIn: missing data, len() %d, found %d", len(tc.in), count)
		}

		cancel()
	}
}

func TestFanOutBufferInt(t *testing.T) {
	cases := []struct {
		in     chan int
		fn     func(context.Context, int) int
		buffer int
		numIn  int
		numFan int
		want   int
	}{
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numIn:  1,
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numIn:  1,
			numFan: 1,
			want:   TestInt,
		},
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numIn:  10,
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numIn:  10,
			numFan: 1,
			want:   TestInt,
		},
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numIn:  0,
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numIn:  0,
			numFan: 1,
			want:   TestInt,
		},
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numIn:  100,
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     make(chan int),
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
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
					tc.in <- tc.want
				}
			}
		}()

		channels := FanOutBuffer(ctx, tc.buffer, tc.in, tc.fn, tc.numFan)

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

func TestFanInBufferIntSlice(t *testing.T) {
	cases := []struct {
		in     []int
		fn     func(context.Context, int) int
		buffer int
		numFan int
		want   int
	}{
		{
			in:     []int{TestInt, TestInt, TestInt, TestInt, TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     []int{TestInt, TestInt, TestInt, TestInt, TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numFan: 1,
			want:   TestInt,
		},
		{
			in:     []int{TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     []int{TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numFan: 1,
			want:   TestInt,
		},
		{
			in:     []int{},
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     []int{},
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numFan: 1,
			want:   TestInt,
		},
	}

	for _, tc := range cases {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

		stream := StreamSlice(ctx, tc.in)
		fanOut := FanOut(ctx, stream, tc.fn, tc.numFan)
		outStream := FanInBuffer(ctx, tc.buffer, fanOut...)

		count := 0
		for out := range outStream {
			count++

			if out != tc.want {
				t.Errorf("FanIn: %d, want %d", out, tc.want)
			}
		}

		if count != len(tc.in) {
			t.Errorf("FanIn: missing data, len() %d, found %d", len(tc.in), count)
		}

		cancel()
	}
}

func TestFanInBufferIntMap(t *testing.T) {
	cases := []struct {
		in     map[int]int
		fn     func(context.Context, int) int
		buffer int
		numFan int
		want   int
	}{
		{
			in:     map[int]int{1: TestInt, 2: TestInt, 3: TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     map[int]int{1: TestInt, 2: TestInt, 3: TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numFan: 1,
			want:   TestInt,
		},
		{
			in:     map[int]int{1: TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     map[int]int{1: TestInt},
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numFan: 1,
			want:   TestInt,
		},
		{
			in:     map[int]int{},
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numFan: 3,
			want:   TestInt,
		},
		{
			in:     map[int]int{},
			fn:     func(ctx context.Context, data int) int { return data },
			buffer: 3,
			numFan: 1,
			want:   TestInt,
		},
	}

	for _, tc := range cases {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

		stream := StreamMap(ctx, tc.in)
		fanOut := FanOut(ctx, stream, tc.fn, tc.numFan)
		outStream := FanInBuffer(ctx, tc.buffer, fanOut...)

		count := 0
		for out := range outStream {
			count++

			if out != tc.want {
				t.Errorf("FanIn: %d, want %d", out, tc.want)
			}
		}

		if count != len(tc.in) {
			t.Errorf("FanIn: missing data, len() %d, found %d", len(tc.in), count)
		}

		cancel()
	}
}
