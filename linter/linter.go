package linter

import (
	"bytes"
	"context"
	"errors"
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
)

// externalBlockAssignments finds the block assignments made outside of the function body.
func externalBlockAssignments(block *ast.BlockStmt) []*ast.Ident {
	var (
		assigns []*ast.Ident
		decls   []*ast.Ident
	)
	ast.Inspect(block, func(n ast.Node) bool {
		switch x := n.(type) {
		// todo: recursive listing for blocks
		case *ast.FuncLit:
		case *ast.DeferStmt:
		case *ast.AssignStmt:
		lhs:
			for i := range x.Lhs {
				ident, ok := x.Lhs[i].(*ast.Ident)
				if !ok {
					continue
				}

				if x.Tok == token.DEFINE {
					decls = append(decls, ident)
					continue
				}

				for j := range decls {
					if decls[j].Name != ident.Name {
						continue
					}
					continue lhs
				}

				assigns = append(assigns, ident)
			}
		case *ast.DeclStmt:
			genDecl, ok := x.Decl.(*ast.GenDecl)
			if !ok {
				return true
			}

			for _, spec := range genDecl.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}

				decls = append(decls, value.Names...)
			}
		}

		return true
	})

	return assigns
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

type assignWithoutNamedReturn struct {
	name     string
	position int
}

func checkFileAsBytes(filename string, bs []byte) []assignWithoutNamedReturn {
	return nil
}

func lintBytes(filename string, bs []byte) (err error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, bs, 0)
	if err != nil {
		return fmt.Errorf("could not parse file (%v) due to error (%w)", filename, err)
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
			// check if are we in a defer
			if lastDefer == nil || x.Pos() < lastDefer.Pos() || x.Pos() > lastDefer.End() {
				return true
			}

			globalAssigns := externalBlockAssignments(outsideFunction.Body)

		deferAssignsLoop:
			for _, assign := range externalBlockAssignments(x.Body) {
				if functionTypeHasNamedVar(outsideFunction, assign.Name) {
					continue
				}

				for _, globalAssign := range globalAssigns {
					if assign.Name == globalAssign.Name {
						continue deferAssignsLoop
					}
				}

				buf := &bytes.Buffer{}
				err = printer.Fprint(buf, fset, lastDefer)
				if err != nil {
					err = fmt.Errorf("could not print ast (%w)", err)
					return false
				}

				log.Printf(
					"%v:%v:%v function literal in defer assigns to (%v) a non-named return in parent function\n%s\n",
					filename,
					fset.Position(x.Pos()).Line,
					fset.Position(x.Pos()).Column,
					assign.Name,
					buf.Bytes(),
				)
			}
		}

		return true
	})
	if err != nil {
		return err
	}

	return nil
}

func LintFile(path string) (err error) {
	var file *os.File
	file, err = os.Open(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("could not ReadFile (%v) error (%w)", path, err)
	}

	defer func() {
		if err2 := file.Close(); err2 != nil {
			err = errors.Join(err, fmt.Errorf("could not close file (%v) due to error (%w)", path, err))
		}
	}()

	var bytes []byte
	bytes, err = io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("could not read all bytes from (%v) error (%w)", filepath.Clean(path), err)
	}

	return lintBytes(path, bytes)
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
