package domain

import (
	"encoding/base64"
	"testing"
)

func TestNormalizeUserAttachments_DataURL(t *testing.T) {
	raw := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	b64 := base64.StdEncoding.EncodeToString(raw)
	atts, err := NormalizeUserAttachments([]UserAttachment{{
		Type: "image",
		Name: "x.png",
		Data: "data:image/png;base64," + b64,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(atts) != 1 || atts[0].MimeType != "image/png" || atts[0].Data != b64 {
		t.Fatalf("unexpected: %+v", atts)
	}
}

func TestNormalizeUserAttachments_RejectsBadMime(t *testing.T) {
	_, err := NormalizeUserAttachments([]UserAttachment{{
		Type:     "image",
		MimeType: "application/pdf",
		Data:     base64.StdEncoding.EncodeToString([]byte("x")),
	}})
	if err == nil {
		t.Fatal("expected error")
	}
}
