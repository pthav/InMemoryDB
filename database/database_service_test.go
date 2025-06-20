package database

import (
	"testing"
)

func TestInMemoryDatabase_Create(t *testing.T) {
	type test []struct {
		value       string
		ttl         *int
		want        bool
		loadedValue string
	}

	tests := []struct {
		name  string
		cases test
	}{
		{
			name: "Create a single new entry",
			cases: test{
				{
					value:       "value",
					want:        true,
					loadedValue: "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := NewInMemoryDatabase()
			for _, testCase := range tt.cases {
				data := struct {
					Value string `json:"value"`
					Ttl   *int   `json:"ttl"`
				}{
					Value: testCase.value,
					Ttl:   testCase.ttl,
				}

				loaded, key := i.Create(data)
				if loaded != testCase.want {
					t.Errorf("Create() = %v, want %v", loaded, testCase.want)
				}

				val, loaded := i.store.Load(key)
				if val != testCase.loadedValue {
					t.Errorf("Error loading value: Create() = %v, want %v where loaded = %v", val, testCase.loadedValue, loaded)
				}
			}
		})
	}
}

func TestInMemoryDatabase_Read(t *testing.T) {
	type test []struct {
		key        string
		wantValue  string
		wantLoaded bool
	}

	tests := []struct {
		name  string
		cases test
	}{
		{
			name: "Read an existing entry",
			cases: test{
				{
					key:        "key",
					wantValue:  "value",
					wantLoaded: true,
				},
			},
		},
		{
			name: "Read a non-existing entry",
			cases: test{
				{
					key:        "yo",
					wantValue:  "",
					wantLoaded: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := NewInMemoryDatabase()
			i.Update(struct {
				Key   string `json:"key"`
				Value string `json:"value"`
				Ttl   *int   `json:"ttl"`
			}{
				Key:   "key",
				Value: "value",
			})
			for _, testCase := range tt.cases {
				if val, loaded := i.Read(testCase.key); loaded != testCase.wantLoaded || val != testCase.wantValue {
					t.Errorf("Read() = %v + %v, want %v + %v", testCase.wantValue, testCase.wantLoaded, val, loaded)
				}
			}
		})
	}
}

func TestInMemoryDatabase_Update(t *testing.T) {
	type test []struct {
		key         string
		value       string
		ttl         *int
		want        bool
		loadedValue string
	}

	tests := []struct {
		name  string
		cases test
	}{
		{
			name: "Update an existing entry",
			cases: test{
				{
					key:         "key",
					value:       "value",
					want:        false,
					loadedValue: "value",
				},
				{
					key:         "key",
					value:       "value",
					want:        true,
					loadedValue: "value",
				},
			},
		},
		{
			name: "Update a non-existing entry",
			cases: test{
				{
					key:         "dog",
					value:       "hello",
					want:        false,
					loadedValue: "hello",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := NewInMemoryDatabase()
			for _, testCase := range tt.cases {
				data := struct {
					Key   string `json:"key"`
					Value string `json:"value"`
					Ttl   *int   `json:"ttl"`
				}{
					Key:   testCase.key,
					Value: testCase.value,
					Ttl:   testCase.ttl,
				}
				if loaded := i.Update(data); loaded != testCase.want {
					t.Errorf("Update() = %v, want %v", loaded, testCase.want)
				}

				val, loaded := i.store.Load(testCase.key)
				if val != testCase.loadedValue {
					t.Errorf("Error loading value: Update() = %v, want %v where loaded = %v", val, testCase.loadedValue, loaded)
				}
			}
		})
	}
}

func TestInMemoryDatabase_Delete(t *testing.T) {
	type test []struct {
		key  string
		want bool
	}

	tests := []struct {
		name  string
		cases test
	}{
		{
			name: "Delete an existing entry",
			cases: test{
				{
					key:  "key",
					want: true,
				},
			},
		},
		{
			name: "Delete a non-existing entry",
			cases: test{
				{
					key:  "yo",
					want: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := NewInMemoryDatabase()
			i.Update(struct {
				Key   string `json:"key"`
				Value string `json:"value"`
				Ttl   *int   `json:"ttl"`
			}{
				Key:   "key",
				Value: "value",
			})
			for _, testCase := range tt.cases {
				if loaded := i.Delete(testCase.key); loaded != testCase.want {
					t.Errorf("Delete() = %v, want %v", loaded, testCase.want)
				}
			}
		})
	}
}
