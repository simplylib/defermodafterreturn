package testdata

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type FakeWriteCloser struct{}

func (f *FakeWriteCloser) Write(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (f *FakeWriteCloser) Close() error {
	return io.ErrClosedPipe
}

func TestBadFunction(t *testing.T) {
	n, err := BadFunction(&FakeWriteCloser{}, strings.NewReader("test"))
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected err = io.EOF instead err = (%v)", err)
	}

	if n != 0 {
		t.Fatalf("expected n = 0 instead n = (%v)", n)
	}

	if errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("expected errors.Is(io.ErrClosedPipe) = false instead true")
	}
}
