package api

import (
	trexapi "github.com/openshift-online/rh-trex-ai/pkg/api"
)

type Meta = trexapi.Meta
type EventType = trexapi.EventType

const (
	CreateEventType = trexapi.CreateEventType
	UpdateEventType = trexapi.UpdateEventType
	DeleteEventType = trexapi.DeleteEventType
)

var NewID = trexapi.NewID
