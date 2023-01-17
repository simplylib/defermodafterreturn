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
