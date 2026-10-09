// Command echo-server echoes a POSTed JSON body back as the data of the
// response, with the request headers.
//
// The external action E2E tests run mozership/probe-graphql against it, so they
// depend on GitHub, where the action is downloaded from, and on nothing else.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8091", "address to listen on")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			reply(w, map[string]any{"ok": true})
			return
		}
		var body any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		headers := map[string]string{}
		for k := range r.Header {
			headers[k] = r.Header.Get(k)
		}
		reply(w, map[string]any{"data": map[string]any{"request": body, "headers": headers}})
	})

	// Listen first and say so, so that a step can wait for the line.
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on %s", ln.Addr())
	log.Fatal(http.Serve(ln, nil))
}

func reply(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}
