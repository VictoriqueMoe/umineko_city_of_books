package media

import (
	"bytes"
	"context"
	"encoding/hex"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tinyWebPHex     = "524946463c000000574542505650382030000000d001009d012a0800080002003425a00274ba01f80003b000fef0e8f7ff20b96175c8d7ff203fe407fc80fff8f2000000"
	tinyAnimWebPHex = "524946463601000057454250565038580a00000002000000070000070000414e494d06000000ffffffff0000414e4d469c000000000000000000070000070000c800000256503820840000009002009d012a0800080002003425b0027432803a400a54a5cad743166000fef1bc7e0ccdbe4f525f62f1262cf4bb050ff98b35f4d78fedcafa4fc4f2e089f2f0a7404bf0f52757b3d94f023277a1dc013e54a3ffc76763b0a42641fbeff95156d1fffca070ff950ff4e4496b1fc4efdef88454061793fdef3df143d6ddcff9a19f480000414e4d4666000000000000030000070000010000c8000000565038204e0000007402009d012a0800020000003425a802749200f200c900050eee4eb800fee25e27e07f5bfe4b0dbebdf80008ffe506c3ff950fffb267fe506c3ff950fde5da5cce4fffee04fb673393fd48000000"
)

func writeFixture(t *testing.T, hexData string) string {
	webpBytes, err := hex.DecodeString(hexData)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "img.webp")
	require.NoError(t, os.WriteFile(path, webpBytes, 0644))
	return path
}

func TestWebPToJPEG(t *testing.T) {
	tests := []struct {
		name            string
		hex             string
		requiresWebpmux bool
	}{
		{name: "static webp", hex: tinyWebPHex},
		{name: "animated webp via first frame", hex: tinyAnimWebPHex, requiresWebpmux: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.requiresWebpmux {
				if _, err := exec.LookPath("webpmux"); err != nil {
					t.Skip("webpmux not installed")
				}
			}

			// given
			path := writeFixture(t, tc.hex)

			// when
			out, err := WebPToJPEG(context.Background(), path)

			// then
			require.NoError(t, err)
			cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
			require.NoError(t, err)
			assert.Equal(t, 8, cfg.Width)
			assert.Equal(t, 8, cfg.Height)
		})
	}
}

func TestWebPToJPEG_MissingFile(t *testing.T) {
	// given
	path := filepath.Join(t.TempDir(), "missing.webp")

	// when
	_, err := WebPToJPEG(context.Background(), path)

	// then
	require.Error(t, err)
}
