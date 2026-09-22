package provider

import (
	"errors"
	"net/http"
	"time"

	"github.com/typesense/typesense-go/v3/typesense"
)

type TypesenseProviderData struct {
	client *typesense.Client
	keys   keyAPI
	alter  chan struct{}
}

const (
	defaultOperationTimeout        = 5 * time.Minute
	defaultCollectionUpdateTimeout = 30 * time.Minute
)

func typesenseNotFound(err error) bool {
	response, ok := errors.AsType[*typesense.HTTPError](err)

	return ok && response.Status == http.StatusNotFound
}
