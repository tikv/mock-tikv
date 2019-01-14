package api

// Peer contains id of backing store
type Peer struct {
	ID      uint64 `json:"id"`
	StoreID uint64 `json:"store_id"`
}
