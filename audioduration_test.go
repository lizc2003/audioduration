package audioduration

import (
	"fmt"
	"os"
	"testing"
)

const delta = 1e-2

type audioTest struct {
	path     string
	duration float64
}

func TestFLAC(t *testing.T) {
	var sampleDuration float64 = 3.3993650793650794
	testFile := "samples/sample.flac"
	file, err := os.Open(testFile)
	if err != nil {
		t.Errorf("Sample FLAC file(%s): %s.\n", testFile, err)
	}
	defer file.Close()
	d, err := FLAC(file)
	fmt.Println(sampleDuration, d)
	if err != nil {
		t.Errorf("%s\n", err)
	}
	if (d - sampleDuration) > delta {
		t.Errorf("too much error, expected '%v', found '%v'\n", sampleDuration, d)
	}
}

func TestMp4(t *testing.T) {
	var sampleDuration float64 = 3.4133333333333336
	testFile := "samples/sample.mp4"
	file, err := os.Open(testFile)
	if err != nil {
		t.Errorf("Sample MP4 file(%s): %s.\n", testFile, err)
	}
	defer file.Close()
	d, err := Mp4(file)
	fmt.Println(sampleDuration, d)
	if err != nil {
		t.Errorf("%s\n", err)
	}
	if (d - sampleDuration) > delta {
		t.Errorf("too much error, expected '%v', found '%v'\n", sampleDuration, d)
	}
}

func TestM4a(t *testing.T) {
	var sampleDuration float64 = 3.4133333333333336
	testFile := "samples/sample.m4a"
	file, err := os.Open(testFile)
	if err != nil {
		t.Errorf("Sample M4A file(%s): %s.\n", testFile, err)
	}
	defer file.Close()
	d, err := Mp4(file)
	fmt.Println(sampleDuration, d)
	if err != nil {
		t.Errorf("%s\n", err)
	}
	if (d - sampleDuration) > delta {
		t.Errorf("too much error, expected '%v', found '%v'\n", sampleDuration, d)
	}
}

func TestMp3FileSet(t *testing.T) {
	testFileSet := map[string]audioTest{
		// https://commons.wikimedia.org/w/index.php?title=File%3ABWV_543-prelude.ogg
		"MPEG Layer 3 (CBR)": {"samples/sample_cbr.mp3", 3.030204},
		// https://commons.wikimedia.org/w/index.php?title=File%3ABWV_543-prelude.ogg
		"MPEG Layer 3 (VBR)": {"samples/sample_vbr.mp3", 3.030204},
		// https://github.com/dhowden/tag/tree/master/testdata
		"MP3 with ID3 tags":    {"samples/sample.id3v24.mp3", 3.4560563793862875},
		"MP3 without ID3 tags": {"samples/sample.mp3", 3.4560563793862875},
	}
	for k, v := range testFileSet {
		fmt.Printf("Testing: %s\n", k)
		file, err := os.Open(v.path)
		if err != nil {
			t.Errorf("Sample MP3 file(%s): %s.\n", v.path, err)
		}
		defer file.Close()
		d, err := Mp3(file)
		fmt.Println(v.duration, d)
		if err != nil {
			t.Errorf("Sample MP3 file(%s): %s.\n", v.path, err)
		}
		if (d - v.duration) > delta {
			t.Errorf("too much error, expected '%v', found '%v' on item '%v'\n", v.duration, d, k)
		}
	}
}

func TestMp2(t *testing.T) {
	t.SkipNow()
	testFile := "samples/sample.mp2"
	file, err := os.Open(testFile)
	if err != nil {
		t.Errorf("Sample MP3 file(%s): %s.\n", testFile, err)
	}
	defer file.Close()
	d, err := Mp3(file)
	if err != nil {
		t.Errorf("Sample MP3 file(%s): %s.\n", testFile, err)
	}
	fmt.Println(d)
}

func TestOgg(t *testing.T) {
	var sampleDuration float64 = 6.104036281179138
	// https://commons.wikimedia.org/wiki/File:Example.ogg
	// https://upload.wikimedia.org/wikipedia/commons/c/c8/Example.ogg
	testFile := "samples/example.ogg"
	file, err := os.Open(testFile)
	if err != nil {
		t.Errorf("Sample OGG file(%s): %s.\n", testFile, err)
	}
	defer file.Close()
	d, err := Ogg(file)
	fmt.Println(sampleDuration, d)
	if err != nil {
		t.Errorf("Sample OGG file(%s): %s.\n", testFile, err)
	}
	if (d - sampleDuration) > delta {
		t.Errorf("too much error, expected '%v', found '%v'\n", sampleDuration, d)
	}
}

func TestDSD(t *testing.T) {
	// t.SkipNow()
	var sampleDuration float64 = 1.468
	testFile := "samples/sample.dsf"
	file, err := os.Open(testFile)
	if err != nil {
		t.Errorf("Sample DSD file(%s): %s.\n", testFile, err)
	}
	d, err := DSD(file)
	defer file.Close()
	fmt.Println(sampleDuration, d)
	if err != nil {
		t.Errorf("Sample DSD file(%s): %s.\n", testFile, err)
	}
	if (d - sampleDuration) > delta {
		t.Errorf("too much error, expected '%v', found '%v'\n", sampleDuration, d)
	}
}
