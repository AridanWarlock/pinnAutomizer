// Command quality runs project checks with bounded diagnostics and full logs.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	r := runner{root: root, packages: packages, output: os.Stdout, ctx: ctx}
	code, err := r.run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "quality:", err)
		code = 1
	}
	os.Exit(code)
}
