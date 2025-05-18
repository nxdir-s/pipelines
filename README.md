[![Go Reference](https://pkg.go.dev/badge/github.com/nxdir-s/pipelines.svg)](https://pkg.go.dev/github.com/nxdir-s/pipelines)
[![Go Report Card](https://goreportcard.com/badge/github.com/nxdir-s/pipelines)](https://goreportcard.com/report/github.com/nxdir-s/pipelines)
[![Go Coverage](https://github.com/nxdir-s/pipelines/wiki/coverage.svg)](https://raw.githack.com/wiki/nxdir-s/pipelines/coverage.html)

# pipelines

Pipelines contains generic functions that help with concurrent processing

## Usage

A pipeline can be created from a slice or map

```go
stream := pipelines.StreamSlice(ctx, data)
```

Or from a generator function

```go
func GenerateData(ctx context.Context) int { return rand.Intn(10) }

stream := pipelines.GenerateStream(ctx, GenerateData)
```

### FanOut

`FanOut` can be used to process data concurrently. Useful for I/O bound processes, but it can be used in any situation where you have a slice or map of data and want to introduce concurrency

```go
const MaxFan int = 3

fanOutChannels := pipelines.FanOut(ctx, stream, ProcessFunc, MaxFan)
```

### FanIn

`FanIn` can be used to merge data into one channel

```go
fanInData := pipelines.FanIn(ctx, fanOutChannels...)
```

## Example

```go
package main

import (
    "context"
    "fmt"
    "math/rand"
    "os"
    "os/signal"
    "strconv"
    "time"

    "github.com/nxdir-s/pipelines"
)

const (
    MaxFan int = 3
)

func GenerateData(ctx context.Context) int {
    return rand.Intn(5)
}

func Process(ctx context.Context, timeout int) string {
    select {
    case <-ctx.Done():
        return "context cancelled"
    case <-time.After(time.Second * time.Duration(timeout)):
        return "slept for " + strconv.Itoa(timeout) + " seconds!"
    }
}

func Read(ctx context.Context, messages <-chan string) {
    for msg := range messages {
        select {
        case <-ctx.Done():
            return
        default:
            fmt.Fprintf(os.Stdout, "%s\n", msg)
        }
    }
}

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
    defer cancel()

    stream := pipelines.GenerateStream(ctx, GenerateData)
    fanOutChannels := pipelines.FanOut(ctx, stream, Process, MaxFan)
    messages := pipelines.FanIn(ctx, fanOutChannels...)

    go Read(ctx, messages)

    select {
    case <-ctx.Done():
        fmt.Fprint(os.Stdout, "context canceled, exiting...\n")
        os.Exit(0)
    }
}
```
