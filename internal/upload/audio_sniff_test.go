package upload

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mp4Header(majorBrand string) []byte {
	header := []byte{0, 0, 0, 20}
	header = append(header, []byte("ftyp")...)
	header = append(header, []byte(majorBrand)...)
	header = append(header, 0, 0, 0, 0)
	header = append(header, []byte("mp42")...)

	return header
}

func oggPage(codec string) []byte {
	page := append([]byte("OggS\x00"), bytes.Repeat([]byte{0}, 20)...)

	return append(page, []byte(codec)...)
}

func TestDetectContentType_ClassifiesAudioContainers(t *testing.T) {
	tests := []struct {
		name  string
		bytes []byte
		want  string
	}{
		{name: "mp3 with an id3 tag", bytes: []byte("ID3\x03\x00\x00\x00\x00\x00\x00"), want: "audio/mpeg"},
		{name: "wav", bytes: append(append([]byte("RIFF"), 0, 0, 0, 0), []byte("WAVEfmt ")...), want: "audio/wav"},
		{name: "flac, which go cannot sniff on its own", bytes: append([]byte("fLaC"), bytes.Repeat([]byte{0}, 20)...), want: "audio/flac"},
		{name: "ogg vorbis", bytes: oggPage("vorbis"), want: "audio/ogg"},
		{name: "ogg opus", bytes: oggPage("OpusHead"), want: "audio/ogg"},
		{name: "m4a is an mp4 container but must read as audio", bytes: mp4Header("M4A "), want: "audio/mp4"},
		{name: "m4b audiobook", bytes: mp4Header("M4B "), want: "audio/mp4"},
		{name: "an actual mp4 video stays video", bytes: mp4Header("isom"), want: "video/mp4"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			padded := append(tc.bytes, bytes.Repeat([]byte{0}, 512)...)

			// when
			sniffed, _, err := DetectContentType(bytes.NewReader(padded))

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, NormaliseSniffedType(sniffed))
		})
	}
}

func TestAllowedAudioTypes_DoesNotOverlapVideo(t *testing.T) {
	// given both allowlists
	// when a file is sniffed
	// then its type must never be ambiguous between them
	for mime := range AllowedAudioTypes {
		_, clash := AllowedVideoTypes[mime]
		assert.False(t, clash, "%s appears in both the audio and video allowlists", mime)
	}
}
