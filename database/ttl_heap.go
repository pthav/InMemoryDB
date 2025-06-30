package database

type ttlHeapData struct {
	key string
	ttl int64
}

type ttlHeap []ttlHeapData

func (t ttlHeap) Len() int {
	return len(t)
}

func (t ttlHeap) Less(i, j int) bool {
	return t[i].ttl < t[j].ttl
}

func (t ttlHeap) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t *ttlHeap) Push(x any) {
	*t = append(*t, x.(ttlHeapData))
}

func (t *ttlHeap) Pop() any {
	last := (*t)[t.Len()-1]
	*t = (*t)[:t.Len()-1]
	return last
}

func (t *ttlHeap) Peak() any {
	if len(*t) != 0 {
		return (*t)[0]
	}
	return nil
}
