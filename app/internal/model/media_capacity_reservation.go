package model

import "time"

const (
	MediaReservationPending   = "pending"
	MediaReservationConfirmed = "confirmed"
)

// MediaCapacityReservation covers the gap between admission and engine metrics.
type MediaCapacityReservation struct {
	BaseModel
	StreamKey    string     `gorm:"type:varchar(200);not null;uniqueIndex" json:"stream_key"`
	NodeID       string     `gorm:"type:uuid;not null;index" json:"node_id"`
	DecodeSlots  uint32     `gorm:"not null;default:0" json:"decode_slots"`
	EncodeSlots  uint32     `gorm:"not null;default:0" json:"encode_slots"`
	EgressBPS    uint64     `gorm:"not null;default:0" json:"egress_bps"`
	Status       string     `gorm:"type:varchar(16);not null;default:pending;index;check:status IN ('pending','confirmed')" json:"status"`
	LeaseExpires *time.Time `gorm:"index" json:"lease_expires,omitempty"`
}

func (MediaCapacityReservation) TableName() string {
	return "media_capacity_reservations"
}
