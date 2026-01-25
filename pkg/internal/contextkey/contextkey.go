package contextkey

import (
	"context"
	"fmt"
)

type keyType string

type T = keyType

const (
	Workspace keyType = "workspace"
)

func Resolve(ctx context.Context, key keyType) any {
	res, err := tryResolve(ctx, key)
	if err != nil {
		panic(err)
	}
	return res
}

func tryResolve(ctx context.Context, key keyType) (any, error) {
	res := ctx.Value(key)
	if res == nil {
		return nil, fmt.Errorf("expected %v value not present in context", key)
	}
	return res, nil
}
