package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mr-joshcrane/rivulet"
)

func main() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		fmt.Println("Usage: rivulet <publisherName>")
		os.Exit(1)
	}
	ctx := context.Background()
	result, err := rivulet.Read(ctx, os.Args[1], flag.)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, message := range result {
		fmt.Println(message)
	}
}
