package pipelines

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// benchSizes is the set of input lengths each benchmark sweeps over.
var benchSizes = []int{100, 1_000, 10_000}

// benchFans is the set of fan-out widths the FanOut/FanIn benchmarks sweep over.
var benchFans = []int{1, 4, 8, 16}

// benchID is the no-op process function used by the fan benchmarks. It
// isolates pipeline/channel overhead from any user workload.
func benchID(ctx context.Context, x int) int { return x }

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
	for _, n := range benchSizes {
		data := makeSlice(n)

		b.Run(fmt.Sprintf("size=%d", n), func(b *testing.B) {
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
