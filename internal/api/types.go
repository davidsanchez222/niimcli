package api

import (
	"encoding/base64"
	"strings"
)

type Layout string

const (
	LayoutFullImage       Layout = "full-image"
	LayoutQROnly          Layout = "qr-only"
	LayoutQRTitle         Layout = "qr-title"
	LayoutQRTitleSubtitle Layout = "qr-title-subtitle"
)

type PrintRequest struct {
	Printer PrinterSelector `json:"printer"`
	Label   LabelRequest    `json:"label"`
	Image   ImageRequest    `json:"image,omitempty"`
	QR      QRRequest       `json:"qr,omitempty"`
	Content ContentRequest  `json:"content,omitempty"`
	Options PrintOptions    `json:"options,omitempty"`
}

type PrinterSelector struct {
	Selector string `json:"selector"`
}

type LabelRequest struct {
	Preset string `json:"preset"`
	Layout Layout `json:"layout,omitempty"`
}

type ImageRequest struct {
	PNGBase64 Base64PNG `json:"png_base64"`
}

type QRRequest struct {
	PNGBase64 Base64PNG `json:"png_base64"`
}

type ContentRequest struct {
	Title    string `json:"title,omitempty"`
	Subtitle string `json:"subtitle,omitempty"`
}

type PrintOptions struct {
	Copies int `json:"copies,omitempty"`
}

type PrintResponse struct {
	OK      bool        `json:"ok"`
	Printer string      `json:"printer,omitempty"`
	Preset  string      `json:"preset,omitempty"`
	Copies  int         `json:"copies,omitempty"`
	Error   *ErrorBody  `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Base64PNG []byte

func (b Base64PNG) MarshalJSON() ([]byte, error) {
	encoded := base64.StdEncoding.EncodeToString(b)
	return []byte("\"" + encoded + "\""), nil
}

func (b *Base64PNG) UnmarshalJSON(data []byte) error {
	raw := strings.Trim(string(data), "\"")
	if raw == "" || raw == "null" {
		*b = nil
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return err
	}
	*b = Base64PNG(decoded)
	return nil
}
