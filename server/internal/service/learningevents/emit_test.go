package learningevents

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/service/xapi"
)

func TestEmitter_disabled_noPanic(t *testing.T) {
	e := Emitter{Cfg: config.Config{XAPIEmissionEnabled: false}}
	e.LoggedIn(context.Background(), uuid.Nil, "a@b.com", "A")
}

func TestPayload_roundTrip(t *testing.T) {
	stmt := xapi.BuildStatement(xapi.BuildInput{
		ActorEmail: "u@test.invalid",
		VerbID:     xapi.VerbExperienced,
		ObjectID:   "https://example.com/obj",
		Anonymize:  true,
	})
	raw, err := xapi.MarshalStatement(stmt)
	if err != nil {
		t.Fatal(err)
	}
	var p Payload
	if err := json.Unmarshal([]byte(`{"xapi":`+string(raw)+`,"caliper":{}}`), &p); err != nil {
		t.Fatal(err)
	}
	if len(p.XAPI) == 0 {
		t.Fatal("expected xapi payload")
	}
}
