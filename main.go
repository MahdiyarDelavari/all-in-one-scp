package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"allinonescp/internal/app"
)

func main() {
	server := app.New()
	httpServer := &http.Server{
		Handler: server.Routes(),
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Errorf("start local web server: %w", err))
	}

	go func() {
		_ = httpServer.Serve(listener)
	}()

	url := "http://" + listener.Addr().String()
	server.Log("App started at " + url)

	if os.Getenv("ALL_IN_ONE_SCP_NO_BROWSER") == "1" {
		fmt.Println(url)
	} else if err := app.OpenBrowser(url); err != nil {
		server.Log("Could not open browser automatically: " + err.Error())
	}

	<-server.Quit()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
}
