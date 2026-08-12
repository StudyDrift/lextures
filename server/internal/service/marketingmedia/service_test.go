package marketingmedia

import (
	"context"
	"errors"
	"testing"
)

func TestCreateRequiresAltBeforeStorage(t *testing.T) {
	_, _, err := (&Service{}).Create(context.Background(), Upload{Data: []byte("anything")})
	if !errors.Is(err, ErrAltRequired) {
		t.Fatalf("got %v", err)
	}
}

func TestCreateRejectsSpoofedPDF(t *testing.T) {
	_, _, err := (&Service{}).Create(context.Background(), Upload{Data: []byte("%PDF-1.7 fake"), AltText: "A useful description"})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("got %v", err)
	}
}
