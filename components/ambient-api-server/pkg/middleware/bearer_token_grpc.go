package middleware

import (
	"context"
	"crypto/subtle"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var grpcBypassMethods = map[string]bool{
	"/grpc.health.v1.Health/Check":                                   true,
	"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo": true,
}

func bearerTokenGRPCUnaryInterceptor(expectedToken string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if grpcBypassMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if authHeader := md.Get("authorization"); len(authHeader) > 0 {
				if token, err := extractBearerToken(authHeader[0]); err == nil {
					if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) == 1 {
						return handler(withCallerType(ctx, CallerTypeService), req)
					}
				}
			}
		}

		return handler(ctx, req)
	}
}

func bearerTokenGRPCStreamInterceptor(expectedToken string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if grpcBypassMethods[info.FullMethod] {
			return handler(srv, ss)
		}

		if md, ok := metadata.FromIncomingContext(ss.Context()); ok {
			if authHeader := md.Get("authorization"); len(authHeader) > 0 {
				if token, err := extractBearerToken(authHeader[0]); err == nil {
					if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) == 1 {
						return handler(srv, &serviceCallerStream{ServerStream: ss, ctx: withCallerType(ss.Context(), CallerTypeService)})
					}
				}
			}
		}

		return handler(srv, ss)
	}
}

type serviceCallerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serviceCallerStream) Context() context.Context {
	return s.ctx
}
