package dto

import (
	"encoding/json"

	"anoa.com/telkomalumiforum/internal/entity"
)

// CSSPayload is the Cosmetic.Payload shape when RenderType == "css".
// PresetKey must match a ring preset registered in FE code — admins don't
// submit free-form CSS.
type CSSPayload struct {
	PresetKey string `json:"preset_key" binding:"required"`
}

// ImagePayload is the Cosmetic.Payload shape when RenderType == "image".
// StaticURL is used when prefers-reduced-motion is set.
type ImagePayload struct {
	AnimatedURL string `json:"animated_url" binding:"required,url"`
	StaticURL   string `json:"static_url" binding:"required,url"`
}

type CosmeticResponse struct {
	ID         uint        `json:"id"`
	Slot       string      `json:"slot"`
	SubType    string      `json:"sub_type,omitempty"`
	RenderType string      `json:"render_type"`
	Name       string      `json:"name"`
	Payload    interface{} `json:"payload"`
	Price      int         `json:"price"`
	MinRank    string      `json:"min_rank,omitempty"`
	Status     string      `json:"status"`
}

type EquipResponse struct {
	AvatarBorder *CosmeticResponse `json:"avatar_border"`
	ThreadBg     *CosmeticResponse `json:"thread_bg"`
	ProfileBg    *CosmeticResponse `json:"profile_bg"`
}

type InventoryItemResponse struct {
	Cosmetic    CosmeticResponse `json:"cosmetic"`
	PurchasedAt string           `json:"purchased_at"`
}

type EquipRequest struct {
	Slot       string `json:"slot" binding:"required,oneof=avatar_border thread_bg profile_bg"`
	CosmeticID *uint  `json:"cosmetic_id"`
}

type BatchCosmeticsRequest struct {
	Usernames []string `json:"usernames" binding:"required,min=1,max=100"`
}

type AdminCosmeticInput struct {
	Slot       string `form:"slot" binding:"required,oneof=avatar_border thread_bg profile_bg"`
	SubType    string `form:"sub_type" binding:"omitempty,oneof=ring decoration"`
	RenderType string `form:"render_type" binding:"required,oneof=css image"`
	Name       string `form:"name" binding:"required"`
	Price      int    `form:"price" binding:"required,min=0"`
	MinRank    string `form:"min_rank"`
	Status     string `form:"status" binding:"omitempty,oneof=draft published retired"`
	PresetKey  string `form:"preset_key"` // required when render_type=css
}

// FromCosmeticEntity converts a Cosmetic entity to its API response shape,
// decoding Payload into the concrete CSSPayload/ImagePayload per RenderType.
// Shared by the cosmetic module's own responses and by every other module
// (thread/post/leaderboard) that embeds equipped cosmetics on an author.
func FromCosmeticEntity(c *entity.Cosmetic) CosmeticResponse {
	resp := CosmeticResponse{
		ID:         c.ID,
		Slot:       c.Slot,
		SubType:    c.SubType,
		RenderType: c.RenderType,
		Name:       c.Name,
		Price:      c.Price,
		MinRank:    c.MinRank,
		Status:     c.Status,
	}
	switch c.RenderType {
	case "css":
		var p CSSPayload
		if err := json.Unmarshal(c.Payload, &p); err == nil {
			resp.Payload = p
		}
	case "image":
		var p ImagePayload
		if err := json.Unmarshal(c.Payload, &p); err == nil {
			resp.Payload = p
		}
	}
	return resp
}

// FromEquipEntity converts a UserEquip entity to its API response shape.
// Returns nil for a nil input so callers can embed it as an omitempty
// pointer field without a nil-check at every call site.
func FromEquipEntity(equip *entity.UserEquip) *EquipResponse {
	if equip == nil {
		return nil
	}
	resp := &EquipResponse{}
	if equip.AvatarBorder != nil {
		r := FromCosmeticEntity(equip.AvatarBorder)
		resp.AvatarBorder = &r
	}
	if equip.ThreadBg != nil {
		r := FromCosmeticEntity(equip.ThreadBg)
		resp.ThreadBg = &r
	}
	if equip.ProfileBg != nil {
		r := FromCosmeticEntity(equip.ProfileBg)
		resp.ProfileBg = &r
	}
	return resp
}
