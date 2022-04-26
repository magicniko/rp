package rp

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/bogdanovich/dns_resolver"
)

type proxyConnection struct {
	reverseProxy *httputil.ReverseProxy
	target       *url.URL
	weight       int
}

func (p *proxyConnection) String() string {
	return p.target.String()
}

func (p *proxyConnection) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.URL.Host = p.target.Host
	r.URL.Scheme = p.target.Scheme
	r.Host = p.target.Host
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
  // https://pkg.go.dev/net/http#RoundTripper
	p.reverseProxy.Transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 10 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	p.reverseProxy.ServeHTTP(w, r)
}

func newProxyConnection(target *url.URL, weight int) *proxyConnection {
	return &proxyConnection{
		reverseProxy: httputil.NewSingleHostReverseProxy(target),
		target:       target,
		weight:       weight,
	}
}

type ReverseProxy struct {
	rr  *roundRobin
	log bool
}

func New() *ReverseProxy {
	return &ReverseProxy{
		rr: newRoundRobin(),
	}
}

func (rp *ReverseProxy) ChangeDNS(domainServers ...string) {
	if len(domainServers) == 0 {
		return
	}

	if rp.log {
		log.Println("using domain servers:", domainServers)
	}

	dialer := &net.Dialer{
		DualStack: true,
	}
	http.DefaultTransport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		remote := strings.Split(addr, ":")
		if net.ParseIP(remote[0]).String() == remote[0] {
			resolver := dns_resolver.New(domainServers)
			resolver.RetryTimes = 5
			ip, err := resolver.LookupHost(remote[0])
			if err != nil {
				return nil, err
			}
			remote[0] = ip[0].String()
		}
		addr = strings.Join(remote, ":")
		return dialer.DialContext(ctx, network, addr)
	}
}

func (rp *ReverseProxy) Log(mode bool) {
	rp.log = mode
}

func (rp *ReverseProxy) Add(target *url.URL, weight int) {
	rp.rr.Add(newProxyConnection(target, weight))
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	n, err := w.ResponseWriter.Write(b)
	return n, err
}

func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proxy := rp.rr.Get()
	if rp.log {
		start := time.Now()
		path := r.URL.Path
		raw := r.URL.RawQuery

		sw := &statusWriter{ResponseWriter: w}
		proxy.ServeHTTP(sw, r)

		end := time.Now()
		latency := end.Sub(start)

		clientIP := clientIP(r)
		method := r.Method
		if raw != "" {
			path = path + "?" + raw
		}

		log.Printf("| %3d | %13v | %15s | %-7s %s -> %s",
			sw.status,
			latency,
			clientIP,
			method,
			path,
			proxy.String(),
		)
		return
	}
	proxy.ServeHTTP(w, r)
}

func (rp *ReverseProxy) ListenAndServe(addr string) error {
	if len(rp.rr.conns) == 0 {
		return fmt.Errorf("not enough remote addresses")
	}
	return http.ListenAndServe(addr, rp)
}

func clientIP(r *http.Request) string {
	clientIP := r.Header.Get("X-Forwarded-For")
	clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
	if clientIP == "" {
		clientIP = strings.TrimSpace(r.Header.Get("X-Real-Ip"))
	}
	if clientIP != "" {
		return clientIP
	}

	if ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil {
		return ip
	}

	return ""
}
