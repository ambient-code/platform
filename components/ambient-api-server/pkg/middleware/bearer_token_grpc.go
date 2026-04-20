package middleware

import (
	"context"
	"crypto/subtle"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/openshift-online/rh-trex-ai/pkg/auth"
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
					if username := usernameFromJWT(token); username != "" {
						return handler(auth.SetUsernameContext(ctx, username), req)
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
					if username := usernameFromJWT(token); username != "" {
						ctx := auth.SetUsernameContext(ss.Context(), username)
						return handler(srv, &serviceCallerStream{ServerStream: ss, ctx: ctx})
					}
				}
			}
		}

		return handler(srv, ss)
	}
}

// usernameFromJWT extracts the username claim without signature verification.
// This is safe because this interceptor runs in the pre-auth chain; the
// downstream AuthUnaryInterceptor / AuthStreamInterceptor performs full
// JWT signature verification via JWKKeyProvider.KeyFunc. If the token is
// invalid, the downstream interceptor will reject the request regardless
// of the username set here.
func usernameFromJWT(tokenString string) string {
	p := jwt.NewParser()
	token, _, err := p.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return ""
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ""
	}
	for _, key := range []string{"preferred_username", "username", "sub"} {
		if v, _ := claims[key].(string); v != "" && !strings.Contains(v, ":") {
			return v
		}
	}
	return ""
}

type serviceCallerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serviceCallerStream) Context() context.Context {
	return s.ctx
}
