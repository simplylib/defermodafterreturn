package linter

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/simplylib/errgroup"
	"github.com/simplylib/multierror"
)

// externalBlockAssignments finds the block assignments made outside of the function body.
func externalBlockAssignments(block *ast.BlockStmt) ([]*ast.Ident, error) {
	var (
		assigns []*ast.Ident
		decls   []*ast.Ident
		err     error
	)
	ast.Inspect(block, func(n ast.Node) bool {
		switch x := n.(type) {
		// todo: recursive listing for blocks
		case *ast.FuncLit:
		case *ast.DeferStmt:
		case *ast.AssignStmt:
			for i := range x.Lhs {
				if x.Tok == token.DEFINE {
					continue
				}

				ident, ok := x.Lhs[i].(*ast.Ident)
				if !ok {
					err = fmt.Errorf("expected an *ast.Ident on left of := / = instead got (%T)", x.Lhs[i])
					return false
				}

				for j := range decls {
					if decls[j].Name != ident.Name {
						continue
					}
				}

				assigns = append(assigns, ident)
			}
		case *ast.DeclStmt:
			genDecl, ok := x.Decl.(*ast.GenDecl)
			if !ok {
				err = fmt.Errorf("expected an *ast.GenDecl in *ast.DeclStmt instead got (%T)", x.Decl)
				return false
			}

			for _, spec := range genDecl.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					err = fmt.Errorf("expected *ast.ValueSpec instead got (%T)", spec)
					return false
				}

				decls = append(decls, value.Names...)
			}
		}

		return true
	})
	if err != nil {
		return nil, err
	}

	return assigns, nil
}

func functionTypeHasNamedVar(f *ast.FuncDecl, name string) bool {
	if f.Type == nil {
		return false
	}

	if f.Type.Results == nil {
		return false
	}

	if f.Type.Results.List == nil {
		return false
	}

	returns := f.Type.Results.List
	for i := range returns {
		if len(returns[i].Names) == 0 {
			return false
		}

		for _, namedVar := range returns[i].Names {
			if namedVar.Name != name {
				continue
			}
			return true
		}
	}
	return true
}

func LintFile(path string) (err error) {
	var file *os.File
	file, err = os.Open(filepath.Clean(path))
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
		1. find a function literal (anonymous function) called from defer
		2. determine if that function's body modifies a variable not declared within it
		3. check if that variable is returned to the outside function

		todo: guess if a variable generally returned is modified but not referenced
		      directly by a return (like err)
		      ex: func() error {
				  	f, err := os.Open("")
					if err != nil {
						return err
					}
					defer func() {
						if err2 := f.Close(); err2 != nil && err == nil {
							err = err2
						}
					}()

					// generally this can be assumed to be err
					return nil
				}
	*/
	var (
		outsideFunction *ast.FuncDecl
		lastDefer       *ast.DeferStmt
	)
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			outsideFunction = x
		case *ast.DeferStmt:
			lastDefer = x
		case *ast.FuncLit:
			if lastDefer == nil || x.Pos() < lastDefer.Pos() || x.Pos() > lastDefer.End() {
				return true
			}

			var assigns []*ast.Ident
			assigns, err = externalBlockAssignments(x.Body)
			if err != nil {
				err = fmt.Errorf("could not get externalBlockAssignments (%w)", err)
				return false
			}

			for i := range assigns {
				if functionTypeHasNamedVar(outsideFunction, assigns[i].Name) {
					continue
				}

				buf := &bytes.Buffer{}
				err = printer.Fprint(buf, fset, lastDefer)
				if err != nil {
					err = fmt.Errorf("could not print ast (%w)", err)
					return false
				}

				log.Printf(
					"%v:%v:%v function literal in defer assigns to (%v) a non-named return in parent function\n%s\n",
					file.Name(),
					fset.Position(x.Pos()).Line,
					fset.Position(x.Pos()).Column,
					assigns[i].Name,
					buf.Bytes(),
				)
			}
		}

		return true
	})
	if err != nil {
		return err
	}

	_ = outsideFunction

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
