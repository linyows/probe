package actionrpc

import (
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"

	"github.com/linyows/probe/pb"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Guard is what a run allows its actions to do. It is set by whoever runs
// probe, on the command line, rather than by the workflow, so that a
// workflow cannot loosen it. The zero Guard allows everything.
//
// A guard keeps a workflow from writing to, or reaching, what it was not
// meant to; it is not a sandbox. An action keeps to it only as far as the
// action can tell what it is about to do.
type Guard struct {
	// ReadOnly allows only what reads, and nothing that writes, such as an
	// HTTP POST or an UPDATE.
	ReadOnly bool
	// AllowHosts are the hosts actions may connect to: a name or an address,
	// with a port or without one for any port, or *.example.com for the
	// names under example.com. Empty allows any host.
	AllowHosts []string
	// AllowActions are the actions run under the guard although they do not
	// keep to it, as the person running probe asks.
	AllowActions []string
	// Keeping are the actions that keep to the guard themselves. Any other
	// action is refused under an active guard, unless AllowActions names it.
	Keeping []string
}

// Active reports whether the guard limits anything.
func (g Guard) Active() bool {
	return g.ReadOnly || len(g.AllowHosts) > 0
}

// Runs reports whether the action named uses may run under the guard: it
// keeps to the guard, or the person running probe allowed it.
func (g Guard) Runs(uses string) bool {
	return !g.Active() || slices.Contains(g.Keeping, uses) || slices.Contains(g.AllowActions, uses)
}

// AllowsHost reports whether the guard allows connecting to hostport, a
// host with or without a port.
func (g Guard) AllowsHost(hostport string) bool {
	if len(g.AllowHosts) == 0 {
		return true
	}
	host, port := splitHostPort(hostport)
	for _, allowed := range g.AllowHosts {
		h, p := splitHostPort(allowed)
		if p != "" && p != port {
			continue
		}
		if matchHost(h, host) {
			return true
		}
	}
	return false
}

// CheckHost returns a Refused error when the guard does not allow
// connecting to hostport.
func (g Guard) CheckHost(hostport string) error {
	if g.AllowsHost(hostport) {
		return nil
	}
	return Refuse("the host %s is not one the run allows (%s)", hostport, strings.Join(g.AllowHosts, ", "))
}

// splitHostPort splits s into its host, lowercased and without brackets,
// and its port, which is empty when s has none.
func splitHostPort(s string) (string, string) {
	if host, port, err := net.SplitHostPort(s); err == nil {
		return strings.ToLower(host), port
	}
	return strings.ToLower(strings.Trim(s, "[]")), ""
}

func matchHost(pattern, host string) bool {
	if suffix, ok := strings.CutPrefix(pattern, "*."); ok {
		return strings.HasSuffix(host, "."+suffix)
	}
	return pattern == host
}

// Refused is the error an action returns for what the guard of the run
// does not allow. The runner tells it from other errors of an action.
type Refused struct {
	Reason string
}

func (r *Refused) Error() string {
	return "refused: " + r.Reason
}

// Refuse returns a Refused error with the reason formatted.
func Refuse(format string, args ...any) error {
	return &Refused{Reason: fmt.Sprintf(format, args...)}
}

// IsRefused reports whether err is, or wraps, a Refused error.
func IsRefused(err error) bool {
	var r *Refused
	return errors.As(err, &r)
}

// refusedInfo marks the gRPC status a Refused error is carried in, apart
// from a PermissionDenied an action passes on from a server it called.
var refusedInfo = &errdetails.ErrorInfo{Domain: "probe", Reason: "GUARD_REFUSED"}

// toStatus turns a Refused error into the gRPC status that carries it to
// the runner, marked as a refusal, and leaves any other error as it is.
func toStatus(err error) error {
	var r *Refused
	if !errors.As(err, &r) {
		return err
	}
	s, detailErr := status.New(codes.PermissionDenied, r.Reason).WithDetails(refusedInfo)
	if detailErr != nil {
		return status.Error(codes.PermissionDenied, r.Reason)
	}
	return s.Err()
}

// fromStatus turns the gRPC status a Refused error was carried in back
// into one, and leaves any other error as it is: a PermissionDenied without
// the mark of a refusal stays an error of the action.
func fromStatus(err error) error {
	s, ok := status.FromError(err)
	if !ok || s.Code() != codes.PermissionDenied {
		return err
	}
	for _, d := range s.Details() {
		if info, ok := d.(*errdetails.ErrorInfo); ok && info.GetDomain() == refusedInfo.Domain && info.GetReason() == refusedInfo.Reason {
			return &Refused{Reason: s.Message()}
		}
	}
	return err
}

func guardToPB(g Guard) *pb.Guard {
	if !g.Active() && len(g.AllowActions) == 0 && len(g.Keeping) == 0 {
		return nil
	}
	return &pb.Guard{
		ReadOnly:     g.ReadOnly,
		AllowHosts:   g.AllowHosts,
		AllowActions: g.AllowActions,
		Keeping:      g.Keeping,
	}
}

func guardFromPB(g *pb.Guard) Guard {
	return Guard{
		ReadOnly:     g.GetReadOnly(),
		AllowHosts:   g.GetAllowHosts(),
		AllowActions: g.GetAllowActions(),
		Keeping:      g.GetKeeping(),
	}
}
