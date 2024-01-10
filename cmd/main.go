package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

type ServerPool struct {
	backends []*Backend
	current  uint64
}

var serverPool ServerPool

func main() {

	//router.LoadRouter() used for webApis
	StartLoadBalancerServer()

}

func StartLoadBalancerServer() {

	serverList := []string{"http://localhost:8001", "http://localhost:8002"}

	for _, server := range serverList {
		fmt.Println("Url registered: ", server)
		urlServer, err := url.Parse(server)
		if err != nil {
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(urlServer)

		serverPool.backends = append(serverPool.backends, &Backend{
			URL:          urlServer,
			Alive:        true,
			ReverseProxy: proxy,
		})

	}

	//init server
	srv := &http.Server{
		Addr:    "localhost:8000",
		Handler: http.HandlerFunc(LoadBalance),
	}

	fmt.Println("Load Balance starting at : ", srv.Addr)
	err := srv.ListenAndServe()
	if err != nil {
		fmt.Println("Error on api:", err)

	}
}

func LoadBalance(w http.ResponseWriter, r *http.Request) {
	nextPeer := serverPool.GetNextPeer()
	if nextPeer != nil {
		nextPeer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
}

// find the next backend alive
func (s *ServerPool) GetNextPeer() *Backend {

	next := s.NextIndex()
	l := len(s.backends) + next

	for i := next; i < l; i++ {
		index := i % len(s.backends)

		if s.backends[index].IsAlive() {
			if i != index {
				atomic.StoreUint64(&s.current, uint64(index))
			}

		}
		return s.backends[index]
	}
	return nil
}

func (b *Backend) IsAlive() (alive bool) {
	b.mux.Lock()
	alive = b.Alive
	b.mux.Unlock()
	return
}
