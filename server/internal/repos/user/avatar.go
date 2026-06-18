package user

import (
	"strings"
)

// Default 50×50 initials-style placeholder assigned to new accounts before they upload a photo.
const defaultBlankAvatarDataURL = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAYAAAAeP4ixAAACKElEQVR42u3PzWdcURjH8aFcshpKVlmVEmbb1SWErMLQbRi6Ct32b8gfUIYSwjDm5c57ZqaU0lUoWd1VV2EodzWEMqswDE+/k7TOGK3pebmTe6KHz+JZnd83JyJPwtMJ+Tge+yz1kCIixJj9EiNC0YeQYySQDRIUsxgS4Byi6RxBlkIiiKHISciYw9J7iKWy6f8qZDSycQBx5MBkg4uQADcOQ24QPEbICcSxE+OQEYehK4hjV7o7VMhwaOIFJCUvNXZYh5ymGHK6zZBqiiFVo5Dh5aWJGJKSbxo7VMglh4EEkpIfOltUyGBg4g6SkoXGjkyH3BmFDPp9EwkkJYnGDuuQOMWQeJshlRRDKkYh/V7PRAmSkjcaO6xD9rCAOLbAnlFIj8NQB+JYR3eHCul2TYUQx0LNDZYhStVhRBU545Auh4U8EoilBHmD/1VIp9OxVcAUYmiKgun/KqTddmEfE4imCfYt/nUYouzgTCPiDDvIOQlpc1jaxRHeoQL5RzE+oYy3CLGr+78KabV0BSjiAlOIYzMMUcLzDVuMQl4hwgyyRV/wemNIi2ODEJ8hj+wa4fo+FRJFf5NHH5Ix5ZWNKiTi+IMQ3yEZtXwBVkKazXUlLCAZV4UKaXKsOMIc4olDFdJo/FbADOKRCZ6th3yFeOj4PqRBBA4hnvrwEFKvL11APHW9GjLxOOT2PqROCOYQT81XQ8RnDyG12pJ47H9INkNqhEB8JoT8BNEuPSTVExuQAAAAAElFTkSuQmCC"

// Prefix of the 120×120 Canvas default silhouette JPEG (XMP metadata block).
const canvasDefaultAvatarJPEGBase64Prefix = "4gHYSUNDX1BST0ZJTEUAAQEAAAHIAAAAAAQwAABtbnRyUkdCIFhZWiAH4AABAAEAAAAAAABhY3Nw"

// Marker for the transparent 120×120 blank JPEG used in test fixtures.
const blankAvatarJPEGDimensionMarker = "wAARCAD4AHwD"

// IsMissingOrDefaultBlankAvatarURL reports whether avatar_url is unset or still the
// platform/Canvas placeholder image (not a user-chosen profile photo).
func IsMissingOrDefaultBlankAvatarURL(raw string) bool {
	u := strings.TrimSpace(raw)
	if u == "" {
		return true
	}
	lower := strings.ToLower(u)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return isDefaultBlankAvatarHTTPURL(lower)
	}
	if strings.HasPrefix(lower, "data:image/") {
		return isDefaultBlankAvatarDataURL(u)
	}
	return false
}

func isDefaultBlankAvatarHTTPURL(lower string) bool {
	return strings.Contains(lower, "/images/messages/avatar-")
}

func isDefaultBlankAvatarDataURL(raw string) bool {
	if raw == defaultBlankAvatarDataURL {
		return true
	}
	comma := strings.Index(raw, ",")
	if comma < 0 {
		return false
	}
	payload := raw[comma+1:]
	lowerPayload := strings.ToLower(payload)
	if strings.Contains(payload, canvasDefaultAvatarJPEGBase64Prefix) {
		return true
	}
	if strings.Contains(payload, blankAvatarJPEGDimensionMarker) &&
		strings.Contains(lowerPayload, "/ajraaaa") {
		return true
	}
	return false
}