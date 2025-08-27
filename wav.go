package audioduration

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// Wav Calculate wav files duration.
// It parses RIFF/WAVE with fmt and data chunks. PCM and non-PCM
// with block alignment are supported via byteRate/blockAlign.
func Wav(r *os.File) (float64, error) {
	buf4 := make([]byte, 4)
	buf2 := make([]byte, 2)

	// RIFF header
	_, err := io.ReadFull(r, buf4)
	if err != nil {
		return 0, err
	}
	if string(buf4) != "RIFF" {
		return 0, errors.New("not RIFF")
	}
	// skip RIFF size (4 bytes)
	_, err = io.ReadFull(r, buf4)
	if err != nil {
		return 0, err
	}
	// WAVE
	_, err = io.ReadFull(r, buf4)
	if err != nil {
		return 0, err
	}
	if string(buf4) != "WAVE" {
		return 0, errors.New("not WAVE")
	}

	//var sampleRate uint32 = 0
	//var blockAlign uint16 = 0
	var bytesPerSec uint32 = 0
	var dataSize uint32 = 0

	// iterate chunks
loop:
	for {
		// chunk id
		_, err = io.ReadFull(r, buf4)
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
		chunkID := string(buf4)
		// chunk size
		_, err = io.ReadFull(r, buf4)
		if err != nil {
			return 0, err
		}
		chunkSize := binary.LittleEndian.Uint32(buf4)

		switch chunkID {
		case "fmt ":
			// audioFormat (2), numChannels (2), sampleRate (4), bytesPerSec (4), blockAlign (2), bitsPerSample (2), optional extra params
			_, err = io.ReadFull(r, buf2)
			if err != nil {
				return 0, err
			}
			// audioFormat := binary.LittleEndian.Uint16(buf2)
			_, err = io.ReadFull(r, buf2) // numChannels
			if err != nil {
				return 0, err
			}
			_, err = io.ReadFull(r, buf4) // sampleRate
			if err != nil {
				return 0, err
			}
			// sampleRate = binary.LittleEndian.Uint32(buf4)
			_, err = io.ReadFull(r, buf4) // byteRate
			if err != nil {
				return 0, err
			}
			bytesPerSec = binary.LittleEndian.Uint32(buf4)
			_, err = io.ReadFull(r, buf2) // blockAlign
			if err != nil {
				return 0, err
			}
			// blockAlign = binary.LittleEndian.Uint16(buf2)
			// bitsPerSample
			_, err = io.ReadFull(r, buf2)
			if err != nil {
				return 0, err
			}
			// Skip any remaining bytes in fmt chunk
			fmtRead := uint32(2 + 2 + 4 + 4 + 2 + 2)
			if chunkSize > fmtRead {
				_, err = r.Seek(int64(chunkSize-fmtRead), io.SeekCurrent)
				if err != nil {
					return 0, err
				}
			}
			if dataSize != 0 {
				break loop
			}
		case "data":
			dataSize = chunkSize
			if bytesPerSec != 0 {
				break loop
			}
			// Skip actual data
			if chunkSize > 0 {
				_, err = r.Seek(int64(chunkSize), io.SeekCurrent)
				if err != nil {
					return 0, err
				}
			}
		default:
			// skip unknown chunk
			// chunks are word aligned; sizes include data only
			if chunkSize > 0 {
				// Some chunks can be huge; use Seek
				_, err = r.Seek(int64(chunkSize), io.SeekCurrent)
				if err != nil {
					return 0, err
				}
			}
		}

		// Handle padding byte if chunk size is odd (RIFF word alignment)
		if chunkSize%2 == 1 {
			_, err = io.ReadFull(r, buf4[:1])
			if err != nil {
				return 0, err
			}
		}
	}

	if bytesPerSec == 0 {
		return 0, errors.New("missing fmt chunk")
	}
	if dataSize == 0 {
		return 0, errors.New("missing data chunk")
	}

	return float64(dataSize) / float64(bytesPerSec), nil
}
