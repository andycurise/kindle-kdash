// Package display provides the small set of image transforms needed by the
// cross-platform host renderer. Device probing, framebuffer access and eips execution live
// in the Kindle pull client, not in this package.
package display

import (
	"fmt"
	"image"
	"image/draw"
)

// RotateForDevice rotates a landscape render into the Kindle's portrait
// orientation. Only 90 and 270 degrees are supported; any other direction
// intentionally falls back to 270 so a bad setting cannot mirror the image.
func RotateForDevice(src image.Image, dir int) image.Image {
	sb := src.Bounds()
	w, h := sb.Dx(), sb.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, h, w))

	for dy := 0; dy < w; dy++ {
		for dx := 0; dx < h; dx++ {
			sx, sy := dy, h-1-dx
			if dir == 90 {
				sx, sy = w-1-dy, dx
			}
			dst.Set(dx, dy, src.At(sb.Min.X+sx, sb.Min.Y+sy))
		}
	}
	return dst
}

// GrayImage converts any source image to the 8-bit grayscale format consumed
// by the Kindle client.
func GrayImage(src image.Image) *image.Gray {
	dst := image.NewGray(image.Rect(0, 0, src.Bounds().Dx(), src.Bounds().Dy()))
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	return dst
}

// ParseScreenSize parses a positive WIDTHxHEIGHT value such as 758x1024.
func ParseScreenSize(value string) (int, int, error) {
	var w, h int
	if _, err := fmt.Sscanf(value, "%dx%d", &w, &h); err != nil || w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("屏幕尺寸应为正数，例如 758x1024")
	}
	return w, h, nil
}
