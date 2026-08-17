package common

import (
	"bufio"
	"io"
	"strings"
)

const SSE_CONTENT_TYPE = "text/event-stream"

const SSE_DEFAULT_EVENT = "message"

type SseEvent struct {
	Id    string
	Event string
	Data  string
}

// bufio.Reader and not Scanner: a data line can exceed Scanner's 64KB token limit
type SseReader struct {
	reader *bufio.Reader
	// the spec keeps the last event id across events, only the server can change it
	lastEventId string
}

func NewSseReader(stream io.Reader) *SseReader {
	return &SseReader{reader: bufio.NewReader(stream)}
}

func (r *SseReader) Next() (*SseEvent, error) {
	var (
		event string
		data  []string
	)

	for {
		line, err := r.reader.ReadString('\n')
		if line == "" && err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")

		switch {
		case line == "":
			if len(data) > 0 {
				if event == "" {
					event = SSE_DEFAULT_EVENT
				}
				return &SseEvent{Id: r.lastEventId, Event: event, Data: strings.Join(data, "\n")}, nil
			}
			event, data = "", nil
		case !strings.HasPrefix(line, ":"):
			switch name, value := splitSseField(line); name {
			case "id":
				r.lastEventId = value
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
