package util

import (
	"net/url"
	"os"
)

func TypesenseURLFromEnv() (string, bool) {
	if typesenseURLFromEnv, found := os.LookupEnv("TYPESENSE_URL"); found && typesenseURLFromEnv != "" {
		return typesenseURLFromEnv, true
	}

	return TypesenseURLFromEnvComponents()
}

func TypesenseURLFromEnvComponents() (string, bool) {
	foundHost := false

	var typesenseURLFromEnvComponents url.URL

	if typesenseProtocolFromEnv, found := os.LookupEnv("TYPESENSE_PROTOCOL"); found && typesenseProtocolFromEnv != "" {
		typesenseURLFromEnvComponents.Scheme = typesenseProtocolFromEnv
	} else {
		typesenseURLFromEnvComponents.Scheme = "http"
	}

	if typesenseHostFromEnv, found := os.LookupEnv("TYPESENSE_HOST"); found && typesenseHostFromEnv != "" {
		foundHost = true
		typesenseURLFromEnvComponents.Host = typesenseHostFromEnv
	}

	if typesensePortFromEnv, found := os.LookupEnv("TYPESENSE_PORT"); found && typesensePortFromEnv != "" {
		typesenseURLFromEnvComponents.Host += ":" + typesensePortFromEnv
	}

	if !foundHost {
		return "", false
	}

	return typesenseURLFromEnvComponents.String(), true
}
