package tests

import (
	"InMemoryDB/database"
	"InMemoryDB/handler"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Helper functions for method parameter generations
func intToPtr(i int64) *int64 {
	return &i
}

func randomString(length int) string {
	var builder strings.Builder
	builder.Grow(length)
	for i := 0; i < length; i++ {
		builder.WriteByte(byte(rand.Intn(75-65) + 65))
	}
	return builder.String()
}

func generatePut() struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Ttl   *int64 `json:"ttl"`
} {
	data := struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Ttl   *int64 `json:"ttl"`
	}{
		Key:   randomString(10),
		Value: randomString(10),
	}

	if rand.Intn(2) == 1 {
		data.Ttl = intToPtr(int64(rand.Intn(2)))
	}

	return data
}

func generatePost() struct {
	Value string `json:"value"`
	Ttl   *int64 `json:"ttl"`
} {
	data := struct {
		Value string `json:"value"`
		Ttl   *int64 `json:"ttl"`
	}{
		Value: randomString(10),
	}

	if rand.Intn(2) == 1 {
		data.Ttl = intToPtr(int64(rand.Intn(1000)))
	}

	return data
}

// BenchmarkDatabaseOperations only benchmarks the database
func BenchmarkDatabaseOperations(b *testing.B) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name     string   // The test case name
		validOps []string // The valid operations to be selected from
	}{
		{
			name:     "PUT only",
			validOps: []string{"PUT"},
		},
		{
			name:     "CREATE only",
			validOps: []string{"POST"},
		},
		{
			name:     "GET only",
			validOps: []string{"GET"},
		},
		{
			name:     "TTL only",
			validOps: []string{"TTL"},
		},
		{
			name:     "DELETE only",
			validOps: []string{"DELETE"},
		},
		{
			name:     "ALL",
			validOps: []string{"GET", "POST", "PUT", "DELETE", "TTL"},
		},
	}

	puSize := 500000
	putRequests := make([]struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Ttl   *int64 `json:"ttl"`
	}, puSize)
	for i := 0; i < puSize; i++ {
		putRequests = append(putRequests, generatePut())
	}
	var pu atomic.Int64

	poSize := 500000
	postRequests := make([]struct {
		Value string `json:"value"`
		Ttl   *int64 `json:"ttl"`
	}, poSize)
	for i := 0; i < poSize; i++ {
		postRequests = append(postRequests, generatePost())
	}
	var po atomic.Int64

	gSize := 500000
	getRequests := make([]string, gSize)
	for i := 0; i < gSize; i++ {
		getRequests = append(getRequests, randomString(10))
	}
	var g atomic.Int64

	gtSize := 500000
	getTTLRequests := make([]string, gtSize)
	for i := 0; i < gtSize; i++ {
		getTTLRequests = append(getTTLRequests, randomString(10))
	}
	var gt atomic.Int64

	dSize := 500000
	deleteRequests := make([]string, dSize)
	for i := 0; i < dSize; i++ {
		deleteRequests = append(deleteRequests, randomString(10))
	}
	var d atomic.Int64

	for _, tt := range tests {
		tt := tt // Capture for go routines
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()

			db, _ := database.NewInMemoryDatabase(database.WithLogger(discardLogger))

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					funcType := tt.validOps[rand.Intn(len(tt.validOps))]
					switch funcType {
					case "PUT":
						index := int(pu.Add(1)) % puSize
						db.Put(putRequests[index])
					case "POST":
						index := int(po.Add(1)) % poSize
						db.Create(postRequests[index])
					case "GET":
						index := int(g.Add(1)) % gSize
						db.Get(getRequests[index])
					case "DELETE":
						index := int(d.Add(1)) % dSize
						db.Delete(deleteRequests[index])
					case "TTL":
						index := int(gt.Add(1)) % gtSize
						db.GetTTL(getTTLRequests[index])
					}
				}
			})
		})
	}
}

// BenchmarkHTTP benchmarks the http handler injected with InMemoryDatabase
func BenchmarkHTTP(b *testing.B) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name     string   // The test case name
		validOps []string // The valid operations to be selected from
	}{
		{
			name:     "PUT only",
			validOps: []string{"PUT"},
		},
		{
			name:     "CREATE only",
			validOps: []string{"POST"},
		},
		{
			name:     "GET only",
			validOps: []string{"GET"},
		},
		{
			name:     "TTL only",
			validOps: []string{"TTL"},
		},
		{
			name:     "DELETE only",
			validOps: []string{"DELETE"},
		},
		{
			name:     "PUB only",
			validOps: []string{"PUB"},
		},
		{
			name:     "ALL",
			validOps: []string{"GET", "POST", "PUT", "DELETE", "TTL", "PUB"},
		},
	}

	puSize := 500000
	putRequests := make([]struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Ttl   *int64 `json:"ttl"`
	}, puSize)
	for i := 0; i < puSize; i++ {
		putRequests = append(putRequests, generatePut())
	}
	var pu atomic.Int64

	poSize := 500000
	postRequests := make([]struct {
		Value string `json:"value"`
		Ttl   *int64 `json:"ttl"`
	}, poSize)
	for i := 0; i < poSize; i++ {
		postRequests = append(postRequests, generatePost())
	}
	var po atomic.Int64

	gSize := 500000
	getRequests := make([]string, gSize)
	for i := 0; i < gSize; i++ {
		getRequests = append(getRequests, randomString(10))
	}
	var g atomic.Int64

	gtSize := 500000
	getTTLRequests := make([]string, gtSize)
	for i := 0; i < gtSize; i++ {
		getTTLRequests = append(getTTLRequests, randomString(10))
	}
	var gt atomic.Int64

	dSize := 500000
	deleteRequests := make([]string, dSize)
	for i := 0; i < dSize; i++ {
		deleteRequests = append(deleteRequests, randomString(10))
	}
	var d atomic.Int64

	pubSize := 500000
	pubRequests := make([]struct {
		message string
		channel string
	}, pubSize)
	for i := 0; i < pubSize; i++ {
		pubRequests = append(pubRequests, struct {
			message string
			channel string
		}{message: randomString(10), channel: randomString(2)})
	}
	var pub atomic.Int64

	for _, tt := range tests {
		tt := tt // Capture for go routines
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()

			db, _ := database.NewInMemoryDatabase(database.WithLogger(discardLogger))
			h := handler.NewHandler(db, discardLogger)

			// Add 100,000 subscribers
			if slices.Contains(tt.validOps, "PUB") {
				subSize := 10000
				sUrl := "/v1/subscribe/"
				for i := 0; i < subSize; i++ {
					channel := randomString(2)
					ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond*time.Duration(10))
					defer cancel()
					r := httptest.NewRequestWithContext(ctx, "GET", sUrl+channel, nil)
					go h.ServeHTTP(httptest.NewRecorder(), r)
				}
			}

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					funcType := tt.validOps[rand.Intn(len(tt.validOps))]
					switch funcType {
					case "PUT":
						index := int(pu.Add(1)) % puSize
						url := "/v1/keys/" + putRequests[index].Key
						var body string
						if putRequests[index].Ttl != nil {
							body = fmt.Sprintf(`{"value": %v, "ttl": %v}`, putRequests[index].Value, *putRequests[index].Ttl)
						} else {
							body = fmt.Sprintf(`{"value": %v}`, putRequests[index].Value)
						}
						r := httptest.NewRequest("PUT", url, strings.NewReader(body))
						h.ServeHTTP(httptest.NewRecorder(), r)
					case "POST":
						index := int(po.Add(1)) % poSize
						url := "/v1/keys"
						var body string
						if postRequests[index].Ttl != nil {
							body = fmt.Sprintf(`{"value": %v, "ttl": %v}`, postRequests[index].Value, *postRequests[index].Ttl)
						} else {
							body = fmt.Sprintf(`{"value": %v}`, postRequests[index].Value)
						}
						r := httptest.NewRequest("POST", url, strings.NewReader(body))
						h.ServeHTTP(httptest.NewRecorder(), r)
					case "GET":
						index := int(g.Add(1)) % gSize
						url := "/v1/keys/" + getRequests[index]
						r := httptest.NewRequest("GET", url, strings.NewReader("{}"))
						h.ServeHTTP(httptest.NewRecorder(), r)
					case "DELETE":
						index := int(d.Add(1)) % dSize
						url := "/v1/keys/" + deleteRequests[index]
						r := httptest.NewRequest("DELETE", url, strings.NewReader("{}"))
						h.ServeHTTP(httptest.NewRecorder(), r)
					case "TTL":
						index := int(gt.Add(1)) % gtSize
						url := "/v1/ttl/" + getTTLRequests[index]
						r := httptest.NewRequest("GET", url, strings.NewReader("{}"))
						h.ServeHTTP(httptest.NewRecorder(), r)
					case "PUB":
						index := int(pub.Add(1)) % pubSize
						url := "/v1/publish/" + pubRequests[index].channel
						body := fmt.Sprintf(`{"message":%v}`, pubRequests[index].message)
						r := httptest.NewRequest("POST", url, strings.NewReader(body))
						h.ServeHTTP(httptest.NewRecorder(), r)
					}
				}
			})
		})
	}
}

// BenchmarkCLI benchmarks the cli commands with the http handler and InMemoryDatabase
func BenchmarkCLI(b *testing.B) {

}
