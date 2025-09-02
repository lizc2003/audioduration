package audioduration

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// Minimal EBML/Matroska/WebM duration reader
// It looks for Segment(0x18538067) → Info(0x1549A966) → Duration(0x4489)
// Duration is a Float (4 or 8 bytes) in seconds per Matroska spec.

const (
	ebmlIdEBML    uint64 = 0x1A45DFA3
	ebmlIdSegment uint64 = 0x18538067
	ebmlIdInfo    uint64 = 0x1549A966
	ebmlIdDur     uint64 = 0x4489
)

// WebM returns the duration in seconds by reading the EBML structure.
func WebM(r io.ReadSeeker) (float64, error) {
	// Reset to start
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}

	// Scan top-level until Segment
	for {
		id, _, err := readEbmlId(r)
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
		size, _, err := readEbmlSize(r)
		if err != nil {
			return 0, err
		}

		switch id {
		case ebmlIdSegment:
			// Enter Segment content area of length `size` (unknown size allowed)
			// We'll scan until we find Info → Duration or reach end of this segment area.
			segStart, _ := r.Seek(0, io.SeekCurrent)
			dur, err := readDurationInSegment(r, size)
			if err != nil {
				return 0, err
			}
			if dur > 0 {
				return dur, nil
			}
			// If not found, skip remainder of Segment
			if size != ebmlUnknownSize {
				// move to end of segment before continuing outer loop
				end := segStart + int64(size)
				if _, err := r.Seek(end, io.SeekStart); err != nil {
					return 0, err
				}
			}
		default:
			// Skip other top-level elements
			if err := skipBytes(r, int64(size)); err != nil {
				return 0, err
			}
		}
	}

	return 0, errors.New("webm duration not found")
}

const ebmlUnknownSize uint64 = ^uint64(0) >> 1 // treat unknown as very large; won't be produced by readEbmlSize though

func readDurationInSegment(r io.ReadSeeker, segSize uint64) (float64, error) {
	segStart, _ := r.Seek(0, io.SeekCurrent)
	var limit int64 = -1
	if segSize != ebmlUnknownSize {
		limit = int64(segSize)
	}

	for {
		if limit >= 0 {
			cur, _ := r.Seek(0, io.SeekCurrent)
			if cur-segStart >= limit {
				return 0, nil
			}
		}

		id, _, err := readEbmlId(r)
		if err != nil {
			if err == io.EOF {
				return 0, nil
			}
			return 0, err
		}
		size, _, err := readEbmlSize(r)
		if err != nil {
			return 0, err
		}

		switch id {
		case ebmlIdInfo:
			infoStart, _ := r.Seek(0, io.SeekCurrent)
			d, err := readDurationInInfo(r, size)
			if err != nil {
				return 0, err
			}
			if d > 0 {
				return d, nil
			}
			// skip rest of Info
			if _, err := r.Seek(infoStart+int64(size), io.SeekStart); err != nil {
				return 0, err
			}
		default:
			if err := skipBytes(r, int64(size)); err != nil {
				return 0, err
			}
		}
	}
}

func readDurationInInfo(r io.ReadSeeker, infoSize uint64) (float64, error) {
	infoStart, _ := r.Seek(0, io.SeekCurrent)
	for {
		cur, _ := r.Seek(0, io.SeekCurrent)
		if uint64(cur-infoStart) >= infoSize {
			return 0, nil
		}
		id, _, err := readEbmlId(r)
		if err != nil {
			if err == io.EOF {
				return 0, nil
			}
			return 0, err
		}
		size, _, err := readEbmlSize(r)
		if err != nil {
			return 0, err
		}
		if id == ebmlIdDur {
			// Duration is float (4 or 8 bytes) in seconds
			if size == 4 {
				buf := make([]byte, 4)
				if _, err := io.ReadFull(r, buf); err != nil {
					return 0, err
				}
				bits := binary.BigEndian.Uint32(buf)
				return float64(math.Float32frombits(bits)), nil
			}
			if size == 8 {
				buf := make([]byte, 8)
				if _, err := io.ReadFull(r, buf); err != nil {
					return 0, err
				}
				bits := binary.BigEndian.Uint64(buf)
				return math.Float64frombits(bits), nil
			}
			// Unsupported float size; skip
			if err := skipBytes(r, int64(size)); err != nil {
				return 0, err
			}
			continue
		}
		// Not Duration: skip
		if err := skipBytes(r, int64(size)); err != nil {
			return 0, err
		}
	}
}

// readEbmlId reads a variable-length EBML ID, returning its value and length in bytes.
func readEbmlId(r io.Reader) (uint64, int, error) {
	first := make([]byte, 1)
	if _, err := io.ReadFull(r, first); err != nil {
		return 0, 0, err
	}
	l := leadingOnePos(first[0])
	if l == 0 {
		return 0, 0, errors.New("invalid EBML ID leading bits")
	}
	idLen := l
	buf := make([]byte, idLen-1)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, 0, err
	}
	val := uint64(first[0])
	for _, b := range buf {
		val = (val << 8) | uint64(b)
	}
	return val, idLen, nil
}

// readEbmlSize reads a variable-length EBML size (VINT) and returns its numeric value
// and the byte length consumed. The marker bit in the first byte is cleared per EBML rules.
func readEbmlSize(r io.Reader) (uint64, int, error) {
	first := make([]byte, 1)
	if _, err := io.ReadFull(r, first); err != nil {
		return 0, 0, err
	}
	l := leadingOnePos(first[0])
	if l == 0 {
		return 0, 0, errors.New("invalid EBML size leading bits")
	}
	sizeLen := l
	mask := byte(0xFF >> sizeLen)
	val := uint64(first[0] & mask)
	if sizeLen > 1 {
		buf := make([]byte, sizeLen-1)
		if _, err := io.ReadFull(r, buf); err != nil {
			return 0, 0, err
		}
		for _, b := range buf {
			val = (val << 8) | uint64(b)
		}
	}
	return val, sizeLen, nil
}

func leadingOnePos(b byte) int {
	// returns 1..8 for the position of first '1' bit from MSB, else 0
	for i := 0; i < 8; i++ {
		if (b & (0x80 >> i)) != 0 {
			return i + 1
		}
	}
	return 0
}

func skipBytes(r io.ReadSeeker, n int64) error {
	if n <= 0 {
		return nil
	}
	_, err := r.Seek(n, io.SeekCurrent)
	return err
}
