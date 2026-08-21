package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"

	"niimcli/internal/api"
	"niimcli/internal/config"
)

const dotsPerMM = 8.0

type Result struct {
	Image        image.Image
	WidthPx      int
	HeightPx     int
	PrintablePx  image.Rectangle
	Shape        string
	Rotation     int
	PreviewPNG   []byte
	PreviewBytes int
}

func QRLabel(req api.PrintRequest, printer config.PrinterProfile, preset config.LabelPreset) (Result, error) {
	if preset.WidthMM <= 0 || preset.HeightMM <= 0 {
		return Result{}, fmt.Errorf("invalid preset dimensions")
	}

	sourceImage, err := png.Decode(bytes.NewReader(sourcePNG(req)))
	if err != nil {
		return Result{}, fmt.Errorf("decode source png: %w", err)
	}

	widthPx := mmToPx(preset.WidthMM)
	heightPx := mmToPx(preset.HeightMM)
	marginPx := mmToPx(preset.MarginsMM)
	if widthPx <= 0 || heightPx <= 0 {
		return Result{}, fmt.Errorf("invalid output size")
	}

	printable := image.Rect(marginPx, marginPx, widthPx-marginPx, heightPx-marginPx)
	if printable.Dx() <= 0 || printable.Dy() <= 0 {
		return Result{}, fmt.Errorf("preset margins leave no printable area")
	}

	canvas := image.NewGray(image.Rect(0, 0, widthPx, heightPx))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)

	if sourceImage.Bounds().Dx() == widthPx && sourceImage.Bounds().Dy() == heightPx {
		scaleNearest(canvas, canvas.Bounds(), sourceImage, sourceImage.Bounds())
	} else {
		qrBounds := fitCentered(sourceImage.Bounds(), printable)
		scaleNearest(canvas, qrBounds, sourceImage, sourceImage.Bounds())
	}
	thresholdToMonochrome(canvas)
	if preset.Shape == "round" {
		maskRound(canvas)
	}

	rotation := normalizedRotation(printer.Defaults.Rotate)
	if rotation != 0 {
		canvas = rotateGray(canvas, rotation)
		printable = rotateRect(printable, widthPx, heightPx, rotation)
		if rotation == 90 || rotation == 270 {
			widthPx, heightPx = heightPx, widthPx
		}
	}

	var preview bytes.Buffer
	if err := png.Encode(&preview, canvas); err != nil {
		return Result{}, fmt.Errorf("encode preview: %w", err)
	}

	return Result{
		Image:        canvas,
		WidthPx:      widthPx,
		HeightPx:     heightPx,
		PrintablePx:  printable,
		Shape:        preset.Shape,
		Rotation:     rotation,
		PreviewPNG:   preview.Bytes(),
		PreviewBytes: preview.Len(),
	}, nil
}

func sourcePNG(req api.PrintRequest) []byte {
	if len(req.Image.PNGBase64) > 0 {
		return req.Image.PNGBase64
	}
	return req.QR.PNGBase64
}

func mmToPx(mm float64) int {
	return int(math.Round(mm * dotsPerMM))
}

func fitCentered(src image.Rectangle, dst image.Rectangle) image.Rectangle {
	srcW := src.Dx()
	srcH := src.Dy()
	if srcW <= 0 || srcH <= 0 {
		return image.Rect(dst.Min.X, dst.Min.Y, dst.Min.X, dst.Min.Y)
	}

	scale := math.Min(float64(dst.Dx())/float64(srcW), float64(dst.Dy())/float64(srcH))
	if scale <= 0 {
		return image.Rect(dst.Min.X, dst.Min.Y, dst.Min.X, dst.Min.Y)
	}

	outW := int(math.Round(float64(srcW) * scale))
	outH := int(math.Round(float64(srcH) * scale))
	if outW < 1 {
		outW = 1
	}
	if outH < 1 {
		outH = 1
	}

	offsetX := dst.Min.X + (dst.Dx()-outW)/2
	offsetY := dst.Min.Y + (dst.Dy()-outH)/2
	return image.Rect(offsetX, offsetY, offsetX+outW, offsetY+outH)
}

func scaleNearest(dst draw.Image, dstRect image.Rectangle, src image.Image, srcRect image.Rectangle) {
	if dstRect.Dx() <= 0 || dstRect.Dy() <= 0 || srcRect.Dx() <= 0 || srcRect.Dy() <= 0 {
		return
	}

	for y := dstRect.Min.Y; y < dstRect.Max.Y; y++ {
		sy := srcRect.Min.Y + ((y-dstRect.Min.Y)*srcRect.Dy())/dstRect.Dy()
		for x := dstRect.Min.X; x < dstRect.Max.X; x++ {
			sx := srcRect.Min.X + ((x-dstRect.Min.X)*srcRect.Dx())/dstRect.Dx()
			dst.Set(x, y, src.At(sx, sy))
		}
	}
}

func thresholdToMonochrome(img *image.Gray) {
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if img.GrayAt(x, y).Y < 128 {
				img.SetGray(x, y, color.Gray{Y: 0})
			} else {
				img.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
}

func maskRound(img *image.Gray) {
	b := img.Bounds()
	cx := float64(b.Min.X+b.Max.X-1) / 2
	cy := float64(b.Min.Y+b.Max.Y-1) / 2
	radius := math.Min(float64(b.Dx()), float64(b.Dy())) / 2
	radiusSq := radius * radius

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			if dx*dx+dy*dy > radiusSq {
				img.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
}

func normalizedRotation(rotate int) int {
	r := rotate % 360
	if r < 0 {
		r += 360
	}
	switch r {
	case 0, 90, 180, 270:
		return r
	default:
		return 0
	}
}

func rotateGray(src *image.Gray, rotation int) *image.Gray {
	rotation = normalizedRotation(rotation)
	if rotation == 0 {
		return src
	}

	sb := src.Bounds()
	var dst *image.Gray
	if rotation == 90 || rotation == 270 {
		dst = image.NewGray(image.Rect(0, 0, sb.Dy(), sb.Dx()))
	} else {
		dst = image.NewGray(image.Rect(0, 0, sb.Dx(), sb.Dy()))
	}

	for y := sb.Min.Y; y < sb.Max.Y; y++ {
		for x := sb.Min.X; x < sb.Max.X; x++ {
			v := src.GrayAt(x, y)
			sx := x - sb.Min.X
			sy := y - sb.Min.Y
			switch rotation {
			case 90:
				dst.SetGray(sb.Dy()-1-sy, sx, v)
			case 180:
				dst.SetGray(sb.Dx()-1-sx, sb.Dy()-1-sy, v)
			case 270:
				dst.SetGray(sy, sb.Dx()-1-sx, v)
			}
		}
	}

	return dst
}

func rotateRect(r image.Rectangle, width, height, rotation int) image.Rectangle {
	rotation = normalizedRotation(rotation)
	if rotation == 0 {
		return r
	}

	points := []image.Point{
		{X: r.Min.X, Y: r.Min.Y},
		{X: r.Max.X, Y: r.Min.Y},
		{X: r.Min.X, Y: r.Max.Y},
		{X: r.Max.X, Y: r.Max.Y},
	}
	rotated := make([]image.Point, 0, len(points))
	for _, p := range points {
		switch rotation {
		case 90:
			rotated = append(rotated, image.Point{X: height - p.Y, Y: p.X})
		case 180:
			rotated = append(rotated, image.Point{X: width - p.X, Y: height - p.Y})
		case 270:
			rotated = append(rotated, image.Point{X: p.Y, Y: width - p.X})
		}
	}

	minX, minY := rotated[0].X, rotated[0].Y
	maxX, maxY := rotated[0].X, rotated[0].Y
	for _, p := range rotated[1:] {
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	if minX < 0 || minY < 0 {
		shiftX, shiftY := 0, 0
		if minX < 0 {
			shiftX = -minX
		}
		if minY < 0 {
			shiftY = -minY
		}
		minX += shiftX
		maxX += shiftX
		minY += shiftY
		maxY += shiftY
	}

	return image.Rect(minX, minY, maxX, maxY)
}
