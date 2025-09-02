package audioduration

import (
	"encoding/binary"
	"errors"
	"io"
)

// The specification of MP4 file could be get here.
// https://developer.apple.com/library/archive/documentation/QuickTime/QTFF/QTFFChap2/qtff2.html

// MP4 format has a hierarchy structure. The basic unit is Atom. Different type
// of Atom has different structure. And Atom could contain other types of Atom.
// The duration information could be accessed at:
// moov.trak.mdia.mdhd
// which structure is difined at:
// https://developer.apple.com/library/archive/documentation/QuickTime/QTFF/QTFFChap2/qtff2.html#//apple_ref/doc/uid/TP40000939-CH204-SW34

// Mp4 Calculate mp4 files duration.
func Mp4(r io.ReadSeeker) (float64, error) {
	var timeScale uint64 = 0
	var duration uint64 = 0
	hdr := make([]byte, 8)
	buf4 := make([]byte, 4)
	buf8 := make([]byte, 8)

	for {
		// Read atom header (size32 + type)
		if _, err := io.ReadFull(r, hdr); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return 0, err
		}
		size32 := binary.BigEndian.Uint32(hdr[0:4])
		atomType := string(hdr[4:8])
		var atomSize uint64 = uint64(size32)
		var headerLen uint64 = 8
		if atomSize == 1 {
			// extended 64-bit size
			if _, err := io.ReadFull(r, buf8); err != nil {
				return 0, err
			}
			atomSize = binary.BigEndian.Uint64(buf8)
			headerLen = 16
		}
		if atomSize < headerLen {
			return 0, errors.New("invalid MP4 atom size")
		}

		if atomType == "mdhd" {
			if _, err := io.ReadFull(r, buf4); err != nil {
				return 0, err
			}
			version := buf4[0]

			if version == 1 {
				// creation(8) + modification(8)
				skip := make([]byte, 16)
				if _, err := io.ReadFull(r, skip); err != nil {
					return 0, err
				}
				if _, err := io.ReadFull(r, buf4); err != nil {
					return 0, err
				}
				timeScale = uint64(binary.BigEndian.Uint32(buf4))
				if _, err := io.ReadFull(r, buf8); err != nil {
					return 0, err
				}
				duration = binary.BigEndian.Uint64(buf8)
			} else {
				// version 0: creation(4) + modification(4)
				if _, err := io.ReadFull(r, buf8); err != nil {
					return 0, err
				}
				if _, err := io.ReadFull(r, buf4); err != nil {
					return 0, err
				}
				timeScale = uint64(binary.BigEndian.Uint32(buf4))
				if _, err := io.ReadFull(r, buf4); err != nil {
					return 0, err
				}
				duration = uint64(binary.BigEndian.Uint32(buf4))
			}

			return float64(duration) / float64(timeScale), nil
		}

		// For container boxes, descend by continuing; for others, skip payload
		switch atomType {
		case "moov", "trak", "mdia", "minf", "stbl":
			// we're now at the start of the child area; continue to read child atoms
			continue
		default:
			toSkip := int64(atomSize - headerLen)
			if toSkip > 0 {
				if _, err := r.Seek(toSkip, io.SeekCurrent); err != nil {
					return 0, err
				}
			}
		}
	}

	return 0, errors.New("mdhd not found")
}
