package tools

import "strings"

type RegistryType int

const (
	NormalRegistry RegistryType = iota
	AwsRegistry
)

// RegistryDistinguisher returns registry type, adding help for specific service like aws ecr
func RegistryDistinguisher(registry string) RegistryType {
	// judge whether registry is an aws registry
	if strings.Contains(registry, "amazonaws.com") {
		return AwsRegistry
	}
	return NormalRegistry
}
