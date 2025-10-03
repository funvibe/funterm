package handler

import "go-parser/pkg/common"

func DefaultConfig(name string) common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled:        true,
		Priority:         0,
		Order:            0,
		FallbackPriority: 0,
		IsFallback:       false,
		Name:             name,
	}
}
