package main

import (
	"io"
	"os"

	"github.com/futuretea/mcm/internal/cli"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, in io.Reader, out, errOut io.Writer) int {
	return cli.Run(args, os.Getenv("HOME"), in, out, errOut)
}
