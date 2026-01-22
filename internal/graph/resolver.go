package graph

import (
	"github.com/hmans/beans/internal/beancore"
)

//go:generate go tool gqlgen generate

// Resolver is the root resolver for the GraphQL schema.
// It holds a reference to beancore.Core for data access.
type Resolver struct {
	Core *beancore.Core
}
