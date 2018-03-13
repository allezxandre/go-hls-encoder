package iframe_playlist_generator

import (
	"github.com/grafov/m3u8"
	"log"
	"math"
	"testing"
)

var eps = 0.001 // Comparison precision

func TestFFprobe1(t *testing.T) {
	_, err := probePackets("", "tests/bigbuckbunny-400k.m3u8")
	if err != nil {
		t.Error("Cannot probe file:", err)
	}
}

func TestFFprobe2(t *testing.T) {
	_, err := probePackets("", "tests/bigbuckbunny-400k-00004.ts")
	if err != nil {
		t.Error("Cannot probe file:", err)
	}
}

func TestVariantsFromMaster(t *testing.T) {
	masterFile := "tests/bigbuckbunny.m3u8"
	variants, ty, err := variantsFromMaster(masterFile)
	if err != nil {
		t.Error("Error running function:", err)
		return
	}
	if ty != m3u8.MASTER {
		t.Error("Unexpected type for playlist")
		return
	}
	if len(variants) != 3 {
		t.Error("Unexpected number of variants")
		return
	}
	fillVariants("tests/", variants...)
	log.Println(variants[0].Chunklist)
}

func TestIFramePlaylistSegment1(t *testing.T) {
	segmentURI := "tests/bigbuckbunny-400k-00001.ts"
	p, err := iframeEntryForSegment("", segmentURI)
	if err != nil {
		t.Error("Error running iframeEntryForSegment:", err)
		return
	}
	length := len(p)
	if length != 2 {
		t.Error("Bad lenght:", length)
	}
	if length < 1 {
		return
	}
	actualFirstFrame := p[0]
	expectedFirstFrame := &IFrameEntry{
		SegmentURI:     segmentURI,
		PacketPosition: 3008,
		PacketSize:     376,
		Duration:       9.08,
	}
	if actualFirstFrame.SegmentURI != expectedFirstFrame.SegmentURI {
		t.Error("Wrong segment URI. Expected",
			expectedFirstFrame.SegmentURI,
			"got", actualFirstFrame.SegmentURI)
	}
	if actualFirstFrame.PacketPosition != expectedFirstFrame.PacketPosition {
		t.Error("Wrong packet position. Expected",
			expectedFirstFrame.PacketPosition,
			"got", actualFirstFrame.PacketPosition)
	}
	if actualFirstFrame.PacketSize != expectedFirstFrame.PacketSize {
		t.Error("Wrong packet size. Expected",
			expectedFirstFrame.PacketSize,
			"got", actualFirstFrame.PacketSize)
	}
	if math.Abs(actualFirstFrame.Duration-expectedFirstFrame.Duration) > eps {
		t.Error("Wrong duration. Expected",
			expectedFirstFrame.Duration,
			"got", actualFirstFrame.Duration)
	}
}

func TestIFramePlaylistSegment4(t *testing.T) {
	segmentURI := "tests/bigbuckbunny-400k-00004.ts"
	p, err := iframeEntryForSegment("", segmentURI)
	if err != nil {
		t.Error("Error running iframeEntryForSegment:", err)
		return
	}
	length := len(p)
	if length != 4 {
		t.Error("Bad lenght:", length)
	}
	if length < 2 {
		return
	}
	actualFirstFrame := p[1]
	expectedFirstFrame := &IFrameEntry{
		SegmentURI:     segmentURI,
		PacketPosition: 28388,
		PacketSize:     4888,
		Duration:       0.04,
	}
	if actualFirstFrame.SegmentURI != expectedFirstFrame.SegmentURI {
		t.Error("Wrong segment URI. Expected",
			expectedFirstFrame.SegmentURI,
			"got", actualFirstFrame.SegmentURI)
	}
	if actualFirstFrame.PacketPosition != expectedFirstFrame.PacketPosition {
		t.Error("Wrong packet position. Expected",
			expectedFirstFrame.PacketPosition,
			"got", actualFirstFrame.PacketPosition)
	}
	if actualFirstFrame.PacketSize != expectedFirstFrame.PacketSize {
		t.Error("Wrong packet size. Expected",
			expectedFirstFrame.PacketSize,
			"got", actualFirstFrame.PacketSize)
	}
	if math.Abs(actualFirstFrame.Duration-expectedFirstFrame.Duration) > eps {
		t.Error("Wrong duration. Expected",
			expectedFirstFrame.Duration,
			"got", actualFirstFrame.Duration)
	}
}

func TestPlaylistForVariant(t *testing.T) {
	masterFile := "tests/bigbuckbunny.m3u8"
	variants, _, _ := variantsFromMaster(masterFile)
	dir := "tests/"
	fillVariants(dir, variants...)
	p, err := iframePlaylistForVariant(dir, variants[0])
	if err != nil {
		t.Error("Cannot run `iframePlaylistForVariant`", err)
		return
	}
	if len(p.Segments) != 17 {
		t.Error("Unexpected number of segments:", len(p.Segments))
		return
	}
}
