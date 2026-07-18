package domain

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	MaxImageAttachmentBytes = 10 << 20 // 10 MiB decoded
)

var allowedImageMimes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/jpg":  true,
	"image/webp": true,
	"image/gif":  true,
}

// NormalizeUserAttachments validates and normalizes image attachments for LLM vision.
func NormalizeUserAttachments(atts []UserAttachment) ([]UserAttachment, error) {
	if len(atts) == 0 {
		return nil, nil
	}
	out := make([]UserAttachment, 0, len(atts))
	for i, a := range atts {
		typ := strings.ToLower(strings.TrimSpace(a.Type))
		if typ == "" {
			typ = "image"
		}
		if typ != "image" {
			return nil, fmt.Errorf("attachments[%d]: unsupported type %q", i, a.Type)
		}
		mime, data, err := parseImageData(a.MimeType, a.Data)
		if err != nil {
			return nil, fmt.Errorf("attachments[%d]: %w", i, err)
		}
		raw, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			// try raw URL-encoding-safe alphabet
			raw, err = base64.RawStdEncoding.DecodeString(data)
			if err != nil {
				return nil, fmt.Errorf("attachments[%d]: invalid base64: %w", i, err)
			}
		}
		if len(raw) == 0 {
			return nil, fmt.Errorf("attachments[%d]: empty image data", i)
		}
		if len(raw) > MaxImageAttachmentBytes {
			return nil, fmt.Errorf("attachments[%d]: image exceeds %d bytes", i, MaxImageAttachmentBytes)
		}
		// Re-encode as standard base64 for providers.
		out = append(out, UserAttachment{
			Type:     "image",
			Name:     a.Name,
			MimeType: mime,
			Data:     base64.StdEncoding.EncodeToString(raw),
		})
	}
	return out, nil
}

func parseImageData(mimeType, data string) (mime string, b64 string, err error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return "", "", fmt.Errorf("missing image data")
	}
	mime = strings.ToLower(strings.TrimSpace(mimeType))
	if strings.HasPrefix(data, "data:") {
		// data:image/png;base64,xxxx
		rest := strings.TrimPrefix(data, "data:")
		parts := strings.SplitN(rest, ",", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid data URL")
		}
		meta := parts[0]
		b64 = parts[1]
		if i := strings.Index(meta, ";"); i >= 0 {
			if mime == "" {
				mime = meta[:i]
			}
		} else if mime == "" {
			mime = meta
		}
	} else {
		b64 = data
	}
	if mime == "image/jpg" {
		mime = "image/jpeg"
	}
	if mime == "" {
		mime = "image/png"
	}
	if !allowedImageMimes[mime] {
		return "", "", fmt.Errorf("unsupported mime type %q", mime)
	}
	// strip whitespace/newlines from base64
	b64 = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, b64)
	return mime, b64, nil
}
