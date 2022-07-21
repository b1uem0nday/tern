package main

import (
	"context"
	"flag"
	"github.com/b1uem0nday/tern/internal/grpc"
	"log"
)

const BuildNumber = "1.0.0"

//проверить хранимые процедуры + функции на корректность создания
func main() {
	port := flag.String("port", "5400", "grpc port ")
	filePath := flag.String("path", "./migration/", "migration scripts path")
	flag.Parse()
	flag.Usage = func() {

	}
	flag.Parse()
	ctx := context.Background()

	server, err := grpc.NewServer(ctx, *filePath, *port)
	if err != nil {
		log.Fatalf("grpc connection: %v", err)
	}
	server.Run()
	<-ctx.Done()

}
