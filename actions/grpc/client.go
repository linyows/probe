package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"sync/atomic"
	"time"

	"github.com/linyows/probe/mapping"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type Req struct {
	Addr     string            `map:"addr" validate:"required"`
	Service  string            `map:"service" validate:"required"`
	Method   string            `map:"method" validate:"required"`
	Body     string            `map:"body"`
	Timeout  string            `map:"timeout"`
	TLS      bool              `map:"tls"`
	Insecure bool              `map:"insecure"`
	CertFile string            `map:"cert_file"`
	KeyFile  string            `map:"key_file"`
	CAFile   string            `map:"ca_file"`
	Metadata map[string]string `map:"metadata"`
	cb       *Callback
	// contract, when it is set, checks the call against .proto files.
	contract *contract
}

type Res struct {
	Body          string            `map:"body"`
	StatusCode    string            `map:"status_code"`
	StatusMessage string            `map:"status_message"`
	Metadata      map[string]string `map:"metadata"`
	// violations are what the .proto files do not allow in the call; nil
	// when there is no contract.
	violations []any
}

type Result struct {
	Req    Req           `map:"req"`
	Res    Res           `map:"res"`
	RT     time.Duration `map:"rt"`
	Status int           `map:"status"`
}

func NewReq() *Req {
	return &Req{
		Timeout:  "30s",
		TLS:      false,
		Insecure: false,
		Metadata: make(map[string]string),
	}
}

func (r *Req) Do() (re *Result, er error) {
	if r.Addr == "" {
		return nil, errors.New("Req.Addr is required")
	}
	if r.Service == "" {
		return nil, errors.New("Req.Service is required")
	}
	if r.Method == "" {
		return nil, errors.New("Req.Method is required")
	}

	// Setup timeout
	timeout, err := time.ParseDuration(r.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout %q: use a duration such as 30s", r.Timeout)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Setup connection credentials
	var creds credentials.TransportCredentials
	if !r.TLS {
		// Plain text connection
		creds = insecure.NewCredentials()
	} else {
		if r.Insecure {
			// TLS without certificate verification (for development)
			creds = credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
		} else {
			// Normal TLS connection
			tlsConfig := &tls.Config{}

			// Custom CA certificate
			if r.CAFile != "" {
				caCert, err := os.ReadFile(r.CAFile)
				if err != nil {
					return nil, fmt.Errorf("failed to read CA file: %w", err)
				}
				caCertPool := x509.NewCertPool()
				if !caCertPool.AppendCertsFromPEM(caCert) {
					return nil, errors.New("failed to parse CA certificate")
				}
				tlsConfig.RootCAs = caCertPool
			}

			// Client certificate for mTLS
			if r.CertFile != "" && r.KeyFile != "" {
				cert, err := tls.LoadX509KeyPair(r.CertFile, r.KeyFile)
				if err != nil {
					return nil, fmt.Errorf("failed to load client certificate: %w", err)
				}
				tlsConfig.Certificates = []tls.Certificate{cert}
			}

			creds = credentials.NewTLS(tlsConfig)
		}
	}

	// Establish connection
	answer := &answerHandler{}
	conn, err := grpc.NewClient(r.Addr, grpc.WithTransportCredentials(creds), grpc.WithStatsHandler(answer))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	defer func() {
		err := conn.Close()
		if er == nil {
			er = err
		}
	}()

	// Prepare metadata
	md := metadata.New(r.Metadata)
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Callback before request
	if r.cb != nil && r.cb.before != nil {
		r.cb.before(ctx, r.Service, r.Method)
	}

	result := &Result{Req: *r}
	start := time.Now()

	// Use reflection to get service descriptor
	reflectionClient := grpc_reflection_v1alpha.NewServerReflectionClient(conn)
	res, err := r.invokeMethod(ctx, conn, reflectionClient, answer)
	result.RT = time.Since(start)

	if err != nil {
		result.Status = 1 // failure
		return result, err
	}

	result.Res = *res
	result.Status = 0 // success
	if res.StatusCode != "OK" {
		result.Status = 1
	}

	// Callback after response
	if r.cb != nil && r.cb.after != nil {
		r.cb.after(res)
	}

	return result, nil
}

func (r *Req) invokeMethod(ctx context.Context, conn *grpc.ClientConn, reflectionClient grpc_reflection_v1alpha.ServerReflectionClient, answer *answerHandler) (*Res, error) {
	// Get service descriptor using reflection
	serviceDesc, err := r.getServiceDescriptor(ctx, reflectionClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get service descriptor: %w", err)
	}

	// Find method descriptor
	methodDesc := serviceDesc.Methods().ByName(protoreflect.Name(r.Method))
	if methodDesc == nil {
		return nil, fmt.Errorf("method %s not found in service %s", r.Method, r.Service)
	}

	// The .proto files, when they are given, are checked against the
	// request, the server's definition and then the response.
	var violations []any
	var spec protoreflect.MethodDescriptor
	if r.contract != nil {
		violations = []any{}
		spec = r.contract.method(r.Service, r.Method)
		if spec == nil {
			violations = append(violations, violation("response", fmt.Sprintf("the .proto files declare no method %s in %s", r.Method, r.Service), "", ""))
		} else {
			violations = append(violations, r.contract.checkRequest(spec, r.Body)...)
			violations = append(violations, r.contract.checkDefinition(spec, methodDesc)...)
		}
	}

	// Create dynamic message for request
	requestMsg := dynamicpb.NewMessage(methodDesc.Input())
	if r.Body != "" {
		if err := protojson.Unmarshal([]byte(r.Body), requestMsg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request JSON: %w", err)
		}
	}

	// Create dynamic message for response
	responseMsg := dynamicpb.NewMessage(methodDesc.Output())

	// Invoke the method
	fullMethodName := fmt.Sprintf("/%s/%s", serviceDesc.FullName(), methodDesc.Name())
	var header, trailer metadata.MD
	err = conn.Invoke(answer.mark(ctx), fullMethodName, requestMsg, responseMsg, grpc.Header(&header), grpc.Trailer(&trailer))

	// The server's metadata arrives as headers before the reply and trailers
	// after it; a trailer wins over a header of the same name.
	metadataMap := make(map[string]string)
	for _, md := range []metadata.MD{header, trailer} {
		for key, values := range md {
			if len(values) > 0 {
				metadataMap[key] = values[0] // Take first value
			}
		}
	}

	// Convert response to JSON
	responseJSON := ""
	if responseMsg != nil {
		responseBytes, jsonErr := protojson.Marshal(responseMsg)
		if jsonErr == nil {
			responseJSON = string(responseBytes)
		}
	}

	res := &Res{
		Body:          responseJSON,
		StatusCode:    "OK",
		StatusMessage: "",
		Metadata:      metadataMap,
		violations:    violations,
	}
	// Only a call that succeeded answers with a message to check.
	if err == nil && spec != nil {
		res.violations = append(res.violations, r.contract.checkResponse(spec, responseMsg)...)
	}

	// A status the server sent is its answer, which a test can check like an
	// HTTP status code. Every error from a call is a status, but one made up
	// on this side, such as the timeout running out or the connection
	// failing, arrives without the server's trailers and stays an error.
	if err != nil {
		st, ok := status.FromError(err)
		if !ok || !answer.answered.Load() {
			return nil, err
		}
		res.StatusCode = statusCodeName(st.Code())
		res.StatusMessage = st.Message()
	}

	return res, nil
}

// answerHandler records whether the server sent the trailers that carry the
// status of the call marked with mark. It ignores the reflection lookup that
// goes over the same connection.
type answerHandler struct {
	answered atomic.Bool
}

type answerKey struct{}

// mark returns the context to make the call with.
func (h *answerHandler) mark(ctx context.Context) context.Context {
	return context.WithValue(ctx, answerKey{}, true)
}

func (h *answerHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	return ctx
}

func (h *answerHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	if _, ok := s.(*stats.InTrailer); ok && ctx.Value(answerKey{}) != nil {
		h.answered.Store(true)
	}
}

func (h *answerHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h *answerHandler) HandleConn(context.Context, stats.ConnStats) {}

// statusCodeName returns the canonical name of a gRPC status code, such as
// NOT_FOUND, which is how the gRPC specification and other tools spell it.
func statusCodeName(c codes.Code) string {
	switch c {
	case codes.OK:
		return "OK"
	case codes.Canceled:
		return "CANCELLED"
	case codes.Unknown:
		return "UNKNOWN"
	case codes.InvalidArgument:
		return "INVALID_ARGUMENT"
	case codes.DeadlineExceeded:
		return "DEADLINE_EXCEEDED"
	case codes.NotFound:
		return "NOT_FOUND"
	case codes.AlreadyExists:
		return "ALREADY_EXISTS"
	case codes.PermissionDenied:
		return "PERMISSION_DENIED"
	case codes.ResourceExhausted:
		return "RESOURCE_EXHAUSTED"
	case codes.FailedPrecondition:
		return "FAILED_PRECONDITION"
	case codes.Aborted:
		return "ABORTED"
	case codes.OutOfRange:
		return "OUT_OF_RANGE"
	case codes.Unimplemented:
		return "UNIMPLEMENTED"
	case codes.Internal:
		return "INTERNAL"
	case codes.Unavailable:
		return "UNAVAILABLE"
	case codes.DataLoss:
		return "DATA_LOSS"
	case codes.Unauthenticated:
		return "UNAUTHENTICATED"
	default:
		return c.String()
	}
}

func (r *Req) getServiceDescriptor(ctx context.Context, client grpc_reflection_v1alpha.ServerReflectionClient) (re protoreflect.ServiceDescriptor, er error) {
	stream, err := client.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reflection stream: %w", err)
	}
	defer func() {
		err := stream.CloseSend()
		if er == nil {
			er = err
		}
	}()

	// Request service descriptor
	//nolint:staticcheck // v1alpha reflection API is still widely used
	err = stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: r.Service,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send reflection request: %w", err)
	}

	// Receive response
	resp, err := stream.Recv()
	if err != nil {
		if err == io.EOF {
			return nil, errors.New("unexpected end of reflection stream")
		}
		return nil, fmt.Errorf("failed to receive reflection response: %w", err)
	}

	// Handle error response
	//nolint:staticcheck // v1alpha reflection API is still widely used
	if errResp := resp.GetErrorResponse(); errResp != nil {
		//nolint:staticcheck // v1alpha reflection API is still widely used
		return nil, fmt.Errorf("reflection error: %s", errResp.GetErrorMessage())
	}

	// Get file descriptor response
	//nolint:staticcheck // v1alpha reflection API is still widely used
	fileDescResp := resp.GetFileDescriptorResponse()
	if fileDescResp == nil {
		return nil, errors.New("unexpected response type from reflection")
	}

	// Parse file descriptors
	var fileDesc protoreflect.FileDescriptor
	//nolint:staticcheck // v1alpha reflection API is still widely used
	for _, fdBytes := range fileDescResp.GetFileDescriptorProto() {
		fd := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(fdBytes, fd); err != nil {
			return nil, fmt.Errorf("failed to unmarshal file descriptor: %w", err)
		}

		parsedFd, err := protodesc.NewFile(fd, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create file descriptor: %w", err)
		}

		// Check if this file contains our service
		services := parsedFd.Services()
		for i := 0; i < services.Len(); i++ {
			service := services.Get(i)
			serviceName := string(service.Name())
			serviceFullName := string(service.FullName())
			// Support both short name and full name
			if serviceName == r.Service || serviceFullName == r.Service {
				fileDesc = parsedFd
				break
			}
		}
	}

	if fileDesc == nil {
		return nil, fmt.Errorf("service %s not found", r.Service)
	}

	// Find the service in the file descriptor
	services := fileDesc.Services()
	for i := 0; i < services.Len(); i++ {
		service := services.Get(i)
		serviceName := string(service.Name())
		serviceFullName := string(service.FullName())
		// Support both short name and full name
		if serviceName == r.Service || serviceFullName == r.Service {
			return service, nil
		}
	}

	return nil, fmt.Errorf("service %s not found in file descriptor", r.Service)
}

type Option func(*Callback)

type Callback struct {
	before func(ctx context.Context, service, method string)
	after  func(res *Res)
}

func Request(data map[string]any, opts ...Option) (map[string]any, error) {
	// Create a copy to avoid modifying the original data
	m := make(map[string]any)
	maps.Copy(m, data)

	// Handle body conversion for structured data
	if bodyData, bodyExists := m["body"]; bodyExists {
		if bodyMap, isMap := bodyData.(map[string]any); isMap {
			// Convert body map to JSON string
			if jsonBytes, err := json.Marshal(bodyMap); err == nil {
				m["body"] = string(jsonBytes)
			}
		}
	}

	contract, err := takeProto(m)
	if err != nil {
		return map[string]any{}, err
	}

	m = mapping.HeaderToStringValue(m)

	// Create new request
	r := NewReq()
	r.contract = contract

	cb := &Callback{}
	for _, opt := range opts {
		opt(cb)
	}
	r.cb = cb

	if err := mapping.MapToStructByTags(m, r); err != nil {
		return map[string]any{}, err
	}

	ret, err := r.Do()
	if err != nil {
		return map[string]any{}, err
	}

	mapRet, err := mapping.StructToMapByTags(ret)
	if err != nil {
		return map[string]any{}, err
	}

	// The violations are in res, where the runner looks for them.
	if ret.Res.violations != nil {
		if res, ok := mapRet["res"].(map[string]any); ok {
			res["violations"] = ret.Res.violations
		}
	}

	return mapRet, nil
}

func WithBefore(f func(ctx context.Context, service, method string)) Option {
	return func(c *Callback) {
		c.before = f
	}
}

func WithAfter(f func(res *Res)) Option {
	return func(c *Callback) {
		c.after = f
	}
}
