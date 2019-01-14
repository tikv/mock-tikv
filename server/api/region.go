package api

// RegionEpoch records epoch data of a region
type RegionEpoch struct {
	ConfVer uint64 `json:"conf_ver"`
	Version uint64 `json:"version"`
}

// Region records region setup for api usage.
type Region struct {
	ID          uint64       `json:"id"`
	StartKey    []byte       `json:"start_key"`
	EndKey      []byte       `json:"end_key"`
	RegionEpoch *RegionEpoch `json:"region_epoch"`
	Peers       []*Peer      `json:"peers"`
}
