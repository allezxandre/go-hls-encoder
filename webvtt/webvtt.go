package webvtt

// Code from this file comes from `go-astisub` falls under the MIT LICENSE,
// as per `https://github.com/asticode/go-astisub/blob/master/LICENSE` at the time this code was taken.

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

type SubtitleBlock struct {
	StartTime, EndTime time.Duration // The block times
	Lines              bytes.Buffer  // A buffer containing the whole block
}

// Segment Segments the webvtt input from `r`
func Segment(r io.Reader, targetDuration time.Duration, outputDir, name string) error {
	c := make(chan SubtitleBlock)
	go ReadFromWebVTT(r, c)
	return segment(c, targetDuration, outputDir, name)
}

func segment(c <-chan SubtitleBlock, targetDuration time.Duration, outputDir, name string) error {
	playlistPath := filepath.Join(outputDir, name+".m3u8")
	playlist, err := createPlaylistFile(playlistPath, targetDuration)
	if err != nil {
		log.Println(err)
		return err
	}
	defer closePlaylistFile(playlist)

	var blocks []SubtitleBlock
	var startTime time.Duration = 0
	var endTime time.Duration = 0
	var count uint = 0

	b, ok := <-c
	segmentAdded := false
	for ok {
		newEnd := b.EndTime
		if newEnd-startTime > targetDuration+500*time.Millisecond {
			newEnd = targetDuration + startTime
		}
		endTime = newEnd
		if b.StartTime < endTime {
			// Add block to segment
			blocks = append(blocks, b)
			segmentAdded = true
		}

		// Segment now?
		if endTime-startTime >= targetDuration {
			// Yes
			createSegment(name, count, outputDir, blocks, playlist, startTime, endTime)

			// New segment
			blocks = make([]SubtitleBlock, 0, 5)
			startTime = endTime
			count += 1

			if b.StartTime < startTime && b.EndTime > startTime {
				// Add last block as it's astride these two segments
				segmentAdded = false
			}
		}
		if segmentAdded {
			// Next block
			b, ok = <-c
			segmentAdded = false
		}
	}
	if endTime-startTime > 0 {
		createSegment(name, count, outputDir, blocks, playlist, startTime, endTime)
	}
	return nil
}
func createSegment(basename string, segmentCount uint, outputDir string, blocks []SubtitleBlock, playlist *os.File, startTime, endTime time.Duration) {
	segmentName := fmt.Sprintf("%s-%05d.vtt", basename, segmentCount)
	segmentFilepath := filepath.Join(outputDir, segmentName)
	writeBlocksToVTT(blocks, segmentFilepath)
	addSegmentToPlaylist(playlist, endTime-startTime, segmentName)
}

func writeBlocksToVTT(blocks []SubtitleBlock, filepath string) error {
	var dataBuffer bytes.Buffer

	// Add header
	dataBuffer.WriteString("WEBVTT\n\n")
	// Add blocks
	for _, b := range blocks {
		dataBuffer.Write(b.Lines.Bytes())
	}

	// Save to file
	err := ioutil.WriteFile(filepath, dataBuffer.Bytes(), 0644)
	if err != nil {
		log.Println("Cannot write blocks:", err)
	}
	return err
}

func createPlaylistFile(filepath string, targetDuration time.Duration) (f *os.File, err error) {
	f, err = os.Create(filepath)
	if err != nil {
		return
	}

	// Write M3U8 header
	_, err = f.WriteString("#EXTM3U\n" + "#EXT-X-VERSION:5\n" +
		fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", int(targetDuration.Seconds())))

	return
}

func closePlaylistFile(f *os.File) {
	f.WriteString("#EXT-X-ENDLIST\n")
	f.Close()
}

func addSegmentToPlaylist(p *os.File, duration time.Duration, name string) {
	p.WriteString(fmt.Sprintf("#EXTINF:%.6f,\n%s\n", duration.Seconds(), name))
}
