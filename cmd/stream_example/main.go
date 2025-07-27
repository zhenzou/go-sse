package main

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/tmaxmax/go-sse"
)

type exampleReadCloser struct {
	io.Reader
}

func (e *exampleReadCloser) Close() error {
	return nil
}

func main() {
	// Example SSE stream data
	sseData := `id: 1
event: message
data: Hello, World!

id: 2
event: notification
data: This is a notification
data: with multiple lines

id: 3
data: Simple message without event type

`

	// Create a stream from the data
	reader := &exampleReadCloser{Reader: strings.NewReader(sseData)}
	stream := sse.NewStream(reader)
	defer stream.Close()

	fmt.Println("Reading SSE events from stream:")
	fmt.Println("================================")

	// Read events one by one
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("End of stream reached")
			break
		}
		if err != nil {
			log.Fatalf("Error reading event: %v", err)
		}

		fmt.Printf("Event ID: %q\n", event.LastEventID)
		fmt.Printf("Event Type: %q\n", event.Type)
		fmt.Printf("Event Data: %q\n", event.Data)
		fmt.Println("---")
	}

	fmt.Println("\nExample with configuration:")
	fmt.Println("===========================")

	// Example with configuration
	reader2 := &exampleReadCloser{Reader: strings.NewReader("data: Small event\n\n")}
	config := &sse.StreamConfig{
		MaxEventSize: 1024, // 1KB max event size
	}
	stream2 := sse.NewStreamWithConfig(reader2, config)
	defer stream2.Close()

	event, err := stream2.Recv()
	if err != nil && err != io.EOF {
		log.Fatalf("Error reading event: %v", err)
	}
	if err != io.EOF {
		fmt.Printf("Configured stream event: %q\n", event.Data)
	}
}