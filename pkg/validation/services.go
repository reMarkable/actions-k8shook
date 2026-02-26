// Package validation provides validation functions for service definitions
package validation

import (
	"errors"
	"fmt"

	"github.com/reMarkable/k8s-hook/pkg/types"
)

var (
	ErrEmptyImage           = errors.New("service image cannot be empty")
	ErrDuplicateServiceName = errors.New("duplicate service name")
	ErrInvalidServiceName   = errors.New("service image must be a valid DNS label")
	ErrReservedServiceName  = errors.New("service name cannot be 'job'")
)

// ValidateServices validates a slice of service definitions and returns an error if validation fails.
// It checks for:
// - Image required
// - Duplicate service names
// - Invalid service names (must be valid Kubernetes container names)
// - Reserved names (e.g., "job")
func ValidateServices(services []types.ServiceDefinition) error {
	if len(services) == 0 {
		return nil
	}

	seenNames := make(map[string]bool)

	for i, service := range services {

		if service.Image == "" {
			return fmt.Errorf("service[%d] (%s): %w", i, service.ContextName, ErrEmptyImage)
		}

		if service.ContextName == "job" {
			return fmt.Errorf("service[%d]: %w: 'job'", i, ErrReservedServiceName)
		}

		if !isValidDNSLabel(service.ContextName) {
			return fmt.Errorf("service[%d]: %w: '%s' (must contain only lowercase alphanumeric characters or '-', start with alphanumeric, and be at most 63 characters)",
				i, ErrInvalidServiceName, service.ContextName)
		}

		if seenNames[service.ContextName] {
			return fmt.Errorf("service[%d]: %w: '%s'", i, ErrDuplicateServiceName, service.ContextName)
		}
		seenNames[service.ContextName] = true
	}

	return nil
}

// isValidDNSLabel checks if a string is a valid DNS label (RFC 1123)
// Valid labels:
// - contain only lowercase alphanumeric characters or '-'
// - start with an alphanumeric character
// - end with an alphanumeric character
// - be at most 63 characters
func isValidDNSLabel(name string) bool {
	if len(name) == 0 || len(name) > 63 {
		return false
	}

	for i, ch := range name {
		if (i == 0 || i == len(name)-1) && ch == '-' {
			return false
		}
		if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' {
			return false
		}
	}

	return true
}
