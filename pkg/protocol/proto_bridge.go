package protocol

import (
	pb "github.com/cyper98/zenc-protocol/generated/go"
	"time"
)

// ToProto converts a domain Room to a protobuf Room message
func (r *Room) ToProto() *pb.Room {
	if r == nil {
		return nil
	}
	meta := make(map[string]string)
	for k, v := range r.Metadata {
		if str, ok := v.(string); ok {
			meta[k] = str
		}
	}
	return &pb.Room{
		Id:        r.ID,
		Name:      r.Name,
		MaxUsers:  int32(r.MaxUsers),
		CreatedAt: r.CreatedAt.Unix(),
		Zone:      r.Zone,
		Metadata:  meta,
	}
}

// RoomFromProto converts a protobuf Room message to a domain Room
func RoomFromProto(p *pb.Room) *Room {
	if p == nil {
		return nil
	}
	meta := make(map[string]interface{})
	for k, v := range p.Metadata {
		meta[k] = v
	}
	return &Room{
		ID:        p.Id,
		Name:      p.Name,
		MaxUsers:  int(p.MaxUsers),
		CreatedAt: time.Unix(p.CreatedAt, 0),
		Zone:      p.Zone,
		Metadata:  meta,
	}
}

// ToProto converts a domain Participant to a protobuf Participant message
func (p *Participant) ToProto() *pb.Participant {
	if p == nil {
		return nil
	}
	return &pb.Participant{
		Id:       p.ID,
		RoomId:   p.RoomID,
		UserId:   p.UserID,
		Role:     p.Role.ToProto(),
		JoinedAt: p.JoinedAt.Unix(),
	}
}

// ToProto converts a domain RoomRole to protobuf RoomRole enum
func (r RoomRole) ToProto() pb.RoomRole {
	switch r {
	case RolePublisher:
		return pb.RoomRole_ROOM_ROLE_PUBLISHER
	case RoleSubscriber:
		return pb.RoomRole_ROOM_ROLE_SUBSCRIBER
	case RoleModerator:
		return pb.RoomRole_ROOM_ROLE_MODERATOR
	default:
		return pb.RoomRole_ROOM_ROLE_UNSPECIFIED
	}
}

// RoomRoleFromProto converts a protobuf RoomRole enum to domain RoomRole
func RoomRoleFromProto(r pb.RoomRole) RoomRole {
	switch r {
	case pb.RoomRole_ROOM_ROLE_PUBLISHER:
		return RolePublisher
	case pb.RoomRole_ROOM_ROLE_SUBSCRIBER:
		return RoleSubscriber
	case pb.RoomRole_ROOM_ROLE_MODERATOR:
		return RoleModerator
	default:
		return RoleSubscriber
	}
}

// ToProto converts a domain Track to a protobuf Track message
func (t *Track) ToProto() *pb.Track {
	if t == nil {
		return nil
	}
	return &pb.Track{
		Id:     t.ID,
		Kind:   t.Kind.ToProto(),
		Source: t.Source.ToProto(),
		Status: t.Status.ToProto(),
	}
}

// ToProto converts a domain TrackKind to protobuf TrackKind enum
func (k TrackKind) ToProto() pb.TrackKind {
	switch k {
	case TrackKindAudio:
		return pb.TrackKind_TRACK_KIND_AUDIO
	case TrackKindVideo:
		return pb.TrackKind_TRACK_KIND_VIDEO
	default:
		return pb.TrackKind_TRACK_KIND_UNSPECIFIED
	}
}

// ToProto converts a domain TrackSource to protobuf TrackSource enum
func (s TrackSource) ToProto() pb.TrackSource {
	switch s {
	case TrackSourceCamera:
		return pb.TrackSource_TRACK_SOURCE_CAMERA
	case TrackSourceScreen:
		return pb.TrackSource_TRACK_SOURCE_SCREEN
	case TrackSourceMicrophone:
		return pb.TrackSource_TRACK_SOURCE_MICROPHONE
	default:
		return pb.TrackSource_TRACK_SOURCE_UNSPECIFIED
	}
}

// ToProto converts a domain TrackStatus to protobuf TrackStatus enum
func (s TrackStatus) ToProto() pb.TrackStatus {
	switch s {
	case TrackStatusActive:
		return pb.TrackStatus_TRACK_STATUS_ACTIVE
	case TrackStatusPaused:
		return pb.TrackStatus_TRACK_STATUS_PAUSED
	case TrackStatusStopped:
		return pb.TrackStatus_TRACK_STATUS_STOPPED
	default:
		return pb.TrackStatus_TRACK_STATUS_UNSPECIFIED
	}
}

