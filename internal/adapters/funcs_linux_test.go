//go:build linux

package adapters

import (
	"errors"
	"testing"

	"github.com/gostafa/inputstats/internal/adapters/linux"
)

func TestNew(t *testing.T) {
	_, err := New()
	if err != nil {
		t.Log(err)
	}
}

func TestNewOK(t *testing.T) {
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
	port, err := wrapLinux(nil, errors.New("denied"))
	if err == nil || port != nil {
		t.Fatal("expected wrapped error")
	}
}

func TestWrapOK(t *testing.T) {
	in := &linux.Adapter{}

	port, err := wrapLinux(in, nil)
	if err != nil || port != in {
		t.Fatalf("port=%v err=%v", port, err)
	}
}
