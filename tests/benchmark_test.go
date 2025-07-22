package tests

import (
	"InMemoryDB/database"
	"io"
	"log/slog"
	"math/rand"
	"strings"
	"sync/atomic"
	"testing"
)

// Helper functions for method parameter generations
func intToPtr(i int64) *int64 {
	return &i
}

func randomString() string {
	var builder strings.Builder
	length := rand.Intn(10)
	builder.Grow(length)
	for i := 0; i < length; i++ {
		builder.WriteByte(byte(rand.Intn(96-65) + 65))
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
		Key:   randomString(),
		Value: randomString(),
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
		Value: randomString(),
	}

	if rand.Intn(2) == 1 {
		data.Ttl = intToPtr(int64(rand.Intn(1000)))
	}

	return data
}

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
		getRequests = append(getRequests, randomString())
	}
	var g atomic.Int64

	gtSize := 500000
	getTTLRequests := make([]string, gtSize)
	for i := 0; i < gtSize; i++ {
		getTTLRequests = append(getTTLRequests, randomString())
	}
	var gt atomic.Int64

	dSize := 500000
	deleteRequests := make([]string, dSize)
	for i := 0; i < dSize; i++ {
		deleteRequests = append(deleteRequests, randomString())
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
						index := int(po.Add(1)) % puSize
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

func BenchmarkHTTP(b *testing.B) {

}

func BenchmarkAll(b *testing.B) {

}
