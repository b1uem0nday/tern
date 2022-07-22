package main

import (
	"context"
	"github.com/b1uem0nday/tern/internal/service"
	"log"
	"os"
)

func main() {
	var err error
	if len(os.Args) < 2 {
		err = service.RunWithConfig(context.Background())
	} else {
		err = service.RunWithFlags(context.Background())
	}
	if err != nil {
		log.Fatal(err)
	}
}
