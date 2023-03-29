package service

import (
	"context"
	"crypto/tls"
	"github.com/gorilla/mux"
	"github.com/soheilhy/cmux"
	"log"
	"net"
	"net/http"
	"sync"
)

type SimpleWeb struct {
	rawListener net.Listener
	tcpMux      cmux.CMux
	httpMux     *mux.Router

	pm *PemManager

	httpListener net.Listener
	tlsListener  net.Listener
}

func (sw *SimpleWeb) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sw.httpMux.ServeHTTP(w, r)
}

func (sw *SimpleWeb) Serve(ctx context.Context) {
	sw.regHandlers()

	var wg sync.WaitGroup

	// http svc
	wg.Add(1)
	go func(ctx context.Context, wg *sync.WaitGroup) {
		log.Println("svc http")
		defer func() {
			log.Println("svc http finished")
			wg.Done()
		}()

		server := &http.Server{
			Handler: sw,
		}

		svcCtx, cancelFunc := context.WithCancel(ctx)

		go func() {
			defer cancelFunc()
			if err := server.Serve(sw.httpListener); err != nil {
				log.Println("http server svc err:", err)
			}
		}()

		select {
		case <-ctx.Done():
			if err := server.Shutdown(context.Background()); err != nil {
				log.Println("http server shutdown err:", err)
			}
			return
		case <-svcCtx.Done():
			return
		}
	}(ctx, &wg)

	// https svc
	wg.Add(1)
	go func(ctx context.Context, wg *sync.WaitGroup) {
		log.Println("svc https")
		defer func() {
			log.Println("svc https finished")
			wg.Done()
		}()

		cfg := &tls.Config{
			GetCertificate: sw.pm.GetCertificate,
		}
		tlsl := tls.NewListener(sw.tlsListener, cfg)

		server := &http.Server{
			Handler: sw,
		}

		svcCtx, cancelFunc := context.WithCancel(ctx)

		go func() {
			defer cancelFunc()
			if err := server.Serve(tlsl); err != nil {
				log.Println("https server svc err:", err)
			}
		}()

		select {
		case <-ctx.Done():
			if err := server.Shutdown(context.Background()); err != nil {
				log.Println("https server shutdown err:", err)
			}
			return
		case <-svcCtx.Done():
			return
		}
	}(ctx, &wg)

	// mux
	wg.Add(1)
	go func(ctx context.Context, wg *sync.WaitGroup) {
		defer wg.Done()
		if err := sw.tcpMux.Serve(); err != nil {
			log.Println("mux serve err:", err)
		}
	}(ctx, &wg)
	wg.Wait()
}

func New(addr string) (*SimpleWeb, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	tcpMux := cmux.New(l)
	httpMux := mux.NewRouter()

	hl := tcpMux.Match(cmux.HTTP1Fast())
	tlsl := tcpMux.Match(cmux.Any())

	return &SimpleWeb{
		rawListener:  l,
		tcpMux:       tcpMux,
		httpMux:      httpMux,
		pm:           NewPemManager(),
		httpListener: hl,
		tlsListener:  tlsl,
	}, nil
}
