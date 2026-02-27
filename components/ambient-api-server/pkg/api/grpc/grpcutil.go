package grpc

import (
	pb "github.com/ambient/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
)

func APIEventTypeToProto(et api.EventType) pb.EventType {
	switch et {
	case api.CreateEventType:
		return pb.EventType_EVENT_TYPE_CREATED
	case api.UpdateEventType:
		return pb.EventType_EVENT_TYPE_UPDATED
	case api.DeleteEventType:
		return pb.EventType_EVENT_TYPE_DELETED
	default:
		return pb.EventType_EVENT_TYPE_UNSPECIFIED
	}
}
