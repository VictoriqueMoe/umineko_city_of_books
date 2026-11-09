package media

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	jpegMarkerSOI  = 0xffd8
	jpegMarkerEOI  = 0xffd9
	jpegMarkerSOS  = 0xffda
	jpegMarkerAPP1 = 0xffe1

	exifPreamble   = "Exif\x00\x00"
	tiffHeaderSize = 8
	ifdEntrySize   = 12
	orientationTag = 0x0112
	maxOrientation = 8
)

func readJPEGOrientation(r *bufio.Reader) (int, error) {
	tiff, err := exifSegment(r)
	if err != nil {
		return 0, err
	}

	if tiff == nil {
		return 0, nil
	}

	return exifOrientation(tiff)
}

func exifSegment(r *bufio.Reader) ([]byte, error) {
	var start uint16
	if err := binary.Read(r, binary.BigEndian, &start); err != nil {
		return nil, fmt.Errorf("read jpeg start: %w", err)
	}

	if start != jpegMarkerSOI {
		return nil, errMalformedExif
	}

	for {
		var marker uint16
		if err := binary.Read(r, binary.BigEndian, &marker); err != nil {
			return nil, fmt.Errorf("read jpeg marker: %w", err)
		}

		if marker>>8 != 0xff {
			return nil, errMalformedExif
		}

		if marker == jpegMarkerSOS || marker == jpegMarkerEOI {
			return nil, nil
		}

		var size uint16
		if err := binary.Read(r, binary.BigEndian, &size); err != nil {
			return nil, fmt.Errorf("read jpeg segment size: %w", err)
		}

		if size < 2 {
			return nil, errMalformedExif
		}

		if marker != jpegMarkerAPP1 {
			if _, err := r.Discard(int(size) - 2); err != nil {
				return nil, fmt.Errorf("skip jpeg segment: %w", err)
			}

			continue
		}

		segment := make([]byte, int(size)-2)
		if _, err := io.ReadFull(r, segment); err != nil {
			return nil, fmt.Errorf("read jpeg app1 segment: %w", err)
		}

		if tiff, found := bytes.CutPrefix(segment, []byte(exifPreamble)); found {
			return tiff, nil
		}
	}
}

func exifOrientation(tiff []byte) (int, error) {
	if len(tiff) < tiffHeaderSize {
		return 0, errMalformedExif
	}

	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0, errMalformedExif
	}

	offset := uint64(order.Uint32(tiff[4:8]))
	if offset+2 > uint64(len(tiff)) {
		return 0, errMalformedExif
	}

	count := int(order.Uint16(tiff[offset:]))
	entries := tiff[offset+2:]

	for range count {
		if len(entries) < ifdEntrySize {
			return 0, errMalformedExif
		}

		if order.Uint16(entries) != orientationTag {
			entries = entries[ifdEntrySize:]

			continue
		}

		value := int(order.Uint16(entries[8:]))
		if value > maxOrientation {
			return 0, nil
		}

		return value, nil
	}

	return 0, nil
}
