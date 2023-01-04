package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/simplylib/defermodafterreturn/linter"
)

func run() (err error) {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	verbose := flag.Bool("v", false, "be verbose about operations")
	workers := flag.Int("t", runtime.NumCPU(), "how many files to work on at once")

	flag.CommandLine.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(),
			os.Args[0]+" detects uses of a defer that attempts to modify return values in a function without named returns\n",
			"\nUsage: "+os.Args[0]+" [flags] <dir>\n\n",
			"Ex: "+os.Args[0]+" -v -t 8 . // recursively find all go files in current directory and scan for defers\n",
			"\nFlags:\n",
		)
		flag.CommandLine.PrintDefaults()
	}

	flag.Parse()

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	go func() {
		osSignal := make(chan os.Signal, 1)
		signal.Notify(osSignal, syscall.SIGTERM, os.Interrupt)

		s := <-osSignal
		log.Printf("Cancelling operations due to (%v)\n", s.String())
		cancelFunc()
		log.Println("operations cancelled")
	}()

	if *verbose {
		log.SetFlags(log.Ltime | log.Lshortfile)
	}

	if flag.NArg() != 1 {
		return errors.New("expected 1 argument <dir>")
	}

	err = linter.LintDirectory(ctx, filepath.Clean(flag.Arg(0)), *workers)
	if err != nil {
		return fmt.Errorf("could not lint directory (%v) error (%w)", flag.Arg(0), err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
