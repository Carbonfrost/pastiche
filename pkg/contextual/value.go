package contextual

import (
	"context"
	"fmt"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/pastiche/pkg/internal/contextkey"
	"github.com/Carbonfrost/pastiche/pkg/workspace"
)

// FromContext gets the given value from the context otherwise panics
func FromContext[T any](ctx context.Context) T {
	var zero T
	return contextkey.Resolve(ctx, keyFor(zero)).(T)
}

// ContextValue provides an action that sets the given value into the context.
// The only supported type is *Workspace.
func ContextValue(k any) cli.Action {
	return cli.ContextValue(keyFor(k), k)
}

// Workspace gets the Workspace from the context otherwise panics
func Workspace(ctx context.Context) *workspace.Workspace {
	return FromContext[*workspace.Workspace](ctx)
}

// With adds the specified values into the context
func With(ctx context.Context, values ...any) context.Context {
	for _, v := range values {
		ctx = context.WithValue(ctx, keyFor(v), v)
	}
	return ctx
}

func keyFor(k any) contextkey.T {
	switch k.(type) {
	case *workspace.Workspace:
		return contextkey.Workspace
	}
	panic(fmt.Errorf("type not supported for context %T", k))
}
