// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package mock

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var httpServer http.Server
var mux *http.ServeMux = http.NewServeMux()

func StartHttpServer(port int) error {
	ctx := context.Background()

	httpServer = http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)

	var serverErr error = nil

	go func() {
		log.Print("Mock server started")

		if err := httpServer.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				serverErr = err
				stop <- syscall.SIGINT // Simulate Ctrl+C
			}
		}
	}()

	<-stop

	if serverErr != nil {
		return serverErr
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*5))
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}
