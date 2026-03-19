package middleware

import "context"

type callerTypeKey struct{}

const (
	CallerTypeService = "service"
	CallerTypeUser    = "user"
)

func withCallerType(ctx context.Context, callerType string) context.Context {
	return context.WithValue(ctx, callerTypeKey{}, callerType)
}

func IsServiceCaller(ctx context.Context) bool {
	v, _ := ctx.Value(callerTypeKey{}).(string)
	return v == CallerTypeService
}
