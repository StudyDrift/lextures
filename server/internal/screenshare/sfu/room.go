package sfu

import (
	"fmt"
	"io"
	"sync"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
)

// SignalOut is how the SFU asks the transport to deliver SDP/ICE to a peer.
type SignalOut interface {
	SendOffer(peerID uuid.UUID, sdp string)
	SendAnswer(peerID uuid.UUID, sdp string)
	SendICE(peerID uuid.UUID, candidate webrtc.ICECandidateInit)
}

// Room is a one-presenter / N-viewer Selective Forwarding Unit for a session.
type Room struct {
	mu         sync.Mutex
	sessionID  uuid.UUID
	iceServers []webrtc.ICEServer
	signal     SignalOut

	presenterPC    *webrtc.PeerConnection
	presenterID    uuid.UUID
	presenterTrack *webrtc.TrackLocalStaticRTP

	viewers map[uuid.UUID]*viewerPeer
}

type viewerPeer struct {
	pc     *webrtc.PeerConnection
	sender *webrtc.RTPSender
}

// NewRoom creates an empty SFU room. iceServers may be empty for loopback tests.
func NewRoom(sessionID uuid.UUID, iceServers []webrtc.ICEServer, signal SignalOut) *Room {
	return &Room{
		sessionID:  sessionID,
		iceServers: iceServers,
		signal:     signal,
		viewers:    make(map[uuid.UUID]*viewerPeer),
	}
}

func (r *Room) api() (*webrtc.API, error) {
	m := &webrtc.MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		return nil, err
	}
	s := webrtc.SettingEngine{}
	return webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithSettingEngine(s)), nil
}

func (r *Room) newPC() (*webrtc.PeerConnection, error) {
	api, err := r.api()
	if err != nil {
		return nil, err
	}
	return api.NewPeerConnection(webrtc.Configuration{ICEServers: r.iceServers})
}

// AttachPresenter wires a peer as the upstream publisher. Replaces any prior presenter.
func (r *Room) AttachPresenter(peerID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.presenterPC != nil {
		_ = r.presenterPC.Close()
		r.presenterPC = nil
		r.presenterTrack = nil
	}

	pc, err := r.newPC()
	if err != nil {
		return err
	}
	r.presenterPC = pc
	r.presenterID = peerID

	pc.OnTrack(func(remote *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		r.mu.Lock()
		local, err := webrtc.NewTrackLocalStaticRTP(remote.Codec().RTPCodecCapability, remote.ID(), remote.StreamID())
		if err != nil {
			r.mu.Unlock()
			return
		}
		r.presenterTrack = local
		viewers := make([]*viewerPeer, 0, len(r.viewers))
		for _, v := range r.viewers {
			viewers = append(viewers, v)
		}
		r.mu.Unlock()

		for _, v := range viewers {
			r.attachTrackToViewer(v, local)
		}

		go func() {
			buf := make([]byte, 1500)
			for {
				n, _, readErr := remote.Read(buf)
				if readErr != nil {
					return
				}
				if _, writeErr := local.Write(buf[:n]); writeErr != nil && writeErr != io.ErrClosedPipe {
					return
				}
			}
		}()
	})

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil || r.signal == nil {
			return
		}
		r.signal.SendICE(peerID, c.ToJSON())
	})

	return nil
}

func (r *Room) attachTrackToViewer(v *viewerPeer, track *webrtc.TrackLocalStaticRTP) {
	if v == nil || v.pc == nil || track == nil {
		return
	}
	if v.sender != nil {
		_ = v.pc.RemoveTrack(v.sender)
		v.sender = nil
	}
	sender, err := v.pc.AddTrack(track)
	if err != nil {
		return
	}
	v.sender = sender
}

// AttachViewer creates a downstream peer that will receive the presenter's track.
func (r *Room) AttachViewer(peerID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.viewers[peerID]; ok && existing.pc != nil {
		_ = existing.pc.Close()
		delete(r.viewers, peerID)
	}

	pc, err := r.newPC()
	if err != nil {
		return err
	}
	v := &viewerPeer{pc: pc}
	r.viewers[peerID] = v

	if r.presenterTrack != nil {
		r.attachTrackToViewer(v, r.presenterTrack)
	}

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil || r.signal == nil {
			return
		}
		r.signal.SendICE(peerID, c.ToJSON())
	})

	return nil
}

// HandleOffer processes a remote SDP offer and returns an answer SDP.
func (r *Room) HandleOffer(peerID uuid.UUID, sdp string) (string, error) {
	r.mu.Lock()
	pc := r.pcFor(peerID)
	r.mu.Unlock()
	if pc == nil {
		return "", fmt.Errorf("sfu: unknown peer %s", peerID)
	}
	if err := pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}); err != nil {
		return "", err
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return "", err
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		return "", err
	}
	if r.signal != nil {
		r.signal.SendAnswer(peerID, answer.SDP)
	}
	return answer.SDP, nil
}

// HandleAnswer sets a remote answer (when the SFU offered first — rare for viewers).
func (r *Room) HandleAnswer(peerID uuid.UUID, sdp string) error {
	r.mu.Lock()
	pc := r.pcFor(peerID)
	r.mu.Unlock()
	if pc == nil {
		return fmt.Errorf("sfu: unknown peer %s", peerID)
	}
	return pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: sdp})
}

// HandleICE adds a trickle ICE candidate.
func (r *Room) HandleICE(peerID uuid.UUID, cand webrtc.ICECandidateInit) error {
	r.mu.Lock()
	pc := r.pcFor(peerID)
	r.mu.Unlock()
	if pc == nil {
		return fmt.Errorf("sfu: unknown peer %s", peerID)
	}
	return pc.AddICECandidate(cand)
}

// CreateViewerOffer creates an SFU→viewer offer after the presenter track is available.
func (r *Room) CreateViewerOffer(peerID uuid.UUID) (string, error) {
	r.mu.Lock()
	pc := r.pcFor(peerID)
	track := r.presenterTrack
	r.mu.Unlock()
	if pc == nil {
		return "", fmt.Errorf("sfu: unknown peer %s", peerID)
	}
	if track != nil {
		r.mu.Lock()
		if v := r.viewers[peerID]; v != nil && v.sender == nil {
			r.attachTrackToViewer(v, track)
		}
		r.mu.Unlock()
	}
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return "", err
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		return "", err
	}
	if r.signal != nil {
		r.signal.SendOffer(peerID, offer.SDP)
	}
	return offer.SDP, nil
}

// Detach removes a peer connection.
func (r *Room) Detach(peerID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.presenterID == peerID && r.presenterPC != nil {
		_ = r.presenterPC.Close()
		r.presenterPC = nil
		r.presenterTrack = nil
		r.presenterID = uuid.Nil
		return
	}
	if v, ok := r.viewers[peerID]; ok {
		if v.pc != nil {
			_ = v.pc.Close()
		}
		delete(r.viewers, peerID)
	}
}

// ClearPresenter stops forwarding without closing viewers (hand-off pause).
func (r *Room) ClearPresenter() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.presenterPC != nil {
		_ = r.presenterPC.Close()
		r.presenterPC = nil
	}
	r.presenterTrack = nil
	r.presenterID = uuid.Nil
	for _, v := range r.viewers {
		if v.sender != nil && v.pc != nil {
			_ = v.pc.RemoveTrack(v.sender)
			v.sender = nil
		}
	}
}

// Close tears down all peer connections.
func (r *Room) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.presenterPC != nil {
		_ = r.presenterPC.Close()
		r.presenterPC = nil
	}
	for id, v := range r.viewers {
		if v.pc != nil {
			_ = v.pc.Close()
		}
		delete(r.viewers, id)
	}
	r.presenterTrack = nil
}

// ViewerCount returns connected viewer peer count (for tests / metrics).
func (r *Room) ViewerCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.viewers)
}

// HasPresenter reports whether an upstream is attached.
func (r *Room) HasPresenter() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.presenterPC != nil
}

func (r *Room) pcFor(peerID uuid.UUID) *webrtc.PeerConnection {
	if r.presenterID == peerID {
		return r.presenterPC
	}
	if v, ok := r.viewers[peerID]; ok {
		return v.pc
	}
	return nil
}

// Registry holds SFU rooms keyed by session ID.
type Registry struct {
	mu    sync.Mutex
	rooms map[uuid.UUID]*Room
}

// NewRegistry creates an empty room registry.
func NewRegistry() *Registry {
	return &Registry{rooms: make(map[uuid.UUID]*Room)}
}

// GetOrCreate returns the room for sessionID, creating it if needed.
func (reg *Registry) GetOrCreate(sessionID uuid.UUID, ice []webrtc.ICEServer, signal SignalOut) *Room {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	if r, ok := reg.rooms[sessionID]; ok {
		return r
	}
	r := NewRoom(sessionID, ice, signal)
	reg.rooms[sessionID] = r
	return r
}

// Get returns an existing room or nil.
func (reg *Registry) Get(sessionID uuid.UUID) *Room {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	return reg.rooms[sessionID]
}

// Delete closes and removes a room.
func (reg *Registry) Delete(sessionID uuid.UUID) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	if r, ok := reg.rooms[sessionID]; ok {
		r.Close()
		delete(reg.rooms, sessionID)
	}
}

// Len returns active room count.
func (reg *Registry) Len() int {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	return len(reg.rooms)
}
