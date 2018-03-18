package webvtt

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestRead(t *testing.T) {
	// This test comes from https://trac.ffmpeg.org/ticket/4048
	f, err := os.Open("tests/test1.vtt")
	if err != nil {
		t.Error("Cannot read test vtt file:", err)
	}

	outputDir, err := ioutil.TempDir("", "go-hls-encoder-test")
	if err != nil {
		t.Error("Cannot create output dir:", err)
	}
	fmt.Printf("Output directory: %q\n", outputDir)

	c := make(chan SubtitleBlock)
	go ReadFromWebVTT(f, c)
	segment(c, 5*time.Second, outputDir, "test1")
	// TODO: Test output
}
