package middleware

import (
	"context"
	"crypto/subtle"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
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

		peerAddr := grpcPeerAddr(ctx)

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			glog.Warningf("gRPC auth failure: missing metadata for %s from %s", info.FullMethod, peerAddr)
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			glog.Warningf("gRPC auth failure: missing authorization header for %s from %s", info.FullMethod, peerAddr)
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		token, err := extractBearerToken(authHeader[0])
		if err != nil {
			glog.Warningf("gRPC auth failure: %v for %s from %s", err, info.FullMethod, peerAddr)
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
			glog.Warningf("gRPC auth failure: invalid bearer token for %s from %s", info.FullMethod, peerAddr)
			return nil, status.Error(codes.Unauthenticated, "invalid bearer token")
		}

		return handler(ctx, req)
	}
}

func bearerTokenGRPCStreamInterceptor(expectedToken string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if grpcBypassMethods[info.FullMethod] {
			return handler(srv, ss)
		}

		peerAddr := grpcPeerAddr(ss.Context())

		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			glog.Warningf("gRPC stream auth failure: missing metadata for %s from %s", info.FullMethod, peerAddr)
			return status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			glog.Warningf("gRPC stream auth failure: missing authorization header for %s from %s", info.FullMethod, peerAddr)
			return status.Error(codes.Unauthenticated, "missing authorization header")
		}

		token, err := extractBearerToken(authHeader[0])
		if err != nil {
			glog.Warningf("gRPC stream auth failure: %v for %s from %s", err, info.FullMethod, peerAddr)
			return status.Error(codes.Unauthenticated, err.Error())
		}

		if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
			glog.Warningf("gRPC stream auth failure: invalid bearer token for %s from %s", info.FullMethod, peerAddr)
			return status.Error(codes.Unauthenticated, "invalid bearer token")
		}

		return handler(srv, ss)
	}
}

func grpcPeerAddr(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}
	return "unknown"
}
