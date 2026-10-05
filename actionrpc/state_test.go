package actionrpc

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/pb"
	"google.golang.org/grpc"
)

// statefulAction keeps the number of times it ran in its state.
type statefulAction struct {
	MockActions
	keep bool
}

func (a *statefulAction) RunWithState(with, state map[string]any) (map[string]any, map[string]any, error) {
	if a.keep {
		return map[string]any{"kept": true}, nil, nil
	}
	n, _ := state["n"].(int64)
	return map[string]any{"n": n}, map[string]any{"n": n + 1, "secret": "s3cr3t"}, nil
}

// directClient calls a Server in the same process, as the gRPC client of a
// plugin would.
type directClient struct{ s *Server }

func (c directClient) Run(ctx context.Context, in *pb.RunRequest, _ ...grpc.CallOption) (*pb.RunResponse, error) {
	return c.s.Run(ctx, in)
}

func TestClientRunWithState(t *testing.T) {
	var logBuf bytes.Buffer
	log := hclog.New(&hclog.LoggerOptions{Output: &logBuf, Level: hclog.Debug})
	c := &Client{client: directClient{&Server{Impl: &statefulAction{}, log: log}}}

	result, state, err := c.RunWithState(map[string]any{"a": "b"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(result, map[string]any{"n": int64(0)}) {
		t.Errorf("result = %v", result)
	}
	if !reflect.DeepEqual(state, map[string]any{"n": int64(1), "secret": "s3cr3t"}) {
		t.Errorf("state = %v", state)
	}

	result, state, err = c.RunWithState(map[string]any{}, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["n"] != int64(1) || state["n"] != int64(2) {
		t.Errorf("the state should reach the action, got result %v and state %v", result, state)
	}

	// The state may hold credentials the runner has not learned to hide.
	if strings.Contains(logBuf.String(), "s3cr3t") {
		t.Errorf("the state should not be logged, got %s", logBuf.String())
	}
}

func TestClientRunWithStateOfAnActionWithout(t *testing.T) {
	for _, impl := range []Action{&MockActions{}, &statefulAction{keep: true}} {
		c := &Client{client: directClient{&Server{Impl: impl}}}
		_, state, err := c.RunWithState(map[string]any{}, map[string]any{"n": 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state != nil {
			t.Errorf("%T should leave no state, got %v", impl, state)
		}
	}
}

func TestClientRunDropsState(t *testing.T) {
	c := &Client{client: directClient{&Server{Impl: &statefulAction{}}}}
	result, err := c.Run(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(result, map[string]any{"n": int64(0)}) {
		t.Errorf("result = %v", result)
	}
}

// TestServerLogsResultWithCredentialsHidden checks that the result the server
// logs has the credentials in it hidden, since the runner learns them only
// once it has the result.
func TestServerLogsResultWithCredentialsHidden(t *testing.T) {
	var logBuf bytes.Buffer
	log := hclog.New(&hclog.LoggerOptions{Output: &logBuf, Level: hclog.Debug})
	s := &Server{Impl: &MockActions{RunFunc: func(map[string]any) (map[string]any, error) {
		return map[string]any{"res": map[string]any{
			"cookies": map[string]any{"session": "cookie-value"},
			"headers": map[string]any{"Set-Cookie": "session=header-value; Path=/"},
			"body":    `{"session":"cookie-value"}`,
		}}, nil
	}}, log: log}
	if _, err := s.Run(context.Background(), &pb.RunRequest{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range []string{"cookie-value", "header-value"} {
		if strings.Contains(logBuf.String(), v) {
			t.Errorf("the log should not hold %q, got %s", v, logBuf.String())
		}
	}
}
