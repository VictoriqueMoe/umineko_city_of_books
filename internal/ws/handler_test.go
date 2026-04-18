package ws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOriginAllowed(t *testing.T) {
	cases := []struct {
		name    string
		origin  string
		allowed string
		want    bool
	}{
		{"exact match", "https://meta.auaurora.moe", "https://meta.auaurora.moe", true},
		{"localhost dev", "http://localhost:4323", "http://localhost:4323", true},
		{"different host", "https://evil.example.com", "https://meta.auaurora.moe", false},
		{"different scheme", "http://meta.auaurora.moe", "https://meta.auaurora.moe", false},
		{"different port", "https://meta.auaurora.moe:8080", "https://meta.auaurora.moe", false},
		{"browser with trailing slash is still rejected", "https://meta.auaurora.moe/", "https://meta.auaurora.moe", false},
		{"allowed with trailing slash is tolerated", "https://meta.auaurora.moe", "https://meta.auaurora.moe/", true},
		{"allowed with trailing slash, dev", "http://localhost:5173", "http://localhost:5173/", true},
		{"empty origin rejected", "", "https://meta.auaurora.moe", false},
		{"empty allowed rejected", "https://meta.auaurora.moe", "", false},
		{"both empty rejected", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, originAllowed(tc.origin, tc.allowed))
		})
	}
}
