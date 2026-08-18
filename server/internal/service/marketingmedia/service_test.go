package marketingmedia

import (
	"context"
	"errors"
	"image"
	"testing"
)

func TestCreateRequiresAltBeforeStorage(t *testing.T) {
	_, _, err := (&Service{}).Create(context.Background(), Upload{Data: []byte("anything")})
	if !errors.Is(err, ErrAltRequired) {
		t.Fatalf("got %v", err)
	}
}

func TestCoverCropProducesSocialCardSize(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 200, 100))
	got := coverCrop(src, socialCardWidth, socialCardHeight)
	if got.Bounds().Dx() != socialCardWidth || got.Bounds().Dy() != socialCardHeight {
		t.Fatalf("got %dx%d", got.Bounds().Dx(), got.Bounds().Dy())
	}
}

func TestCreateRejectsSpoofedPDF(t *testing.T) {
	_, _, err := (&Service{}).Create(context.Background(), Upload{Data: []byte("%PDF-1.7 fake"), AltText: "A useful description"})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("got %v", err)
	}
}
