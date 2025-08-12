package database

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
)

func (e databaseEntry) GobEncode() ([]byte, error) {
	temp := struct {
		Value string
		TTL   *int64
	}{
		e.value,
		e.ttl,
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(temp); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (e *databaseEntry) GobDecode(b []byte) error {
	var E struct {
		Value string `json:"value"`
		TTL   *int64 `json:"ttl"`
	}

	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&E); err != nil {
		return err
	}

	e.value = E.Value
	e.ttl = E.TTL

	return nil
}

func (t ttlHeapData) GobEncode() ([]byte, error) {
	temp := struct {
		Key string
		TTL int64
	}{
		Key: t.key,
		TTL: t.ttl,
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(temp); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (t *ttlHeapData) GobDecode(b []byte) error {
	var T struct {
		Key string
		TTL int64
	}

	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&T); err != nil {
		return err
	}

	t.key = T.Key
	t.ttl = T.TTL

	return nil
}

func (i *InMemoryDatabase) GobEncode() ([]byte, error) {
	temp := struct {
		DbStore dbStore  `json:"dbStore"`
		TTL     *ttlHeap `json:"ttlHeap"`
	}{
		DbStore: i.database,
		TTL:     i.ttl,
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(temp); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (i *InMemoryDatabase) GobDecode(b []byte) error {
	var I struct {
		DbStore dbStore
		TTL     *ttlHeap
	}

	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&I); err != nil {
		return err
	}

	i.database = I.DbStore
	i.ttl = I.TTL

	return nil
}

func (e databaseEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Value string `json:"value"`
		TTL   *int64 `json:"ttl"`
	}{
		Value: e.value,
		TTL:   e.ttl,
	})
}

func (e *databaseEntry) UnmarshalJSON(data []byte) error {
	var E struct {
		Value string `json:"value"`
		TTL   *int64 `json:"ttl"`
	}

	if err := json.Unmarshal(data, &E); err != nil {
		return err
	}

	e.value = E.Value
	e.ttl = E.TTL

	return nil
}

func (t ttlHeapData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Key string `json:"key"`
		TTL int64  `json:"ttl"`
	}{
		Key: t.key,
		TTL: t.ttl,
	})
}

func (t *ttlHeapData) UnmarshalJSON(data []byte) error {
	var T struct {
		Key string `json:"key"`
		TTL int64  `json:"ttl"`
	}

	if err := json.Unmarshal(data, &T); err != nil {
		return err
	}

	t.key = T.Key
	t.ttl = T.TTL

	return nil
}

func (i *InMemoryDatabase) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		DbStore dbStore  `json:"dbStore"`
		TTL     *ttlHeap `json:"ttlHeap"`
	}{
		DbStore: i.database,
		TTL:     i.ttl,
	})
}

func (i *InMemoryDatabase) UnmarshalJSON(data []byte) error {
	var I struct {
		DbStore dbStore  `json:"dbStore"`
		TTL     *ttlHeap `json:"ttlHeap"`
	}

	if err := json.Unmarshal(data, &I); err != nil {
		return err
	}

	i.database = I.DbStore
	i.ttl = I.TTL

	return nil
}
