package linter

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/simplylib/errgroup"
	"github.com/simplylib/multierror"
)

func LintFile(path string) (err error) {
	var file *os.File
	file, err = os.Open(path)
	if err != nil {
		return fmt.Errorf("could not ReadFile (%v) error (%w)", path, err)
	}
	defer func() {
		if err2 := file.Close(); err2 != nil {
			err = multierror.Append(err, fmt.Errorf("could not close file (%v) due to error (%w)", path, err))
		}
	}()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Base(path), file, 0)
	if err != nil {
		return fmt.Errorf("could not parse file (%v) due to error (%w)", path, err)
	}

	/*
		naive implementation described below:
		1. find a function declaration or function literal (anonymous functions)
		2. determine if that function signature contains a 
	*/
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			fmt.Printf("file: (%v) found function (%v) line (%v)\n", file.Name(), x.Name.Name, fset.Position(x.Pos()))
		case *ast.FuncLit:
			fmt.Printf("file: (%v) found anonymous function on line (%v)\n", file.Name(), fset.Position(x.Pos()))
		}

		return true
	})

	return nil
}

func LintDirectory(ctx context.Context, path string, workers int) (err error) {
	errg := errgroup.Group{}
	errg.SetLimit(workers)

	err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error reading file for walkdir path (%v) error (%w)", path, err)
		}

		select {
		case <-ctx.Done():
			return io.EOF
		default:
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		errg.Go(func() error {
			err := LintFile(path)
			if err != nil {
				return fmt.Errorf("could not lintfile (%v) error (%w)", path, err)
			}

			return nil
		})

		return nil
	})
	if err != nil && err != io.EOF {
		return fmt.Errorf("could not walkdir (%v) error (%w)", path, err)
	}

	err = errg.Wait()
	if err != nil {
		return fmt.Errorf("could not lint file(s) (%w)", err)
	}

	return nil
}
