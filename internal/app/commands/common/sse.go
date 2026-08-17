package common

import (
	"bufio"
	"io"
	"strings"
)

const SSE_CONTENT_TYPE = "text/event-stream"

const SSE_DEFAULT_EVENT = "message"

// SseEvent is a single event dispatched by a Server-Sent Events stream.
type SseEvent struct {
	Id    string
	Event string
	Data  string
}

// SseReader parses a Server-Sent Events stream one event at a time.
type SseReader struct {
	reader *bufio.Reader
}

func NewSseReader(stream io.Reader) *SseReader {
	return &SseReader{reader: bufio.NewReader(stream)}
}

// Next reads the stream until the next event is dispatched. Comments, unknown
// fields and events without data are skipped, as is an incomplete event at the
// end of the stream. Returns io.EOF once the stream has no more events.
func (r *SseReader) Next() (*SseEvent, error) {
	var (
		id, event string
		data      []string
	)

	for {
		line, err := r.reader.ReadString('\n')
		if line == "" && err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")

		switch {
		case line == "":
			// a blank line dispatches the event, unless there is nothing to dispatch
			if len(data) > 0 {
				if event == "" {
					event = SSE_DEFAULT_EVENT
				}
				return &SseEvent{Id: id, Event: event, Data: strings.Join(data, "\n")}, nil
			}
			id, event, data = "", "", nil
		case strings.HasPrefix(line, ":"):
			// comment
		default:
			// "retry" and unknown fields are ignored
			switch name, value := splitSseField(line); name {
			case "id":
				id = value
			case "event":
				event = value
			case "data":
				data = append(data, value)
			}
		}

		if err != nil {
			return nil, err
		}
	}
}

func splitSseField(line string) (string, string) {
	name, value, found := strings.Cut(line, ":")
	if !found {
		return name, ""
	}
	return name, strings.TrimPrefix(value, " ")
}
