package tests

import (
	"context"
	"fmt"
	"github.com/pthav/InMemoryDB/database"
	"github.com/pthav/InMemoryDB/handler"
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

type benchmarkHelperStruct struct {
	tests []struct {
		name     string   // The test case name
		validOps []string // The valid operations to be selected from
	}
	puSize      int
	putRequests []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Ttl   *int64 `json:"ttl"`
	}
	pu           *atomic.Int64
	poSize       int
	postRequests []struct {
		Value string `json:"value"`
		Ttl   *int64 `json:"ttl"`
	}
	po             *atomic.Int64
	gSize          int
	getRequests    []string
	g              *atomic.Int64
	gtSize         int
	dSize          int
	getTTLRequests []string
	gt             *atomic.Int64
	deleteRequests []string
	d              *atomic.Int64
	pubSize        int
	pubRequests    []struct {
		message string
		channel string
	}
	pub *atomic.Int64
}

// Setup randomly generated operations
func benchmarkHelper() benchmarkHelperStruct {
	b := benchmarkHelperStruct{}
	b.tests = []struct {
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

	b.puSize = 500000
	b.putRequests = make([]struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Ttl   *int64 `json:"ttl"`
	}, b.puSize)
	b.pu = new(atomic.Int64)

	b.poSize = 500000
	b.postRequests = make([]struct {
		Value string `json:"value"`
		Ttl   *int64 `json:"ttl"`
	}, b.poSize)
	b.po = new(atomic.Int64)

	b.gSize = 500000
	b.getRequests = make([]string, b.gSize)
	b.g = new(atomic.Int64)

	b.gtSize = 500000
	b.getTTLRequests = make([]string, b.gtSize)
	b.gt = new(atomic.Int64)

	b.dSize = 500000
	b.deleteRequests = make([]string, b.dSize)
	b.d = new(atomic.Int64)

	b.pubSize = 500000
	b.pubRequests = make([]struct {
		message string
		channel string
	}, b.pubSize)
	b.pub = new(atomic.Int64)

	for i := 0; i < b.puSize; i++ {
		b.putRequests = append(b.putRequests, generatePut())
	}

	for i := 0; i < b.poSize; i++ {
		b.postRequests = append(b.postRequests, generatePost())
	}

	for i := 0; i < b.gSize; i++ {
		b.getRequests = append(b.getRequests, randomString(10))
	}

	for i := 0; i < b.gtSize; i++ {
		b.getTTLRequests = append(b.getTTLRequests, randomString(10))
	}

	for i := 0; i < b.dSize; i++ {
		b.deleteRequests = append(b.deleteRequests, randomString(10))
	}

	for i := 0; i < b.pubSize; i++ {
		b.pubRequests = append(b.pubRequests, struct {
			message string
			channel string
		}{message: randomString(10), channel: randomString(2)})
	}

	return b
}

// BenchmarkDatabaseOperations only benchmarks the database
func BenchmarkDatabaseOperations(b *testing.B) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bstruct := benchmarkHelper()

	for _, tt := range bstruct.tests {
		if tt.name == "PUB only" {
			continue
		}

		tt := tt // Capture for go routines
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()

			db, _ := database.NewInMemoryDatabase(database.WithLogger(discardLogger))

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					funcType := "PUB"
					for funcType == "PUB" {
						funcType = tt.validOps[rand.Intn(len(tt.validOps))]
					}

					switch funcType {
					case "PUT":
						index := int(bstruct.pu.Add(1)) % bstruct.puSize
						db.Put(bstruct.putRequests[index])
					case "POST":
						index := int(bstruct.po.Add(1)) % bstruct.poSize
						db.Create(bstruct.postRequests[index])
					case "GET":
						index := int(bstruct.g.Add(1)) % bstruct.gSize
						db.Get(bstruct.getRequests[index])
					case "DELETE":
						index := int(bstruct.d.Add(1)) % bstruct.dSize
						db.Delete(bstruct.deleteRequests[index])
					case "TTL":
						index := int(bstruct.gt.Add(1)) % bstruct.gtSize
						db.GetTTL(bstruct.getTTLRequests[index])
					}
				}
			})
		})
	}
}

// BenchmarkHTTP benchmarks the http handler injected with InMemoryDatabase
func BenchmarkHTTP(b *testing.B) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bstruct := benchmarkHelper()

	for _, tt := range bstruct.tests {
		tt := tt // Capture for go routines
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()

			db, _ := database.NewInMemoryDatabase(database.WithLogger(discardLogger))
			h := handler.NewHandler(db, discardLogger)

			// Add 10,000 subscribers
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
						index := int(bstruct.pu.Add(1)) % bstruct.puSize
						url := "/v1/keys/" + bstruct.putRequests[index].Key
						var body string
						if bstruct.putRequests[index].Ttl != nil {
							body = fmt.Sprintf(`{"value": %v, "ttl": %v}`, bstruct.putRequests[index].Value, *bstruct.putRequests[index].Ttl)
						} else {
							body = fmt.Sprintf(`{"value": %v}`, bstruct.putRequests[index].Value)
						}
						r := httptest.NewRequest("PUT", url, strings.NewReader(body))
						h.ServeHTTP(httptest.NewRecorder(), r)
					case "POST":
						index := int(bstruct.po.Add(1)) % bstruct.poSize
						url := "/v1/keys"
						var body string
						if bstruct.postRequests[index].Ttl != nil {
							body = fmt.Sprintf(`{"value": %v, "ttl": %v}`, bstruct.postRequests[index].Value, *bstruct.postRequests[index].Ttl)
						} else {
							body = fmt.Sprintf(`{"value": %v}`, bstruct.postRequests[index].Value)
						}
						r := httptest.NewRequest("POST", url, strings.NewReader(body))
						h.ServeHTTP(httptest.NewRecorder(), r)
					case "GET":
						index := int(bstruct.g.Add(1)) % bstruct.gSize
						url := "/v1/keys/" + bstruct.getRequests[index]
						r := httptest.NewRequest("GET", url, strings.NewReader("{}"))
						h.ServeHTTP(httptest.NewRecorder(), r)
					case "DELETE":
						index := int(bstruct.d.Add(1)) % bstruct.dSize
						url := "/v1/keys/" + bstruct.deleteRequests[index]
						r := httptest.NewRequest("DELETE", url, strings.NewReader("{}"))
						h.ServeHTTP(httptest.NewRecorder(), r)
					case "TTL":
						index := int(bstruct.gt.Add(1)) % bstruct.gtSize
						url := "/v1/ttl/" + bstruct.getTTLRequests[index]
						r := httptest.NewRequest("GET", url, strings.NewReader("{}"))
						h.ServeHTTP(httptest.NewRecorder(), r)
					case "PUB":
						index := int(bstruct.pub.Add(1)) % bstruct.pubSize
						url := "/v1/publish/" + bstruct.pubRequests[index].channel
						body := fmt.Sprintf(`{"message":%v}`, bstruct.pubRequests[index].message)
						r := httptest.NewRequest("POST", url, strings.NewReader(body))
						h.ServeHTTP(httptest.NewRecorder(), r)
					}
				}
			})
		})
	}
}
