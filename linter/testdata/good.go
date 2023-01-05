package testdata

import (
	"fmt"
	"io"

	"github.com/simplylib/multierror"
)

// GoodFunction copies data from r to w then close w.
func GoodFunction(w io.WriteCloser, r io.Reader) (n int64, err error) {
	defer func() {
		err2 := w.Close()
		if err2 != nil && err != nil {
			err = multierror.Append(err, fmt.Errorf("could not close WriteCloser (%w)", err2))
			// in go1.20+ this can be replaced with fmt.Errorf with two %w verbs
		}
	}()

	n, err = io.Copy(w, r)
	if err != nil {
		return n, fmt.Errorf("could not copy from Reader to WriteCloser (%w)", err)
	}

	return 0, nil
}
