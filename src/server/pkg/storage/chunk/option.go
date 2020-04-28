package chunk

import (
	"github.com/pachyderm/pachyderm/src/server/pkg/serviceenv"
	"github.com/pachyderm/pachyderm/src/server/pkg/storage/gc"
)

// StorageOption configures a storage.
type StorageOption func(s *Storage)

// WithGarbageCollection sets the garbage collection client
// for the storage. The storage will use a mock client otherwise.
func WithGarbageCollection(gcClient gc.Client) StorageOption {
	return func(s *Storage) {
		s.gcClient = gcClient
	}
}

// ServiceEnvToOptions converts a service environment configuration (specifically
// the storage configuration) to a set of storage options.
func ServiceEnvToOptions(env *serviceenv.ServiceEnv) []StorageOption {
	return nil
}
