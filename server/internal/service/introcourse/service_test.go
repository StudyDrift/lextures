package introcourse

import (
	"testing"

	"github.com/lextures/lextures/server/internal/config"
	icrepo "github.com/lextures/lextures/server/internal/repos/introcourse"
)

func TestEnabled_ReflectsMergedConfig(t *testing.T) {
	if !Enabled(config.Config{IntroCourseEnabled: true}) {
		t.Fatal("expected true when merged config enables intro course")
	}
	if Enabled(config.Config{IntroCourseEnabled: false}) {
		t.Fatal("expected false when explicitly disabled")
	}
}

func TestConstants_SingleSource(t *testing.T) {
	if ShortCode != icrepo.ShortCode {
		t.Fatalf("ShortCode drift: service=%q repo=%q", ShortCode, icrepo.ShortCode)
	}
	if SystemUserID != icrepo.SystemUserID {
		t.Fatal("SystemUserID drift between service and repo")
	}
}