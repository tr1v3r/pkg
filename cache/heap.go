package cache

import "time"

// An entry belongs to both the key index and the expiration heap. key, value,
// and deadline are immutable after publication through sync.Map. Only index
// changes under mu; Get never reads it. Set replaces a node when refreshing.
type entry[K comparable, V any] struct {
	key      K
	value    V
	deadline time.Time
	index    int
}

type expiryHeap[K comparable, V any] []*entry[K, V]

func (h expiryHeap[K, V]) Len() int { return len(h) }

func (h expiryHeap[K, V]) Less(i, j int) bool {
	return h[i].deadline.Before(h[j].deadline)
}

func (h expiryHeap[K, V]) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *expiryHeap[K, V]) Push(value any) {
	item := value.(*entry[K, V])
	item.index = len(*h)
	*h = append(*h, item)
}

func (h *expiryHeap[K, V]) Pop() any {
	old := *h
	last := len(old) - 1
	item := old[last]
	old[last] = nil // Do not retain a removed key or value in the backing array.
	item.index = -1
	*h = old[:last]
	return item
}
