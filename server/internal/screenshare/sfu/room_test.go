package sfu

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
)

type noopSignal struct{}

func (noopSignal) SendOffer(uuid.UUID, string)                          {}
func (noopSignal) SendAnswer(uuid.UUID, string)                         {}
func (noopSignal) SendICE(uuid.UUID, webrtc.ICECandidateInit)           {}

func TestRegistryBookkeeping(t *testing.T) {
	reg := NewRegistry()
	id := uuid.New()
	r := reg.GetOrCreate(id, nil, noopSignal{})
	if reg.Len() != 1 {
		t.Fatalf("len %d", reg.Len())
	}
	if reg.Get(id) != r {
		t.Fatal("get mismatch")
	}
	viewer := uuid.New()
	if err := r.AttachViewer(viewer); err != nil {
		t.Fatal(err)
	}
	if r.ViewerCount() != 1 {
		t.Fatal(r.ViewerCount())
	}
	r.Detach(viewer)
	if r.ViewerCount() != 0 {
		t.Fatal(r.ViewerCount())
	}
	reg.Delete(id)
	if reg.Len() != 0 {
		t.Fatal(reg.Len())
	}
}
