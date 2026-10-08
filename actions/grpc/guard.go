package grpc

import (
	"net"
	"net/url"
	"strings"

	"github.com/linyows/probe/actionrpc"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// hostlessSchemes are the gRPC target schemes that name no host the guard
// can check, such as a Unix socket.
var hostlessSchemes = []string{"unix:", "unix-abstract:", "vsock:", "ipv4:", "ipv6:", "xds:", "google-c2p:"}

// grpcHost returns the host and port a gRPC call to addr connects to, as
// grpc-go reads the target: host:port, or dns:/// or passthrough:/// before
// it, with the port 443 when none is given. It returns false for a target
// that names no host, such as a Unix socket.
func grpcHost(addr string) (string, bool) {
	lower := strings.ToLower(addr)
	for _, s := range hostlessSchemes {
		if strings.HasPrefix(lower, s) {
			return "", false
		}
	}
	endpoint := addr
	if strings.Contains(addr, "://") {
		u, err := url.Parse(addr)
		if err != nil {
			return "", false
		}
		switch strings.ToLower(u.Scheme) {
		case "dns", "passthrough":
			endpoint = strings.TrimPrefix(u.Path, "/")
		default:
			return "", false
		}
	}
	if endpoint == "" {
		return "", false
	}
	if _, _, err := net.SplitHostPort(endpoint); err == nil {
		return endpoint, true
	}
	return net.JoinHostPort(strings.Trim(endpoint, "[]"), "443"), true
}

// checkGRPCHost returns a Refused error when the guard of the run does not
// allow the host the gRPC call connects to, or cannot tell it.
func (r *Req) checkGRPCHost() error {
	if len(r.guard.AllowHosts) == 0 {
		return nil
	}
	host, ok := grpcHost(r.Addr)
	if !ok {
		return actionrpc.Refuse("the addr %s names no host the run can check against the hosts it allows", r.Addr)
	}
	return r.guard.CheckHost(host)
}

// checkConnectHost returns a Refused error when the guard of the run does
// not allow the host of base, the URL a Connect call goes under, taken at
// the port of its scheme when it names none.
func (r *Req) checkConnectHost(base string) error {
	if len(r.guard.AllowHosts) == 0 {
		return nil
	}
	u, err := url.Parse(base)
	if err != nil {
		return err
	}
	host := u.Host
	if u.Port() == "" {
		port := "80"
		if u.Scheme == "https" {
			port = "443"
		}
		host = net.JoinHostPort(u.Hostname(), port)
	}
	return r.guard.CheckHost(host)
}

// checkReadOnly returns a Refused error when the run is read-only and the
// method may write: when a definition of it at hand, the server's or the
// .proto files', does not declare it idempotency_level = NO_SIDE_EFFECTS,
// or when there is none to tell. Either may be nil.
func (r *Req) checkReadOnly(server, spec protoreflect.MethodDescriptor) error {
	if !r.guard.ReadOnly {
		return nil
	}
	if server == nil && spec == nil {
		return actionrpc.Refuse("the run is read-only, and no definition of %s/%s tells that it does not write; give proto.files that declare it idempotency_level = NO_SIDE_EFFECTS", r.Service, r.Method)
	}
	for _, d := range []struct {
		whose string
		md    protoreflect.MethodDescriptor
	}{{"the server's definition declares", server}, {"the .proto files declare", spec}} {
		if d.md == nil {
			continue
		}
		if level := idempotencyLevel(d.md); level != descriptorpb.MethodOptions_NO_SIDE_EFFECTS {
			return actionrpc.Refuse("the run is read-only, and %s may write: %s it idempotency_level %s, not NO_SIDE_EFFECTS", d.md.FullName(), d.whose, level)
		}
	}
	return nil
}

// idempotencyLevel returns the idempotency_level md is declared with,
// IDEMPOTENCY_UNKNOWN when it is declared with none.
func idempotencyLevel(md protoreflect.MethodDescriptor) descriptorpb.MethodOptions_IdempotencyLevel {
	return protodesc.ToMethodDescriptorProto(md).GetOptions().GetIdempotencyLevel()
}
