# defermodafterreturn
A Go (golang) tool for detecting defers that attempt to modify a non-named return value

## Installing
```
go install github.com/simplylib/defermodafterreturn@latest
```

## Usage
```
defermodafterreturn detects uses of a defer that attempts to modify return values in a function without named returns

Usage: defermodafterreturn [flags] <dir/file>

Ex: defermodafterreturn -v -t 8 . // recursively find all go files in current directory and scan for defers

Flags:
  -t int
        how many files to work on at once (default is number of cpu threads)
  -v    be verbose about operations
```

## Example
Running ```defermodafterreturn linter/testdata/bad.go``` inside of this repo
on this [file](https://github.com/simplylib/defermodafterreturn/blob/f42490cf61b1bd3682baedf39f57e01c6c492b7a/linter/testdata/bad.go)
```   
linter/testdata/bad.go:11:8 function literal in defer assigns to non-named return in parent function
defer func() {
        err2 := w.Close()
        if err2 != nil && err != nil {
                err = err2
        }
}()
```
