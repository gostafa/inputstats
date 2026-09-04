//go:build darwin

package adapters

import (
	"errors"
	"testing"

	"github.com/gostafa/inputstats/internal/adapters/darwin"
)

func TestNew(t *testing.T) {
	_, err := New()
	if err != nil {
		t.Log(err)
	}
}

func TestNewAllow(t *testing.T) {
	t.Setenv("INPUTSTATS_TEST_GRANT", "allow")

	port, err := New()
	if err != nil || port == nil {
		t.Fatalf("port=%v err=%v", port, err)
	}
}

func TestNewDeny(t *testing.T) {
	t.Setenv("INPUTSTATS_TEST_GRANT", "deny")

	port, err := New()
	if err == nil || port != nil {
		t.Fatal("expected error")
	}
}

func TestWrapError(t *testing.T) {
	port, err := wrapDarwin(nil, errors.New("denied"))
	if err == nil || port != nil {
		t.Fatal("expected wrapped error")
	}
}

func TestWrapOK(t *testing.T) {
	in := &darwin.Adapter{}

	port, err := wrapDarwin(in, nil)
	if err != nil || port != in {
		t.Fatalf("port=%v err=%v", port, err)
	}
}
