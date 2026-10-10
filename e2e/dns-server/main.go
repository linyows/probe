// Command dns-server answers DNS queries for a few records of example.test,
// over UDP and TCP.
//
// The dns action E2E tests ask it, so they depend on no resolver and on no
// name that someone else keeps.
package main

import (
	"flag"
	"log"
	"net"

	"github.com/miekg/dns"
)

// zone is what the server answers from, by name and type.
var zone = map[string][]string{
	"example.test. A":    {"example.test. 300 IN A 192.0.2.1", "example.test. 300 IN A 192.0.2.2"},
	"example.test. AAAA": {"example.test. 300 IN AAAA 2001:db8::1"},
	"example.test. MX":   {"example.test. 300 IN MX 10 mail.example.test."},
	"example.test. TXT":  {`example.test. 300 IN TXT "v=spf1 mx -all"`},
	"example.test. NS":   {"example.test. 300 IN NS ns1.example.test."},
	"example.test. CAA":  {`example.test. 300 IN CAA 0 issue "letsencrypt.org"`},
	"www.example.test. A": {
		"www.example.test. 300 IN CNAME example.test.",
		"example.test. 300 IN A 192.0.2.1",
	},
	"1.2.0.192.in-addr.arpa. PTR": {"1.2.0.192.in-addr.arpa. 300 IN PTR example.test."},
}

// names are the names the zone has a record of, of any type.
var names = map[string]bool{"example.test.": true, "www.example.test.": true, "1.2.0.192.in-addr.arpa.": true}

func answer(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	q := r.Question[0]
	if lines, ok := zone[q.Name+" "+dns.TypeToString[q.Qtype]]; ok {
		for _, line := range lines {
			rr, err := dns.NewRR(line)
			if err != nil {
				log.Fatal(err)
			}
			m.Answer = append(m.Answer, rr)
		}
	} else if !names[q.Name] {
		m.Rcode = dns.RcodeNameError
	}
	_ = w.WriteMsg(m)
}

func main() {
	addr := flag.String("addr", "127.0.0.1:15353", "address to listen on, over UDP and TCP")
	flag.Parse()

	// Listen first and say so, so that a step can wait for the line.
	pc, err := net.ListenPacket("udp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on %s", *addr)

	handler := dns.HandlerFunc(answer)
	go func() {
		log.Fatal((&dns.Server{Listener: ln, Handler: handler}).ActivateAndServe())
	}()
	log.Fatal((&dns.Server{PacketConn: pc, Handler: handler}).ActivateAndServe())
}
