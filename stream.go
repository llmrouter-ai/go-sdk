package arouter

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// ErrStreamDone is returned by Recv when the stream has ended normally.
var ErrStreamDone = errors.New("arouter: stream done")

// ChatCompletionStream reads server-sent events from a streaming chat
// completion response.
type ChatCompletionStream struct {
	resp    *http.Response
	scanner *bufio.Scanner
	done    bool
}

func newChatCompletionStream(resp *http.Response) *ChatCompletionStream {
	return &ChatCompletionStream{
		resp:    resp,
		scanner: bufio.NewScanner(resp.Body),
	}
}

// Recv reads the next chunk from the stream. Returns ErrStreamDone when the
// server signals completion with "data: [DONE]". Returns io.EOF if the
// underlying connection closes unexpectedly.
func (s *ChatCompletionStream) Recv() (*ChatCompletionChunk, error) {
	if s.done {
		return nil, ErrStreamDone
	}

	for s.scanner.Scan() {
		line := s.scanner.Text()

		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			s.done = true
			return nil, ErrStreamDone
		}

		var chunk ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, err
		}
		return &chunk, nil
	}

	if err := s.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

// Close releases the underlying HTTP response body.
func (s *ChatCompletionStream) Close() error {
	s.done = true
	return s.resp.Body.Close()
}
