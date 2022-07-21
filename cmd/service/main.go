package main

import (
	"context"
	"github.com/b1uem0nday/tern/internal/grpc"
	"log"
)

//проверить хранимые процедуры + функции на корректность создания
func main() {
	ctx := context.Background()
	server, err := grpc.NewServer(ctx, "/home/irina/Documents/tern/scripts/migrations/", 5000)
	if err != nil {
		log.Fatalf("grpc connection: %v", err)
	}
	server.Run()
	<-ctx.Done()
}
