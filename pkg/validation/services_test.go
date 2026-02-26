package validation

import (
	"errors"
	"testing"

	"github.com/reMarkable/k8s-hook/pkg/types"
)

func TestValidateServices(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		services []types.ServiceDefinition
		wantErr  error
	}{
		"empty services list is valid": {
			services: []types.ServiceDefinition{},
			wantErr:  nil,
		},
		"nil services list is valid": {
			services: nil,
			wantErr:  nil,
		},
		"valid single service": {
			services: []types.ServiceDefinition{
				{
					ContextName: "redis",
					Image:       "redis:7",
				},
			},
			wantErr: nil,
		},
		"valid multiple services": {
			services: []types.ServiceDefinition{
				{
					ContextName: "redis",
					Image:       "redis:7",
				},
				{
					ContextName: "postgres",
					Image:       "postgres:14",
				},
			},
			wantErr: nil,
		},
		"missing image": {
			services: []types.ServiceDefinition{
				{
					ContextName: "redis",
					Image:       "",
				},
			},
			wantErr: ErrEmptyImage,
		},
		"duplicate service names": {
			services: []types.ServiceDefinition{
				{
					ContextName: "redis",
					Image:       "redis:7",
				},
				{
					ContextName: "redis",
					Image:       "redis:6",
				},
			},
			wantErr: ErrDuplicateServiceName,
		},
		"reserved name 'job'": {
			services: []types.ServiceDefinition{
				{
					ContextName: "job",
					Image:       "redis:7",
				},
			},
			wantErr: ErrReservedServiceName,
		},
		"invalid name with uppercase": {
			services: []types.ServiceDefinition{
				{
					ContextName: "Redis",
					Image:       "redis:7",
				},
			},
			wantErr: ErrInvalidServiceName,
		},
		"invalid name with special characters": {
			services: []types.ServiceDefinition{
				{
					ContextName: "redis_cache",
					Image:       "redis:7",
				},
			},
			wantErr: ErrInvalidServiceName,
		},
		"invalid name starting with hyphen": {
			services: []types.ServiceDefinition{
				{
					ContextName: "-redis",
					Image:       "redis:7",
				},
			},
			wantErr: ErrInvalidServiceName,
		},
		"invalid name ending with hyphen": {
			services: []types.ServiceDefinition{
				{
					ContextName: "redis-",
					Image:       "redis:7",
				},
			},
			wantErr: ErrInvalidServiceName,
		},
		"valid name with hyphens": {
			services: []types.ServiceDefinition{
				{
					ContextName: "redis-cache-001",
					Image:       "redis:7",
				},
			},
			wantErr: nil,
		},
		"name too long": {
			services: []types.ServiceDefinition{
				{
					ContextName: "a123456789012345678901234567890123456789012345678901234567890123456789",
					Image:       "redis:7",
				},
			},
			wantErr: ErrInvalidServiceName,
		},
		"name exactly 63 characters is valid": {
			services: []types.ServiceDefinition{
				{
					ContextName: "a12345678901234567890123456789012345678901234567890123456789012",
					Image:       "redis:7",
				},
			},
			wantErr: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := ValidateServices(tt.services)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateServices() error = %v, wantErr nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateServices() error = nil, wantErr %v", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateServices() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestIsValidDNSLabel(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		label string
		want  bool
	}{
		"valid simple":               {"redis", true},
		"valid with numbers":         {"redis123", true},
		"valid with hyphens":         {"redis-cache", true},
		"valid starting with number": {"7redis", true},
		"empty string":               {"", false},
		"uppercase":                  {"Redis", false},
		"underscore":                 {"redis_cache", false},
		"starting with hyphen":       {"-redis", false},
		"ending with hyphen":         {"redis-", false},
		"too long":                   {"a123456789012345678901234567890123456789012345678901234567890123456789", false},
		"exactly 63 chars":           {"a12345678901234567890123456789012345678901234567890123456789012", true},
		"special char":               {"redis@cache", false},
		"dot":                        {"redis.cache", false},
		"space":                      {"redis cache", false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := isValidDNSLabel(tt.label)
			if got != tt.want {
				t.Errorf("isValidDNSLabel(%q) = %v, want %v", tt.label, got, tt.want)
			}
		})
	}
}
