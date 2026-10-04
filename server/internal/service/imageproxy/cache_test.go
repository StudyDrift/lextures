package imageproxy

import (
	"bytes"
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

type memStore struct {
	mu   sync.Mutex
	objs map[string][]byte
}

func newMemStore() *memStore {
	return &memStore{objs: map[string][]byte{}}
}

func (m *memStore) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.objs[key]
	if !ok {
		return nil, errCacheMiss
	}
	return append([]byte(nil), b...), nil
}

func (m *memStore) Put(_ context.Context, key string, data []byte, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objs[key] = append([]byte(nil), data...)
	return nil
}

var errCacheMiss = errString("miss")

type errString string

func (e errString) Error() string { return string(e) }

func TestCache_SecondViewDoesNotReadMaster(t *testing.T) {
	src := mustJPEG(t, 640, 160)
	master := append([]byte(nil), src...)
	loads := 0
	store := newMemStore()
	c := Cache{Store: store, Group: &singleflight.Group{}}
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	opts := ResizeOpts{MaxWidth: 320, MaxHeight: 80, Quality: 85, Format: FormatWebP}
	load := func(context.Context) ([]byte, error) {
		loads++
		return append([]byte(nil), src...), nil
	}

	out1, ct1, err := c.GetOrCreate(context.Background(), id, opts, "image/jpeg", load)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	out2, ct2, err := c.GetOrCreate(context.Background(), id, opts, "image/jpeg", load)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if loads != 1 {
		t.Fatalf("master reads = %d want 1", loads)
	}
	if ct1 != "image/webp" || ct2 != "image/webp" {
		t.Fatalf("content types = %s %s want image/webp", ct1, ct2)
	}
	if !bytes.Equal(out1, out2) {
		t.Fatal("cached bytes differ from the first derivative")
	}
	if !bytes.Equal(src, master) {
		t.Fatal("master bytes were modified")
	}
	if _, ok := store.objs[DerivativeKey(id, opts, FormatWebP)]; !ok {
		t.Fatal("webp derivative was not stored")
	}
}

func TestCache_SVGStaysOriginalAndIsCached(t *testing.T) {
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"></svg>`)
	loads := 0
	c := Cache{Store: newMemStore(), Group: &singleflight.Group{}}
	id := uuid.New()
	opts := BannerOpts(FormatWebP)
	load := func(context.Context) ([]byte, error) {
		loads++
		return append([]byte(nil), svg...), nil
	}
	for i := 0; i < 2; i++ {
		out, ct, err := c.GetOrCreate(context.Background(), id, opts, "image/svg+xml", load)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if ct != "image/svg+xml" {
			t.Fatalf("content type = %q", ct)
		}
		if !bytes.Equal(out, svg) {
			t.Fatal("svg bytes changed")
		}
	}
	if loads != 1 {
		t.Fatalf("master reads = %d want 1", loads)
	}
}

func TestDerivativeKey_RaisesQualityFloor(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	key := DerivativeKey(id, ResizeOpts{MaxWidth: BannerWidth, MaxHeight: BannerHeight, Quality: 40, Format: FormatWebP}, FormatWebP)
	want := "image-derivatives/v1/550e8400-e29b-41d4-a716-446655440000/1920x384/q80/webp"
	if key != want {
		t.Fatalf("key = %s want %s", key, want)
	}
}

func TestFormatFromAccept(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{header: "", want: FormatJPEG},
		{header: "*/*", want: FormatJPEG},
		{header: "image/jpeg", want: FormatJPEG},
		{header: "image/webp,image/jpeg;q=0.8", want: FormatWebP},
		{header: "image/webp;q=0,image/jpeg", want: FormatJPEG},
		{header: "image/webp;q=0.0", want: FormatJPEG},
	}
	for _, tc := range cases {
		if got := FormatFromAccept(tc.header); got != tc.want {
			t.Fatalf("Accept %q = %s want %s", tc.header, got, tc.want)
		}
	}
}
