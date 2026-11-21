package display

import (
	"image"
	"image/color"
	"testing"
)

func TestRotateForDeviceDimensionsAndPixels(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			src.SetRGBA(x, y, color.RGBA{R: uint8(10*x + y), G: 20, B: 30, A: 255})
		}
	}

	rotated := RotateForDevice(src, 90)
	if got, want := rotated.Bounds().Size(), (image.Point{X: 2, Y: 3}); got != want {
		t.Fatalf("90° 尺寸=%v，期望 %v", got, want)
	}
	for _, check := range []struct{ x, y, sx, sy int }{
		{0, 0, 2, 0}, {1, 0, 2, 1}, {0, 2, 0, 0}, {1, 2, 0, 1},
	} {
		if got, want := rotated.At(check.x, check.y), src.At(check.sx, check.sy); got != want {
			t.Errorf("90° 像素 (%d,%d)=%v，期望 (%d,%d)=%v", check.x, check.y, got, check.sx, check.sy, want)
		}
	}

	reverse := RotateForDevice(src, 270)
	for _, check := range []struct{ x, y, sx, sy int }{
		{0, 0, 0, 1}, {1, 0, 0, 0}, {0, 2, 2, 1}, {1, 2, 2, 0},
	} {
		if got, want := reverse.At(check.x, check.y), src.At(check.sx, check.sy); got != want {
			t.Errorf("270° 像素 (%d,%d)=%v，期望 (%d,%d)=%v", check.x, check.y, got, check.sx, check.sy, want)
		}
	}
}

func TestRotateForDeviceInvalidDirectionUses270(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, color.Black)
	src.Set(1, 0, color.White)
	got := RotateForDevice(src, 123)
	if got.At(0, 0) != src.At(0, 0) || got.At(0, 1) != src.At(1, 0) {
		t.Fatal("非法方向应按 270° 旋转")
	}
}

func TestGrayImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(4, 7, 6, 8))
	src.Set(4, 7, color.White)
	src.Set(5, 7, color.Black)
	got := GrayImage(src)
	if got.Bounds() != image.Rect(0, 0, 2, 1) {
		t.Fatalf("灰度图 bounds=%v", got.Bounds())
	}
	if got.GrayAt(0, 0).Y != 255 || got.GrayAt(1, 0).Y != 0 {
		t.Fatalf("灰度像素错误: %v %v", got.GrayAt(0, 0), got.GrayAt(1, 0))
	}
}

func TestParseScreenSize(t *testing.T) {
	w, h, err := ParseScreenSize("758x1024")
	if err != nil || w != 758 || h != 1024 {
		t.Fatalf("ParseScreenSize=(%d,%d,%v)", w, h, err)
	}
	for _, value := range []string{"", "0x1024", "758x0", "bad", "758*1024"} {
		if _, _, err := ParseScreenSize(value); err == nil {
			t.Errorf("%q 应返回错误", value)
		}
	}
}
