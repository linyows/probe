//go:build experiments2
// +build experiments2

package probe

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

// generateBinaryData creates binary data of specified size
func generateBinaryData(size int) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

// BenchmarkBinaryData tests performance impact of different binary data sizes
func BenchmarkBinaryData(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
	}

	for _, s := range sizes {
		binaryData := generateBinaryData(s.size)

		// Test data with binary content
		testData := map[string]any{
			"id":     123,
			"name":   "binary test",
			"binary": binaryData,
			"meta": map[string]any{
				"size": s.size,
				"type": "binary",
			},
		}

		// Test data with base64 encoded content (for structpb)
		testDataB64 := map[string]any{
			"id":     123,
			"name":   "binary test",
			"binary": base64.StdEncoding.EncodeToString(binaryData),
			"meta": map[string]any{
				"size": s.size,
				"type": "binary",
			},
		}

		b.Run("FlattenInterface_"+s.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				flattened := FlattenInterface(testData)
				_ = flattened
			}
		})

		b.Run("UnflattenInterface_"+s.name, func(b *testing.B) {
			flattened := FlattenInterface(testData)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				unflattened := UnflattenInterface(flattened)
				_ = unflattened
			}
		})

		b.Run("FlattenUnflattenRoundTrip_"+s.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				flattened := FlattenInterface(testData)
				unflattened := UnflattenInterface(flattened)
				_ = unflattened
			}
		})

		b.Run("StructpbNewStruct_"+s.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				structData, err := structpb.NewStruct(testDataB64)
				if err != nil {
					b.Fatal(err)
				}
				_ = structData
			}
		})

		b.Run("StructpbRoundTrip_"+s.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				structData, err := structpb.NewStruct(testDataB64)
				if err != nil {
					b.Fatal(err)
				}
				mapData := structData.AsMap()
				_ = mapData
			}
		})
	}
}

// BenchmarkHTTPLikeScenario simulates HTTP request/response with binary bodies
func BenchmarkHTTPLikeScenario(b *testing.B) {
	// Simulate different HTTP scenarios with binary content
	scenarios := []struct {
		name        string
		reqBodySize int
		resBodySize int
		description string
	}{
		{"SmallJSON", 512, 1024, "Small JSON request/response"},
		{"ImageUpload", 512 * 1024, 256, "Image upload with small response"},
		{"FileDownload", 256, 2 * 1024 * 1024, "Small request, large file download"},
		{"LargeData", 1024 * 1024, 1024 * 1024, "Large request and response"},
	}

	for _, scenario := range scenarios {
		reqBody := generateBinaryData(scenario.reqBodySize)
		resBody := generateBinaryData(scenario.resBodySize)

		httpData := map[string]any{
			"req": map[string]any{
				"method": "POST",
				"url":    "https://api.example.com/upload",
				"headers": map[string]any{
					"Content-Type": "application/octet-stream",
				},
				"body": reqBody,
			},
			"res": map[string]any{
				"status": "200 OK",
				"code":   200,
				"headers": map[string]any{
					"Content-Type": "application/octet-stream",
				},
				"body": resBody,
			},
		}

		httpDataB64 := map[string]any{
			"req": map[string]any{
				"method": "POST",
				"url":    "https://api.example.com/upload",
				"headers": map[string]any{
					"Content-Type": "application/octet-stream",
				},
				"body": base64.StdEncoding.EncodeToString(reqBody),
			},
			"res": map[string]any{
				"status": "200 OK",
				"code":   200,
				"headers": map[string]any{
					"Content-Type": "application/octet-stream",
				},
				"body": base64.StdEncoding.EncodeToString(resBody),
			},
		}

		b.Run("FlattenInterface_"+scenario.name, func(b *testing.B) {
			b.ReportAllocs()
			totalSize := scenario.reqBodySize + scenario.resBodySize
			b.SetBytes(int64(totalSize))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				flattened := FlattenInterface(httpData)
				unflattened := UnflattenInterface(flattened)
				_ = unflattened
			}
		})

		b.Run("Structpb_"+scenario.name, func(b *testing.B) {
			b.ReportAllocs()
			totalSize := scenario.reqBodySize + scenario.resBodySize
			b.SetBytes(int64(totalSize))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				structData, err := structpb.NewStruct(httpDataB64)
				if err != nil {
					b.Fatal(err)
				}
				mapData := structData.AsMap()
				_ = mapData
			}
		})
	}
}

// BenchmarkKeyCount measures performance based on number of flattened keys
func BenchmarkKeyCount(b *testing.B) {
	sizes := []int{1024, 10 * 1024, 100 * 1024}

	for _, size := range sizes {
		binaryData := generateBinaryData(size)

		testData := map[string]any{
			"binary": binaryData,
		}

		// First, see how many keys this creates
		flattened := FlattenInterface(testData)
		keyCount := len(flattened)

		b.Run("FlattenInterface_"+formatSize(size)+"_Keys"+formatNumber(keyCount), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				flattened := FlattenInterface(testData)
				_ = flattened
			}
			b.ReportMetric(float64(keyCount), "keys")
		})

		b.Run("UnflattenInterface_"+formatSize(size)+"_Keys"+formatNumber(keyCount), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				unflattened := UnflattenInterface(flattened)
				_ = unflattened
			}
			b.ReportMetric(float64(keyCount), "keys")
		})
	}
}

// formatSize formats byte size for display
func formatSize(bytes int) string {
	if bytes >= 1024*1024 {
		return formatNumber(bytes/(1024*1024)) + "MB"
	} else if bytes >= 1024 {
		return formatNumber(bytes/1024) + "KB"
	}
	return formatNumber(bytes) + "B"
}

// formatNumber formats numbers for display
func formatNumber(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%dM", n/1000000)
	} else if n >= 1000 {
		return fmt.Sprintf("%dK", n/1000)
	}
	return fmt.Sprintf("%d", n)
}
