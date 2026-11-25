package provider

import "time"

type TypesenseProviderData struct {
	api   typesenseAPI
	alter chan struct{}
}

const (
	defaultOperationTimeout        = 5 * time.Minute
	defaultCollectionUpdateTimeout = 30 * time.Minute
)
