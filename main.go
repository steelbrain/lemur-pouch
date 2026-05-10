package main

import (
	"embed"
	"errors"
	"flag"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/steelbrain/lemur-pouch/internal/relay"
)

//go:embed all:web/dist
var distFS embed.FS

// listenAddr is the bind address. Defaults to ":8080" — Go's shorthand
// for "all interfaces, port 8080" (binds 0.0.0.0 and, on dual-stack
// systems, ::). To restrict to a single interface, pass an explicit
// host:port (e.g. "127.0.0.1:8080" for localhost-only, or
// "192.168.1.5:8080" to bind one specific NIC).
var listenAddr = flag.String(
	"listen",
	":8080",
	`address to listen on (host:port).
Examples:
  :8080            all interfaces, port 8080 (default)
  0.0.0.0:8080     all IPv4 interfaces only
  [::]:8080        all IPv6 interfaces (and IPv4-mapped on dual-stack)
  127.0.0.1:8080   localhost only
  192.168.1.5:80   bind to one specific interface IP`,
)

func main() {
	flag.Parse()

	hub := relay.NewHub()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", relay.HandleWebSocket(hub))

	staticFS, err := fs.Sub(distFS, "web/dist")
	if err != nil {
		log.Fatalf("derive web/dist sub-FS: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	srv := &http.Server{
		Addr:              *listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	logReachableURLs(*listenAddr)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}

// logReachableURLs prints the URLs the relay is reachable at, given
// the configured listen address.
//
// For wildcard binds (the default ":8080", or explicit "0.0.0.0:..."
// or "[::]:..."), it enumerates every non-loopback, non-link-local
// interface IP and prints a URL per address — so a user starting the
// relay on a laptop sees the LAN IPs they should give to other
// participants without having to dig through `ifconfig` / `ip addr`.
//
// For specific binds (e.g. "127.0.0.1:8080" or "192.168.1.5:8080"),
// it prints only that one URL, since other addresses won't reach the
// listener anyway.
func logReachableURLs(listen string) {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		log.Printf("could not parse --listen %q: %v", listen, err)
		return
	}

	log.Printf("relay listening on %s — reachable at:", listen)

	if !isWildcard(host) {
		// Specific bind — only that address is reachable.
		log.Printf("  http://%s/", net.JoinHostPort(host, port))
		return
	}

	// Wildcard bind. Always print localhost (the local user's URL),
	// then enumerate every interface IP for LAN/peer access.
	log.Printf("  http://localhost:%s/   (this machine)", port)

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("could not enumerate interface addresses: %v", err)
		return
	}

	var v4, v6 []net.IP
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipnet.IP
		// Skip loopback (already covered by the localhost line above)
		// and link-local addresses. Link-local IPv6 (fe80::/10)
		// requires a zone-id suffix in URLs ("[fe80::1%eth0]"), which
		// is awkward and rarely useful for cross-machine LAN sharing.
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			continue
		}
		if ip.To4() != nil {
			v4 = append(v4, ip)
		} else {
			v6 = append(v6, ip)
		}
	}

	// Stable order: IPv4 first (more familiar to LAN users), then
	// IPv6. Sort within each family so the output is deterministic
	// across runs even if the OS reorders interfaces.
	sort.Slice(v4, func(i, j int) bool { return v4[i].String() < v4[j].String() })
	sort.Slice(v6, func(i, j int) bool { return v6[i].String() < v6[j].String() })

	for _, ip := range v4 {
		log.Printf("  http://%s/", net.JoinHostPort(ip.String(), port))
	}
	for _, ip := range v6 {
		// JoinHostPort brackets IPv6 literals automatically, producing
		// the URL-safe "http://[2001:db8::1]:8080/" form.
		log.Printf("  http://%s/", net.JoinHostPort(ip.String(), port))
	}

	if len(v4)+len(v6) == 0 {
		log.Printf("  (no non-loopback interfaces detected — only the localhost URL works)")
	}

	// Inside a container the IPs above are container-internal (bridge
	// addresses like 172.x.x.x) and aren't reachable from the host's
	// LAN. Print a hint so users running `docker run -p 8080:8080 …`
	// know to navigate to their HOST's LAN IP instead.
	//
	// Detection is via the LEMURPOUCH_IN_CONTAINER env var, which the
	// project's Dockerfile sets explicitly. We deliberately don't
	// probe /.dockerenv: that would also fire for users who built a
	// custom container around the binary (or `docker cp`-ed it out
	// and re-ran it elsewhere). A binary that doesn't see this var
	// stays quiet and lets the URLs speak for themselves.
	if os.Getenv("LEMURPOUCH_IN_CONTAINER") != "" {
		log.Printf("  (running in a container — those interface IPs are container-internal;")
		log.Printf("   reach the relay from other LAN hosts via your host's LAN IP)")
	}
}

// isWildcard reports whether host designates "all interfaces" — either
// the empty string (Go's :PORT shorthand) or the unspecified IP
// literal in either family (0.0.0.0 / ::).
func isWildcard(host string) bool {
	if host == "" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsUnspecified()
}
