package api

// StoreState contains state about a store.
type StoreState string

const (
	// StoreStateUp store is up and running
	StoreStateUp = "Up"
	// StoreStateOffline store is offline
	StoreStateOffline = "Offline"
	// StoreStateTombstone store is set to tombstone state
	StoreStateTombstone = "Tombstone"
)

// Store contains information about a store
type Store struct {
	ID      uint64     `json:"id"`
	Address string     `json:"address"`
	State   StoreState `json:"state"`
	Version string     `json:"version"`
}
