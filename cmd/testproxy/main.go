package main

import (
	"context"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/koenbollen/logging"
	"github.com/poki/netlib/internal/util"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := logging.New(ctx, "netlib", "testproxy")
	defer logger.Sync() // nolint:errcheck
	logger.Info("init")
	defer logger.Info("fin")
	ctx = logging.WithLogger(ctx, logger)

	connections := make(map[string]net.Conn)
	interrupts := make(map[string]bool)
	http.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
		id := r.FormValue("id")
		port := r.FormValue("port")
		laddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
		if err != nil {
			panic(err)
		}
		raddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:"+port)
		if err != nil {
			panic(err)
		}
		conn, err := net.ListenUDP("udp", laddr)
		if err != nil {
			panic(err)
		}
		connections[id] = conn
		_, lport, _ := net.SplitHostPort(conn.LocalAddr().String())
		w.Write([]byte(lport)) //nolint:errcheck
		go func() {
			defer conn.Close()
			buffer := make([]byte, 65_535)
			var remote *net.UDPAddr
			for {
				n, addr, err := conn.ReadFromUDP(buffer)
				if err != nil {
					break
				}
				if addr.Port != raddr.Port {
					remote = addr
				}
				recipient := remote
				if addr.Port == remote.Port {
					recipient = raddr
				}
				if yes := interrupts[id]; yes {
					continue
				}
				_, err = conn.WriteToUDP(buffer[:n], recipient)
				if err != nil {
					break
				}
			}
		}()
	})
	http.HandleFunc("/interrupt", func(w http.ResponseWriter, r *http.Request) {
		id := r.FormValue("id")
		interrupts[id] = true
	})
	http.HandleFunc("/uninterrupt", func(w http.ResponseWriter, r *http.Request) {
		id := r.FormValue("id")
		interrupts[id] = false
	})
	http.HandleFunc("/close", func(w http.ResponseWriter, r *http.Request) {
		id := r.FormValue("id")
		conn, ok := connections[id]
		if ok {
			conn.Close()
			delete(connections, id)
		}
	})

	addr := util.Getenv("ADDR", ":8080")
	server := &http.Server{
		Addr:    addr,
		Handler: http.DefaultServeMux,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to listen and server", zap.Error(err))
		}
	}()
	logger.Info("listening", zap.String("addr", addr))

	<-ctx.Done()

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("failed to shutdown server", zap.Error(err))
	}
}
