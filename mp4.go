package audioduration

import (
	"encoding/binary"
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
	var bufSize uint32 = 8
	var err error = nil
	var timeScale uint32 = 0
	var duration uint32 = 0
mainloop:
	for {
		buf := make([]byte, bufSize)
		_, err = io.ReadFull(r, buf)
		if err != nil {
			break
		}
		atomLen := binary.BigEndian.Uint32(buf[0:4])
		atomType := string(buf[4:8])
		switch atomType {
		case "moov", "trak", "mdia", "minf", "stbl":
			continue
		case "mdhd":
			r.Seek(12, io.SeekCurrent)
			mdhdBuf := make([]byte, 8)
			_, err = io.ReadFull(r, mdhdBuf)
			if err != nil {
				break
			}
			timeScale = binary.BigEndian.Uint32(mdhdBuf[0:4])
			duration = binary.BigEndian.Uint32(mdhdBuf[4:8])
			r.Seek(4, io.SeekCurrent)
			break mainloop
		default:
			r.Seek(int64(atomLen-bufSize), io.SeekCurrent)
		}
	}
	return float64(duration) / float64(timeScale), err
}
