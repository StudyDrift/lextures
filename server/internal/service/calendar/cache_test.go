package calendar_test

import (
	"testing"
	"time"

	calendarsvc "github.com/lextures/lextures/server/internal/service/calendar"
)

func TestFeedCache_InvalidateUser(t *testing.T) {
	c := calendarsvc.NewFeedCache()
	now := time.Now()
	c.Set("user:abc", []byte("personal"), now)
	c.Set("course:def:abc", []byte("course"), now)
	c.Set("user:xyz", []byte("other"), now)

	c.InvalidateUser("abc")

	if _, ok := c.Get("user:abc", now); ok {
		t.Fatal("expected user:abc invalidated")
	}
	if _, ok := c.Get("course:def:abc", now); ok {
		t.Fatal("expected course:def:abc invalidated")
	}
	if _, ok := c.Get("user:xyz", now); !ok {
		t.Fatal("expected user:xyz to remain cached")
	}
}

func TestFeedCache_InvalidateCourse(t *testing.T) {
	c := calendarsvc.NewFeedCache()
	now := time.Now()
	c.Set("course:def:abc", []byte("course-a"), now)
	c.Set("course:def:xyz", []byte("course-b"), now)
	c.Set("user:abc", []byte("personal"), now)

	c.InvalidateCourse("def")

	if _, ok := c.Get("course:def:abc", now); ok {
		t.Fatal("expected course:def:abc invalidated")
	}
	if _, ok := c.Get("course:def:xyz", now); ok {
		t.Fatal("expected course:def:xyz invalidated")
	}
	if _, ok := c.Get("user:abc", now); !ok {
		t.Fatal("expected user:abc to remain cached")
	}
}