package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"

	"niimcli/internal/api"
	"niimcli/internal/config"
)

func TestValidateRequestUsesDefaultPresetAndFullImageDefaultLayout(t *testing.T) {
	svc := mustService(t)
	req := api.PrintRequest{
		Printer: api.PrinterSelector{Selector: "d110-desk"},
		Image:   api.ImageRequest{PNGBase64: testPNG(t, 320, 96)},
	}

	printer, preset, errResp := svc.validateRequest(req)
	if errResp != nil {
		t.Fatalf("validateRequest() unexpected error = %#v", errResp)
	}
	if printer.Name != "d110-desk" {
		t.Fatalf("printer = %q, want d110-desk", printer.Name)
	}
	if preset.Name != "d110-12x40" {
		t.Fatalf("preset = %q, want d110-12x40", preset.Name)
	}
	if got := normalizedLayout(req.Label.Layout); got != api.LayoutFullImage {
		t.Fatalf("normalizedLayout() = %q, want %q", got, api.LayoutFullImage)
	}
}

func TestValidateRequestAcceptsLegacyQRField(t *testing.T) {
	svc := mustService(t)
	req := api.PrintRequest{
		Printer: api.PrinterSelector{Selector: "d110-desk"},
		QR:      api.QRRequest{PNGBase64: testPNG(t, 320, 96)},
	}

	_, _, errResp := svc.validateRequest(req)
	if errResp != nil {
		t.Fatalf("validateRequest() unexpected error for legacy qr field = %#v", errResp)
	}
}

func TestValidateRequestRejectsUnknownPrinter(t *testing.T) {
	svc := mustService(t)
	req := api.PrintRequest{
		Printer: api.PrinterSelector{Selector: "missing"},
		Image:   api.ImageRequest{PNGBase64: testPNG(t, 320, 96)},
	}

	_, _, errResp := svc.validateRequest(req)
	if errResp == nil || errResp.Error == nil {
		t.Fatalf("validateRequest() expected error")
	}
	if errResp.Error.Code != ErrPrinterNotFound {
		t.Fatalf("error code = %q, want %q", errResp.Error.Code, ErrPrinterNotFound)
	}
}

func TestValidateRequestRejectsInvalidImage(t *testing.T) {
	svc := mustService(t)
	req := api.PrintRequest{
		Printer: api.PrinterSelector{Selector: "d110-desk"},
		Image:   api.ImageRequest{PNGBase64: api.Base64PNG([]byte("not-a-png"))},
	}

	_, _, errResp := svc.validateRequest(req)
	if errResp == nil || errResp.Error == nil {
		t.Fatalf("validateRequest() expected error")
	}
	if errResp.Error.Code != ErrInvalidImage {
		t.Fatalf("error code = %q, want %q", errResp.Error.Code, ErrInvalidImage)
	}
}

func TestRenderPreviewProducesPNG(t *testing.T) {
	svc := mustService(t)
	req := api.PrintRequest{
		Printer: api.PrinterSelector{Selector: "b1-round"},
		Label:   api.LabelRequest{Preset: "round-40mm"},
		Image:   api.ImageRequest{PNGBase64: testPNG(t, 320, 320)},
	}

	preview, err := svc.RenderPreview(context.Background(), req)
	if err != nil {
		t.Fatalf("RenderPreview() error = %v", err)
	}
	if len(preview) == 0 {
		t.Fatal("RenderPreview() returned empty preview")
	}
	if _, err := png.Decode(bytes.NewReader(preview)); err != nil {
		t.Fatalf("preview is not a valid PNG: %v", err)
	}
}

func TestResolvePrinterIncludesIdentifierProfiles(t *testing.T) {
	svc := mustService(t)
	printer, errResp := svc.resolvePrinter("b1-round")
	if errResp != nil {
		t.Fatalf("resolvePrinter() unexpected error = %#v", errResp)
	}
	if printer.Identifier == "" {
		t.Fatal("resolvePrinter() returned printer without identifier")
	}
}

func mustService(t *testing.T) *Service {
	t.Helper()
	cfg := config.Config{
		Server: config.ServerConfig{Listen: "127.0.0.1:8443", AuthToken: "test-token"},
		Printers: []config.PrinterProfile{
			{
				Name:          "d110-desk",
				Model:         "D110",
				Transport:     "ble",
				DeviceName:    "D110_M-H913040249",
				Identifier:    "ea88cc93-a2a2-8287-1f08-18cd0a81649b",
				DefaultPreset: "d110-12x40",
				Defaults:      config.PrinterDefaults{Density: 3},
			},
			{
				Name:          "b1-round",
				Model:         "B1",
				Transport:     "ble",
				DeviceName:    "B1-I427031488",
				Identifier:    "e6bc3bef-5a60-3bc7-ffab-50edc0e9f122",
				DefaultPreset: "round-40mm",
				Defaults:      config.PrinterDefaults{Density: 3},
			},
		},
		Presets: []config.LabelPreset{
			{Name: "d110-12x40", WidthMM: 40, HeightMM: 12, Shape: "rect", Layout: "full-image", MarginsMM: 1},
			{Name: "round-40mm", WidthMM: 40, HeightMM: 40, Shape: "round", Layout: "full-image", MarginsMM: 2},
		},
	}
	svc, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return svc
}

func testPNG(t *testing.T, width, height int) api.Base64PNG {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	for y := height / 4; y < (height/4)*3; y++ {
		for x := width / 4; x < (width/4)*3; x++ {
			img.SetGray(x, y, color.Gray{Y: 0})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return api.Base64PNG(buf.Bytes())
}
