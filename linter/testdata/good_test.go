package testdata

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestGoodFunction(t *testing.T) {
	n, err := GoodFunction(&FakeWriteCloser{}, strings.NewReader("test"))
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected err = io.EOF instead err = (%v)", err)
	}

	if n != 0 {
		t.Fatalf("expected n = 0 instead n = (%v)", n)
	}

	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("expected errors.Is(io.ErrClosedPipe) = true instead false")
	}
}
