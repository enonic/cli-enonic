package common

import (
	"io"
	"strings"
	"testing"
)

func TestSseReaderParsesEvents(t *testing.T) {
	stream := "id: 1\nevent: list\ndata: {\"applications\":[]}\n\n" +
		"id: 2\nevent: state\ndata: {\"key\":\"com.enonic.app.superhero\"}\n\n"

	reader := NewSseReader(strings.NewReader(stream))

	first, err := reader.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first.Id != "1" || first.Event != "list" || first.Data != `{"applications":[]}` {
		t.Errorf("unexpected first event: %+v", first)
	}

	second, err := reader.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if second.Id != "2" || second.Event != "state" || second.Data != `{"key":"com.enonic.app.superhero"}` {
		t.Errorf("unexpected second event: %+v", second)
	}

	if _, err = reader.Next(); err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestSseReaderSkipsCommentsAndUnknownFields(t *testing.T) {
	stream := ": ping\n\nretry: 3000\nnonsense: value\nevent: list\ndata: {}\n\n"

	event, err := NewSseReader(strings.NewReader(stream)).Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Event != "list" || event.Data != "{}" {
		t.Errorf("unexpected event: %+v", event)
	}
}

func TestSseReaderJoinsMultilineData(t *testing.T) {
	stream := "event: list\ndata: {\ndata: }\n\n"

	event, err := NewSseReader(strings.NewReader(stream)).Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Data != "{\n}" {
		t.Errorf("unexpected data: %q", event.Data)
	}
}

func TestSseReaderDefaultsEventName(t *testing.T) {
	event, err := NewSseReader(strings.NewReader("data: hello\n\n")).Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Event != SSE_DEFAULT_EVENT {
		t.Errorf("expected %q, got %q", SSE_DEFAULT_EVENT, event.Event)
	}
}

func TestSseReaderHandlesCarriageReturns(t *testing.T) {
	event, err := NewSseReader(strings.NewReader("event: list\r\ndata: {}\r\n\r\n")).Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Event != "list" || event.Data != "{}" {
		t.Errorf("unexpected event: %+v", event)
	}
}

func TestSseReaderFieldWithoutSpaceAndWithoutValue(t *testing.T) {
	event, err := NewSseReader(strings.NewReader("event:list\ndata:{}\nid\n\n")).Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Event != "list" || event.Data != "{}" || event.Id != "" {
		t.Errorf("unexpected event: %+v", event)
	}
}

func TestSseReaderDiscardsIncompleteEvent(t *testing.T) {
	// no blank line at the end of the stream, so the event is never dispatched
	if _, err := NewSseReader(strings.NewReader("event: list\ndata: {}\n")).Next(); err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestSseReaderReadsDataLongerThanScannerLimit(t *testing.T) {
	payload := strings.Repeat("a", 128*1024)

	event, err := NewSseReader(strings.NewReader("event: list\ndata: " + payload + "\n\n")).Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Data != payload {
		t.Errorf("expected %d bytes of data, got %d", len(payload), len(event.Data))
	}
}
