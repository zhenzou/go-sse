package sse_test

import (
	"io"
	"strings"
	"testing"

	"github.com/tmaxmax/go-sse"
	"github.com/tmaxmax/go-sse/internal/tests"
)

type testReadCloser struct {
	io.Reader
	closed bool
}

func (t *testReadCloser) Close() error {
	t.closed = true
	return nil
}

func newTestReadCloser(s string) *testReadCloser {
	return &testReadCloser{Reader: strings.NewReader(s)}
}

func TestNewStream(t *testing.T) {
	r := newTestReadCloser("data: test\n\n")
	stream := sse.NewStream(r)
	tests.Expect(t, stream != nil, "NewStream should return a non-nil stream")
	
	// Test that we can receive an event
	event, err := stream.Recv()
	tests.Equal(t, err, nil, "unexpected error")
	tests.Equal(t, event.Data, "test", "unexpected event data")
	
	// Clean up
	err = stream.Close()
	tests.Equal(t, err, nil, "unexpected error closing stream")
	tests.Expect(t, r.closed, "underlying reader should be closed")
}

func TestNewStreamWithConfig(t *testing.T) {
	r := newTestReadCloser("data: test\n\n")
	cfg := &sse.StreamConfig{MaxEventSize: 1024}
	stream := sse.NewStreamWithConfig(r, cfg)
	tests.Expect(t, stream != nil, "NewStreamWithConfig should return a non-nil stream")
	
	// Test that we can receive an event
	event, err := stream.Recv()
	tests.Equal(t, err, nil, "unexpected error")
	tests.Equal(t, event.Data, "test", "unexpected event data")
	
	// Clean up
	err = stream.Close()
	tests.Equal(t, err, nil, "unexpected error closing stream")
}

func TestStream_Recv_SimpleEvent(t *testing.T) {
	r := newTestReadCloser("data: Hello World!\n\n")
	stream := sse.NewStream(r)
	defer stream.Close()

	event, err := stream.Recv()
	tests.Equal(t, err, nil, "unexpected error")
	tests.Equal(t, event.Data, "Hello World!", "unexpected event data")
	tests.Equal(t, event.Type, "", "unexpected event type")
	tests.Equal(t, event.LastEventID, "", "unexpected event ID")
}

func TestStream_Recv_MultipleEvents(t *testing.T) {
	input := "data: First event\n\ndata: Second event\n\ndata: Third event\n\n"
	r := newTestReadCloser(input)
	stream := sse.NewStream(r)
	defer stream.Close()

	// First event
	event, err := stream.Recv()
	tests.Equal(t, err, nil, "unexpected error for first event")
	tests.Equal(t, event.Data, "First event", "unexpected first event data")

	// Second event
	event, err = stream.Recv()
	tests.Equal(t, err, nil, "unexpected error for second event")
	tests.Equal(t, event.Data, "Second event", "unexpected second event data")

	// Third event
	event, err = stream.Recv()
	tests.Equal(t, err, nil, "unexpected error for third event")
	tests.Equal(t, event.Data, "Third event", "unexpected third event data")

	// EOF
	_, err = stream.Recv()
	tests.Equal(t, err, io.EOF, "expected EOF after all events")
}

func TestStream_Recv_EventWithTypeAndID(t *testing.T) {
	input := "id: 123\nevent: message\ndata: Hello World!\n\n"
	r := newTestReadCloser(input)
	stream := sse.NewStream(r)
	defer stream.Close()

	event, err := stream.Recv()
	tests.Equal(t, err, nil, "unexpected error")
	tests.Equal(t, event.Data, "Hello World!", "unexpected event data")
	tests.Equal(t, event.Type, "message", "unexpected event type")
	tests.Equal(t, event.LastEventID, "123", "unexpected event ID")
}

func TestStream_Recv_MultilineData(t *testing.T) {
	input := "data: Line 1\ndata: Line 2\ndata: Line 3\n\n"
	r := newTestReadCloser(input)
	stream := sse.NewStream(r)
	defer stream.Close()

	event, err := stream.Recv()
	tests.Equal(t, err, nil, "unexpected error")
	expected := "Line 1\nLine 2\nLine 3"
	tests.Equal(t, event.Data, expected, "unexpected multiline event data")
}

func TestStream_Recv_LastEventIDPersistence(t *testing.T) {
	input := "id: 1\ndata: First\n\ndata: Second\n\nid: 3\ndata: Third\n\n"
	r := newTestReadCloser(input)
	stream := sse.NewStream(r)
	defer stream.Close()

	// First event with ID
	event, err := stream.Recv()
	tests.Equal(t, err, nil, "unexpected error for first event")
	tests.Equal(t, event.Data, "First", "unexpected first event data")
	tests.Equal(t, event.LastEventID, "1", "unexpected first event ID")

	// Second event should inherit the last ID
	event, err = stream.Recv()
	tests.Equal(t, err, nil, "unexpected error for second event")
	tests.Equal(t, event.Data, "Second", "unexpected second event data")
	tests.Equal(t, event.LastEventID, "1", "second event should inherit last ID")

	// Third event with new ID
	event, err = stream.Recv()
	tests.Equal(t, err, nil, "unexpected error for third event")
	tests.Equal(t, event.Data, "Third", "unexpected third event data")
	tests.Equal(t, event.LastEventID, "3", "unexpected third event ID")
}

func TestStream_Recv_InvalidID(t *testing.T) {
	// ID with null byte should be ignored
	input := "id: \x00invalid\ndata: Test\n\n"
	r := newTestReadCloser(input)
	stream := sse.NewStream(r)
	defer stream.Close()

	event, err := stream.Recv()
	tests.Equal(t, err, nil, "unexpected error")
	tests.Equal(t, event.Data, "Test", "unexpected event data")
	tests.Equal(t, event.LastEventID, "", "ID with null byte should be ignored")
}

func TestStream_Recv_RetryField(t *testing.T) {
	input := "retry: 1000\ndata: Test\n\n"
	r := newTestReadCloser(input)
	stream := sse.NewStream(r)
	defer stream.Close()

	event, err := stream.Recv()
	tests.Equal(t, err, nil, "unexpected error")
	tests.Equal(t, event.Data, "Test", "unexpected event data")
	// Retry field is parsed but not exposed in the Event struct
}

func TestStream_Recv_InvalidRetryField(t *testing.T) {
	input := "retry: invalid\ndata: Test\n\n"
	r := newTestReadCloser(input)
	stream := sse.NewStream(r)
	defer stream.Close()

	event, err := stream.Recv()
	tests.Equal(t, err, nil, "unexpected error")
	tests.Equal(t, event.Data, "Test", "unexpected event data")
}

func TestStream_Recv_EmptyData(t *testing.T) {
	input := "data:\n\n"
	r := newTestReadCloser(input)
	stream := sse.NewStream(r)
	defer stream.Close()

	event, err := stream.Recv()
	tests.Equal(t, err, nil, "unexpected error")
	tests.Equal(t, event.Data, "", "unexpected event data")
}

func TestStream_Recv_EventAtEOF(t *testing.T) {
	// Event without final double newline (at EOF)
	input := "data: Final event"
	r := newTestReadCloser(input)
	stream := sse.NewStream(r)
	defer stream.Close()

	// First, let's test what the Read function does with this input
	r2 := strings.NewReader(input)
	var readEvents []sse.Event
	var readErr error
	events := sse.Read(r2, nil)
	events(func(e sse.Event, err error) bool {
		if err != nil {
			readErr = err
			return false
		}
		readEvents = append(readEvents, e)
		return true
	})

	// If Read function successfully parses this, then Stream should too
	if readErr == nil && len(readEvents) > 0 {
		event, err := stream.Recv()
		tests.Equal(t, err, nil, "unexpected error")
		tests.Equal(t, event.Data, "Final event", "unexpected event data")

		// Next call should return EOF
		_, err = stream.Recv()
		tests.Equal(t, err, io.EOF, "expected EOF after final event")
	} else {
		// If Read function doesn't parse this, then Stream should return EOF immediately
		_, err := stream.Recv()
		tests.Equal(t, err, io.EOF, "expected EOF for incomplete event")
	}
}

func TestStream_Recv_EmptyStream(t *testing.T) {
	r := newTestReadCloser("")
	stream := sse.NewStream(r)
	defer stream.Close()

	_, err := stream.Recv()
	tests.Equal(t, err, io.EOF, "expected EOF for empty stream")
}

func TestStream_Close(t *testing.T) {
	r := newTestReadCloser("data: test\n\n")
	stream := sse.NewStream(r)

	// Close the stream
	err := stream.Close()
	tests.Equal(t, err, nil, "unexpected error closing stream")
	tests.Expect(t, r.closed, "underlying reader should be closed")

	// Subsequent calls to Recv should return EOF
	_, err = stream.Recv()
	tests.Equal(t, err, io.EOF, "expected EOF after closing stream")

	// Subsequent calls to Close should not error
	err = stream.Close()
	tests.Equal(t, err, nil, "unexpected error on second close")
}

func TestStream_Recv_AfterClose(t *testing.T) {
	r := newTestReadCloser("data: test\n\n")
	stream := sse.NewStream(r)

	// Close first
	err := stream.Close()
	tests.Equal(t, err, nil, "unexpected error closing stream")

	// Then try to receive
	_, err = stream.Recv()
	tests.Equal(t, err, io.EOF, "expected EOF when receiving from closed stream")
}

func TestStreamConfig_MaxEventSize(t *testing.T) {
	// Create an event that's larger than the buffer
	largeData := strings.Repeat("x", 100)
	input := "data: " + largeData + "\n\n"
	
	r := newTestReadCloser(input)
	cfg := &sse.StreamConfig{MaxEventSize: 10} // Very small buffer
	stream := sse.NewStreamWithConfig(r, cfg)
	defer stream.Close()

	_, err := stream.Recv()
	tests.Expect(t, err != nil, "should fail with small buffer size")
}

func TestStream_Recv_ComplexEvent(t *testing.T) {
	input := `id: event-1
event: user-message
data: {"user": "john", "message": "Hello"}
data: {"timestamp": "2023-01-01T00:00:00Z"}

`
	r := newTestReadCloser(input)
	stream := sse.NewStream(r)
	defer stream.Close()

	event, err := stream.Recv()
	tests.Equal(t, err, nil, "unexpected error")
	tests.Equal(t, event.LastEventID, "event-1", "unexpected event ID")
	tests.Equal(t, event.Type, "user-message", "unexpected event type")
	expected := `{"user": "john", "message": "Hello"}
{"timestamp": "2023-01-01T00:00:00Z"}`
	tests.Equal(t, event.Data, expected, "unexpected event data")
}

func TestStream_Recv_CommentFields(t *testing.T) {
	input := `: This is a comment
data: Hello
: Another comment
data: World

`
	r := newTestReadCloser(input)
	stream := sse.NewStream(r)
	defer stream.Close()

	event, err := stream.Recv()
	tests.Equal(t, err, nil, "unexpected error")
	tests.Equal(t, event.Data, "Hello\nWorld", "comments should be ignored")
}

// Benchmark tests
func BenchmarkStream_Recv_SimpleEvent(b *testing.B) {
	input := "data: Hello World!\n\n"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := newTestReadCloser(input)
		stream := sse.NewStream(r)
		_, err := stream.Recv()
		if err != nil {
			b.Fatal(err)
		}
		stream.Close()
	}
}

func BenchmarkStream_Recv_ComplexEvent(b *testing.B) {
	input := `id: event-1
event: user-message
data: {"user": "john", "message": "Hello"}
data: {"timestamp": "2023-01-01T00:00:00Z"}

`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := newTestReadCloser(input)
		stream := sse.NewStream(r)
		_, err := stream.Recv()
		if err != nil {
			b.Fatal(err)
		}
		stream.Close()
	}
}