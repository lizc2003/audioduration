package audioduration

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// Minimal EBML/Matroska/WebM duration reader
// It looks for Segment(0x18538067) → Info(0x1549A966) → Duration(0x4489)
// Duration is a Float (4 or 8 bytes) in seconds per Matroska spec.

const (
	ebmlIdEBML          uint64 = 0x1A45DFA3
	ebmlIdSegment       uint64 = 0x18538067
	ebmlIdInfo          uint64 = 0x1549A966
	ebmlIdDur           uint64 = 0x4489
	ebmlIdTimeCodeScale uint64 = 0x2AD7B1
)

// WebM returns the duration in seconds by reading the EBML structure.
func WebM(r io.ReadSeeker) (float64, error) {
	first := true
	for {
		// Scan top-level until Segment
		id, size, err := readElementHeader(r)
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}

		if first {
			first = false
			if id != ebmlIdEBML {
				return 0, errors.New("not a valid EBML file")
			}
		}

		switch id {
		case ebmlIdSegment:
			// We'll scan until we find Info → Duration or reach end of this segment area.
			segStart, _ := r.Seek(0, io.SeekCurrent)
			segEnd := segStart + int64(size)

			dur, err := readDurationInSegment(r, segEnd)
			if err != nil {
				return 0, err
			}
			if dur > 0 {
				return dur, nil
			}

			// move to end of segment before continuing outer loop
			if _, err := r.Seek(segEnd, io.SeekStart); err != nil {
				return 0, err
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

func readDurationInSegment(r io.ReadSeeker, segEnd int64) (float64, error) {
	for {
		cur, _ := r.Seek(0, io.SeekCurrent)
		if cur >= segEnd {
			return 0, nil
		}

		id, size, err := readElementHeader(r)
		if err != nil {
			return 0, err
		}

		switch id {
		case ebmlIdInfo:
			infoStart, _ := r.Seek(0, io.SeekCurrent)
			infoEnd := infoStart + int64(size)
			d, err := readDurationInInfo(r, infoEnd)
			if err != nil {
				return 0, err
			}
			if d > 0 {
				return d, nil
			}
			// skip rest of Info
			if _, err := r.Seek(infoEnd, io.SeekStart); err != nil {
				return 0, err
			}
		default:
			if err := skipBytes(r, int64(size)); err != nil {
				return 0, err
			}
		}
	}
}

func readDurationInInfo(r io.ReadSeeker, infoEnd int64) (float64, error) {
	var duration float64 = 0
	var timeScale uint64 = 1000000
	hasDuration := false
	hasScale := false

loop:
	for {
		cur, _ := r.Seek(0, io.SeekCurrent)
		if cur >= infoEnd {
			break
		}

		id, size, err := readElementHeader(r)
		if err != nil {
			return 0, err
		}

		switch id {
		case ebmlIdDur:
			if size == 4 || size == 8 {
				var durationBytes [8]byte
				_, err := io.ReadFull(r, durationBytes[:size])
				if err != nil {
					return 0, err
				}
				if size == 4 {
					bits := binary.BigEndian.Uint32(durationBytes[:4])
					duration = float64(math.Float32frombits(bits))
				} else {
					bits := binary.BigEndian.Uint64(durationBytes[:8])
					duration = math.Float64frombits(bits)
				}

				hasDuration = true
				if hasScale {
					break loop
				}
			} else {
				return 0, errors.New("duration format wrong")
			}
		case ebmlIdTimeCodeScale:
			if size > 8 || size < 1 {
				return 0, errors.New("time code scale format wrong")
			}
			var scaleBytes [8]byte
			_, err := io.ReadFull(r, scaleBytes[:size])
			if err != nil {
				return 0, err
			}
			timeScale = 0
			for i := uint64(0); i < size; i++ {
				timeScale = (timeScale << 8) | uint64(scaleBytes[i])
			}
			if timeScale == 0 {
				return 0, errors.New("time code scale format wrong")
			}

			hasScale = true
			if hasDuration {
				break loop
			}
		default:
			if err := skipBytes(r, int64(size)); err != nil {
				return 0, err
			}
		}
	}

	if hasDuration {
		return duration * float64(timeScale) / 1e9, nil
	} else {
		return 0, nil
	}
}

func readElementHeader(r io.Reader) (uint64, uint64, error) {
	id, _, err := readVInt(r, false)
	if err != nil {
		return 0, 0, err
	}
	size, _, err := readVInt(r, true)
	if err != nil {
		return 0, 0, err
	}
	return id, size, nil
}

func readVInt(r io.Reader, isMask bool) (uint64, int, error) {
	var firstByte [1]byte
	if _, err := io.ReadFull(r, firstByte[:]); err != nil {
		return 0, 0, err
	}

	first := firstByte[0]

	length := 1
	for i := 7; i >= 0; i-- {
		if first&(1<<uint(i)) != 0 {
			break
		}
		length++
	}

	if length > 8 {
		return 0, 0, fmt.Errorf("VINT length too long: %d", length)
	}

	val := uint64(first)
	if isMask {
		mask := byte(0xFF >> length)
		val = uint64(first & mask)
	}
	if length > 1 {
		buf := make([]byte, length-1)
		if _, err := io.ReadFull(r, buf); err != nil {
			if err == io.ErrUnexpectedEOF {
				err = io.EOF
			}
			return 0, 0, err
		}
		for _, b := range buf {
			val = (val << 8) | uint64(b)
		}
	}

	return val, length, nil
}

func skipBytes(r io.ReadSeeker, n int64) error {
	if n <= 0 {
		return nil
	}
	_, err := r.Seek(n, io.SeekCurrent)
	return err
}
