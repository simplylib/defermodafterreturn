package testdata

import (
	"fmt"
	"io"
)

// BadFunction copies from r to w then closes w.
func BadFunction(w io.WriteCloser, r io.Reader) (int64, error) {
	var err error
	defer func() {
		err2 := w.Close()
		if err2 != nil && err != nil {
			err = err2
		}
	}()

	n, err := io.Copy(w, r)
	if err != nil {
		return n, fmt.Errorf("could not copy from Reader to WriteCloser (%w)", err)
	}

	return n, nil
}
