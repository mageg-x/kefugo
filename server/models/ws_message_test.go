package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseWSMessagePayload_Text(t *testing.T) {
	raw := json.RawMessage(`{"content_type":"text","content":"hello"}`)
	payload, err := ParseWSMessagePayload(raw)
	if err != nil {
		t.Fatalf("ParseWSMessagePayload text failed: %v", err)
	}
	if payload.ContentType != WSContentTypeText {
		t.Fatalf("unexpected content type: %s", payload.ContentType)
	}
	if payload.Content != "hello" {
		t.Fatalf("unexpected content: %s", payload.Content)
	}
}

func TestParseWSMessagePayload_AudioRequiresAudioMediaType(t *testing.T) {
	okRaw := json.RawMessage(`{"content_type":"audio","url":"data:audio/webm;base64,AA==","duration":12}`)
	if _, err := ParseWSMessagePayload(okRaw); err != nil {
		t.Fatalf("audio payload should pass: %v", err)
	}

	badRaw := json.RawMessage(`{"content_type":"audio","url":"data:application/octet-stream;base64,AA==","duration":12}`)
	if _, err := ParseWSMessagePayload(badRaw); err == nil {
		t.Fatalf("audio payload with non-audio media type should fail")
	}
}

func TestParseWSMessagePayload_FileRejectsImageAndAudioMediaType(t *testing.T) {
	okRaw := json.RawMessage(`{"content_type":"file","url":"data:application/pdf;base64,AA==","name":"a.pdf","size":12}`)
	if _, err := ParseWSMessagePayload(okRaw); err != nil {
		t.Fatalf("file payload should pass: %v", err)
	}

	imageBad := json.RawMessage(`{"content_type":"file","url":"data:image/png;base64,AA==","name":"a.png","size":12}`)
	if _, err := ParseWSMessagePayload(imageBad); err == nil {
		t.Fatalf("file payload with image media type should fail")
	}

	audioBad := json.RawMessage(`{"content_type":"file","url":"data:audio/mp3;base64,AA==","name":"a.mp3","size":12}`)
	if _, err := ParseWSMessagePayload(audioBad); err == nil {
		t.Fatalf("file payload with audio media type should fail")
	}
}

func TestParseWSMessagePayload_Limits(t *testing.T) {
	tooLongName := json.RawMessage(`{"content_type":"file","url":"https://example.com/file","name":"` + strings.Repeat("a", MaxFileNameBytes+1) + `","size":12}`)
	if _, err := ParseWSMessagePayload(tooLongName); err == nil {
		t.Fatalf("too long file name should fail")
	}

	tooLarge := json.RawMessage(`{"content_type":"file","url":"https://example.com/file","name":"f","size":99999999}`)
	if _, err := ParseWSMessagePayload(tooLarge); err == nil {
		t.Fatalf("too large file should fail")
	}

	audioNegativeDuration := json.RawMessage(`{"content_type":"audio","url":"https://example.com/a","duration":-1}`)
	if _, err := ParseWSMessagePayload(audioNegativeDuration); err == nil {
		t.Fatalf("negative audio duration should fail")
	}
}
