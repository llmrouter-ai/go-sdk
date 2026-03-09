package arouter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// CreateTranscription sends an audio file for speech-to-text transcription.
//
//	resp, err := client.CreateTranscription(ctx, &arouter.TranscriptionRequest{
//	    Model:    "openai/whisper-1",
//	    FilePath: "recording.webm",
//	})
//	fmt.Println(resp.Text)
func (c *Client) CreateTranscription(ctx context.Context, req *TranscriptionRequest) (*TranscriptionResponse, error) {
	body, contentType, err := buildAudioMultipart(req.FilePath, req.FileReader, req.FileName, func(w *multipart.Writer) error {
		if err := w.WriteField("model", req.Model); err != nil {
			return err
		}
		if req.Language != "" {
			if err := w.WriteField("language", req.Language); err != nil {
				return err
			}
		}
		if req.Prompt != "" {
			if err := w.WriteField("prompt", req.Prompt); err != nil {
				return err
			}
		}
		if req.ResponseFormat != "" {
			if err := w.WriteField("response_format", req.ResponseFormat); err != nil {
				return err
			}
		}
		if req.Temperature != nil {
			if err := w.WriteField("temperature", fmt.Sprintf("%g", *req.Temperature)); err != nil {
				return err
			}
		}
		for _, g := range req.TimestampGranularities {
			if err := w.WriteField("timestamp_granularities[]", g); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var resp TranscriptionResponse
	if err := c.doMultipart(ctx, "/v1/audio/transcriptions", body, contentType, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateTranslation sends an audio file for speech-to-English translation.
//
//	resp, err := client.CreateTranslation(ctx, &arouter.TranslationRequest{
//	    Model:    "openai/whisper-1",
//	    FilePath: "recording.webm",
//	})
//	fmt.Println(resp.Text)
func (c *Client) CreateTranslation(ctx context.Context, req *TranslationRequest) (*TranslationResponse, error) {
	body, contentType, err := buildAudioMultipart(req.FilePath, req.FileReader, req.FileName, func(w *multipart.Writer) error {
		if err := w.WriteField("model", req.Model); err != nil {
			return err
		}
		if req.Prompt != "" {
			if err := w.WriteField("prompt", req.Prompt); err != nil {
				return err
			}
		}
		if req.ResponseFormat != "" {
			if err := w.WriteField("response_format", req.ResponseFormat); err != nil {
				return err
			}
		}
		if req.Temperature != nil {
			if err := w.WriteField("temperature", fmt.Sprintf("%g", *req.Temperature)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var resp TranslationResponse
	if err := c.doMultipart(ctx, "/v1/audio/translations", body, contentType, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// buildAudioMultipart constructs a multipart/form-data body with the audio file
// and additional fields written by writeFields.
func buildAudioMultipart(filePath string, fileReader io.Reader, fileName string, writeFields func(*multipart.Writer) error) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	var src io.Reader
	var name string

	if filePath != "" {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, "", fmt.Errorf("arouter: open audio file: %w", err)
		}
		defer f.Close()
		src = f
		name = filepath.Base(filePath)
	} else if fileReader != nil {
		src = fileReader
		name = fileName
		if name == "" {
			name = "audio.webm"
		}
	} else {
		return nil, "", fmt.Errorf("arouter: either FilePath or FileReader must be provided")
	}

	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		return nil, "", fmt.Errorf("arouter: create form file: %w", err)
	}
	if _, err := io.Copy(part, src); err != nil {
		return nil, "", fmt.Errorf("arouter: copy audio data: %w", err)
	}

	if err := writeFields(writer); err != nil {
		return nil, "", fmt.Errorf("arouter: write form fields: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("arouter: close multipart writer: %w", err)
	}

	return buf.Bytes(), writer.FormDataContentType(), nil
}

func (c *Client) doMultipart(ctx context.Context, path string, body []byte, contentType string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("arouter: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", contentType)

	return c.do(req, dst)
}
