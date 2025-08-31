package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/linyows/probe"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
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
}

type Res struct {
	Body          string            `map:"body"`
	StatusCode    string            `map:"status_code"`
	StatusMessage string            `map:"status_message"`
	Metadata      map[string]string `map:"metadata"`
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

// ConvertMetadataToMap converts flat metadata data to nested map structure
func ConvertMetadataToMap(data map[string]string) error {
	metadataData := map[string]string{}

	// Extract all metadata__ prefixed keys
	for key, value := range data {
		if strings.HasPrefix(key, "metadata__") {
			newKey := strings.TrimPrefix(key, "metadata__")
			metadataData[newKey] = value
			delete(data, key)
		}
	}

	// Add individual metadata fields back to data with metadata prefix
	for key, value := range metadataData {
		data["metadata__"+key] = value
	}

	return nil
}

// PrepareGrpcRequestData prepares all request data including body and metadata conversion
func PrepareGrpcRequestData(data map[string]string) error {
	// Convert body__ fields to JSON (reuse existing HTTP functionality)
	if err := probe.ConvertBodyToJson(data); err != nil {
		return err
	}

	// Handle metadata__ fields
	if err := ConvertMetadataToMap(data); err != nil {
		return err
	}

	return nil
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
		timeout = 30 * time.Second
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
	conn, err := grpc.NewClient(r.Addr, grpc.WithTransportCredentials(creds))
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
	res, err := r.invokeMethod(ctx, conn, reflectionClient)
	result.RT = time.Since(start)

	if err != nil {
		result.Status = 1 // failure
		return result, err
	}

	result.Res = *res
	result.Status = 0 // success

	// Callback after response
	if r.cb != nil && r.cb.after != nil {
		r.cb.after(res)
	}

	return result, nil
}

func (r *Req) invokeMethod(ctx context.Context, conn *grpc.ClientConn, reflectionClient grpc_reflection_v1alpha.ServerReflectionClient) (*Res, error) {
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
	err = conn.Invoke(ctx, fullMethodName, requestMsg, responseMsg)

	// Extract response metadata
	var responseMD metadata.MD
	responseMD, _ = metadata.FromIncomingContext(ctx)

	// Convert metadata to map
	metadataMap := make(map[string]string)
	for key, values := range responseMD {
		if len(values) > 0 {
			metadataMap[key] = values[0] // Take first value
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
	}

	if err != nil {
		res.StatusCode = "ERROR"
		res.StatusMessage = err.Error()
		return res, err
	}

	return res, nil
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
	// Convert map[string]any to map[string]string for internal processing
	dataCopy := make(map[string]string)
	for k, v := range data {
		if str, ok := v.(string); ok {
			dataCopy[k] = str
		} else if k == "body" {
			// Special handling for body field - convert to JSON
			if bodyMap, ok := v.(map[string]any); ok {
				if jsonBytes, err := json.Marshal(bodyMap); err == nil {
					dataCopy[k] = string(jsonBytes)
				} else {
					dataCopy[k] = fmt.Sprintf("%v", v)
				}
			} else {
				dataCopy[k] = fmt.Sprintf("%v", v)
			}
		} else {
			dataCopy[k] = fmt.Sprintf("%v", v)
		}
	}

	// Prepare request data (convert request and metadata fields)
	if err := PrepareGrpcRequestData(dataCopy); err != nil {
		return map[string]any{}, err
	}

	m := probe.HeaderToStringValue(dataCopy)

	// Create new request
	r := NewReq()

	cb := &Callback{}
	for _, opt := range opts {
		opt(cb)
	}
	r.cb = cb

	if err := probe.MapToStructByTags(m, r); err != nil {
		return map[string]any{}, err
	}

	ret, err := r.Do()
	if err != nil {
		return map[string]any{}, err
	}

	mapRet, err := probe.StructToMapByTags(ret)
	if err != nil {
		return map[string]any{}, err
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
