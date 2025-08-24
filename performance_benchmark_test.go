//go:build experiments
// +build experiments

package probe

import (
	"encoding/json"
	"testing"

	"github.com/linyows/probe/testdata/performance"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Sample test data for benchmarking (compatible with structpb)
var testData = map[string]any{
	"user": map[string]any{
		"id":   123,
		"name": "John Doe",
		"age":  30,
		"email": "john@example.com",
		"profile": map[string]any{
			"bio":      "Software engineer with 10+ years of experience",
			"location": "San Francisco, CA",
			"skills": []any{"Go", "Python", "JavaScript", "Docker"},
			"languages": map[string]any{
				"english":  "native",
				"spanish":  "conversational",
				"japanese": "beginner",
			},
		},
		"settings": map[string]any{
			"theme":        "dark",
			"notifications": true,
			"privacy":      map[string]any{
				"profile_visible": true,
				"email_visible":   false,
			},
		},
	},
	"metadata": map[string]any{
		"created_at":    "2023-01-15T10:30:00Z",
		"last_login":    "2023-12-01T14:22:33Z",
		"login_count":   456,
		"is_verified":   true,
		"subscription":  map[string]any{
			"plan":       "premium",
			"expires_at": "2024-12-31T23:59:59Z",
			"features":   []any{"api_access", "priority_support", "advanced_analytics"},
		},
	},
	"config": map[string]any{
		"api": map[string]any{
			"timeout":      30,
			"max_retries":  3,
			"rate_limit":   100,
			"enable_cache": true,
		},
		"database": map[string]any{
			"host":         "localhost",
			"port":         5432,
			"name":         "myapp_db",
			"ssl_enabled":  false,
			"pool_size":    20,
		},
	},
}

// BenchmarkFlattenInterface benchmarks the current FlattenInterface approach
func BenchmarkFlattenInterface(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		flattened := FlattenInterface(testData)
		_ = flattened
	}
}

// BenchmarkUnflattenInterface benchmarks the current UnflattenInterface approach
func BenchmarkUnflattenInterface(b *testing.B) {
	flattened := FlattenInterface(testData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		unflattened := UnflattenInterface(flattened)
		_ = unflattened
	}
}

// BenchmarkFlattenUnflattenRoundTrip benchmarks the complete round-trip
func BenchmarkFlattenUnflattenRoundTrip(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		flattened := FlattenInterface(testData)
		unflattened := UnflattenInterface(flattened)
		_ = unflattened
	}
}

// BenchmarkStructpbNewStruct benchmarks google.protobuf.Struct creation
func BenchmarkStructpbNewStruct(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		structData, err := structpb.NewStruct(testData)
		if err != nil {
			b.Fatal(err)
		}
		_ = structData
	}
}

// BenchmarkStructpbAsMap benchmarks google.protobuf.Struct conversion back to map
func BenchmarkStructpbAsMap(b *testing.B) {
	structData, err := structpb.NewStruct(testData)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mapData := structData.AsMap()
		_ = mapData
	}
}

// BenchmarkStructpbRoundTrip benchmarks the complete google.protobuf.Struct round-trip
func BenchmarkStructpbRoundTrip(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		structData, err := structpb.NewStruct(testData)
		if err != nil {
			b.Fatal(err)
		}
		mapData := structData.AsMap()
		_ = mapData
	}
}

// BenchmarkProtobufMarshal benchmarks protobuf marshaling with map<string,string>
func BenchmarkProtobufMarshalFlat(b *testing.B) {
	flattened := FlattenInterface(testData)
	req := &performance.FlatDataRequest{
		Data: flattened,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := proto.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
		_ = data
	}
}

// BenchmarkProtobufMarshalStruct benchmarks protobuf marshaling with google.protobuf.Struct
func BenchmarkProtobufMarshalStruct(b *testing.B) {
	structData, err := structpb.NewStruct(testData)
	if err != nil {
		b.Fatal(err)
	}
	req := &performance.StructDataRequest{
		Data: structData,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := proto.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
		_ = data
	}
}

// BenchmarkProtobufUnmarshalFlat benchmarks protobuf unmarshaling with map<string,string>
func BenchmarkProtobufUnmarshalFlat(b *testing.B) {
	flattened := FlattenInterface(testData)
	req := &performance.FlatDataRequest{
		Data: flattened,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var unmarshaled performance.FlatDataRequest
		err := proto.Unmarshal(data, &unmarshaled)
		if err != nil {
			b.Fatal(err)
		}
		_ = unmarshaled
	}
}

// BenchmarkProtobufUnmarshalStruct benchmarks protobuf unmarshaling with google.protobuf.Struct
func BenchmarkProtobufUnmarshalStruct(b *testing.B) {
	structData, err := structpb.NewStruct(testData)
	if err != nil {
		b.Fatal(err)
	}
	req := &performance.StructDataRequest{
		Data: structData,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var unmarshaled performance.StructDataRequest
		err := proto.Unmarshal(data, &unmarshaled)
		if err != nil {
			b.Fatal(err)
		}
		_ = unmarshaled
	}
}

// BenchmarkJSONMarshal benchmarks standard JSON marshaling (baseline)
func BenchmarkJSONMarshal(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(testData)
		if err != nil {
			b.Fatal(err)
		}
		_ = data
	}
}

// BenchmarkJSONUnmarshal benchmarks standard JSON unmarshaling (baseline)
func BenchmarkJSONUnmarshal(b *testing.B) {
	data, err := json.Marshal(testData)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var unmarshaled map[string]any
		err := json.Unmarshal(data, &unmarshaled)
		if err != nil {
			b.Fatal(err)
		}
		_ = unmarshaled
	}
}

// BenchmarkCompleteWorkflowFlat benchmarks the complete workflow using FlattenInterface
func BenchmarkCompleteWorkflowFlat(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 1. Flatten the data
		flattened := FlattenInterface(testData)
		
		// 2. Create protobuf message
		req := &performance.FlatDataRequest{Data: flattened}
		
		// 3. Marshal to protobuf
		marshaled, err := proto.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
		
		// 4. Unmarshal from protobuf
		var unmarshaled performance.FlatDataRequest
		err = proto.Unmarshal(marshaled, &unmarshaled)
		if err != nil {
			b.Fatal(err)
		}
		
		// 5. Unflatten back to original structure
		result := UnflattenInterface(unmarshaled.Data)
		_ = result
	}
}

// BenchmarkCompleteWorkflowStruct benchmarks the complete workflow using google.protobuf.Struct
func BenchmarkCompleteWorkflowStruct(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 1. Convert to structpb.Struct
		structData, err := structpb.NewStruct(testData)
		if err != nil {
			b.Fatal(err)
		}
		
		// 2. Create protobuf message
		req := &performance.StructDataRequest{Data: structData}
		
		// 3. Marshal to protobuf
		marshaled, err := proto.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
		
		// 4. Unmarshal from protobuf
		var unmarshaled performance.StructDataRequest
		err = proto.Unmarshal(marshaled, &unmarshaled)
		if err != nil {
			b.Fatal(err)
		}
		
		// 5. Convert back to map
		result := unmarshaled.Data.AsMap()
		_ = result
	}
}