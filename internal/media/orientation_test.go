package media

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ifdEntry struct {
	tag      uint16
	dataType uint16
	count    uint32
	value    uint16
}

func tiffBlock(order binary.ByteOrder, entries ...ifdEntry) []byte {
	var b bytes.Buffer
	if order == binary.ByteOrder(binary.LittleEndian) {
		b.WriteString("II")
	} else {
		b.WriteString("MM")
	}

	_ = binary.Write(&b, order, uint16(42))
	_ = binary.Write(&b, order, uint32(tiffHeaderSize))
	_ = binary.Write(&b, order, uint16(len(entries)))

	for _, entry := range entries {
		_ = binary.Write(&b, order, entry.tag)
		_ = binary.Write(&b, order, entry.dataType)
		_ = binary.Write(&b, order, entry.count)
		_ = binary.Write(&b, order, entry.value)
		_ = binary.Write(&b, order, uint16(0))
	}

	return b.Bytes()
}

func jpegSegment(marker uint16, payload []byte) []byte {
	var b bytes.Buffer
	_ = binary.Write(&b, binary.BigEndian, marker)
	_ = binary.Write(&b, binary.BigEndian, uint16(len(payload)+2))
	b.Write(payload)

	return b.Bytes()
}

func exifPayload(tiff []byte) []byte {
	return append([]byte(exifPreamble), tiff...)
}

func jpegOf(segments ...[]byte) []byte {
	out := []byte{0xff, 0xd8}
	for _, segment := range segments {
		out = append(out, segment...)
	}

	return append(out, 0xff, 0xda)
}

func TestReadJPEGOrientation(t *testing.T) {
	rotated := ifdEntry{tag: orientationTag, dataType: 3, count: 1, value: 6}
	other := ifdEntry{tag: 0x010f, dataType: 2, count: 1, value: 0}
	countBomb := ifdEntry{tag: 0x829a, dataType: 5, count: 0xe0000001, value: 0}

	tests := []struct {
		name string
		data []byte
		want int
	}{
		{name: "little endian", data: jpegOf(jpegSegment(jpegMarkerAPP1, exifPayload(tiffBlock(binary.LittleEndian, rotated)))), want: 6},
		{name: "big endian", data: jpegOf(jpegSegment(jpegMarkerAPP1, exifPayload(tiffBlock(binary.BigEndian, rotated)))), want: 6},
		{name: "orientation after another tag", data: jpegOf(jpegSegment(jpegMarkerAPP1, exifPayload(tiffBlock(binary.LittleEndian, other, rotated)))), want: 6},
		{name: "a tag whose count would overflow is skipped without allocating", data: jpegOf(jpegSegment(jpegMarkerAPP1, exifPayload(tiffBlock(binary.LittleEndian, countBomb, rotated)))), want: 6},
		{name: "other segments come first", data: jpegOf(jpegSegment(0xffe0, []byte("JFIF\x00")), jpegSegment(jpegMarkerAPP1, exifPayload(tiffBlock(binary.LittleEndian, rotated)))), want: 6},
		{name: "xmp app1 before exif app1", data: jpegOf(jpegSegment(jpegMarkerAPP1, []byte("http://ns.adobe.com/xap/1.0/\x00")), jpegSegment(jpegMarkerAPP1, exifPayload(tiffBlock(binary.LittleEndian, rotated)))), want: 6},
		{name: "exif without an orientation tag", data: jpegOf(jpegSegment(jpegMarkerAPP1, exifPayload(tiffBlock(binary.LittleEndian, other)))), want: 0},
		{name: "no exif at all", data: jpegOf(jpegSegment(0xffe0, []byte("JFIF\x00"))), want: 0},
		{name: "an orientation value outside the spec is ignored", data: jpegOf(jpegSegment(jpegMarkerAPP1, exifPayload(tiffBlock(binary.LittleEndian, ifdEntry{tag: orientationTag, dataType: 3, count: 1, value: 9})))), want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			r := bufio.NewReader(bytes.NewReader(tc.data))

			// when
			got, err := readJPEGOrientation(r)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestReadJPEGOrientation_MalformedDataIsAnError(t *testing.T) {
	truncatedDirectory := exifPayload(tiffBlock(binary.LittleEndian, ifdEntry{tag: orientationTag, dataType: 3, count: 1, value: 6}))
	truncatedDirectory = truncatedDirectory[:len(truncatedDirectory)-4]

	offsetPastEnd := tiffBlock(binary.LittleEndian)
	binary.LittleEndian.PutUint32(offsetPastEnd[4:8], 0xffffffff)

	tests := []struct {
		name string
		data []byte
	}{
		{name: "not a jpeg", data: []byte("RIFFxxxxWEBP")},
		{name: "a marker that is not a marker", data: []byte{0xff, 0xd8, 0x00, 0x00}},
		{name: "a segment size below its own header", data: []byte{0xff, 0xd8, 0xff, 0xe1, 0x00, 0x01}},
		{name: "a tiff block shorter than its header", data: jpegOf(jpegSegment(jpegMarkerAPP1, exifPayload([]byte("II"))))},
		{name: "an unknown byte order", data: jpegOf(jpegSegment(jpegMarkerAPP1, exifPayload([]byte("XX*\x00\x08\x00\x00\x00\x00\x00"))))},
		{name: "a directory offset past the end", data: jpegOf(jpegSegment(jpegMarkerAPP1, exifPayload(offsetPastEnd)))},
		{name: "a directory cut short", data: jpegOf(jpegSegment(jpegMarkerAPP1, truncatedDirectory))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			r := bufio.NewReader(bytes.NewReader(tc.data))

			// when
			_, err := readJPEGOrientation(r)

			// then
			assert.ErrorIs(t, err, errMalformedExif)
		})
	}
}

func TestReadJPEGOrientation_ATruncatedFileIsAnError(t *testing.T) {
	// given a file that ends inside its app1 segment
	r := bufio.NewReader(bytes.NewReader([]byte{0xff, 0xd8, 0xff, 0xe1, 0x00, 0x20, 'E'}))

	// when
	_, err := readJPEGOrientation(r)

	// then
	assert.Error(t, err)
}
