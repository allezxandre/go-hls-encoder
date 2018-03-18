package webvtt

// Code from this file comes from `go-astisub` falls under the MIT LICENSE,
// as per `https://github.com/asticode/go-astisub/blob/master/LICENSE` at the time this code was taken.

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	webvttBlockNameComment        = "comment"
	webvttBlockNameRegion         = "region"
	webvttBlockNameStyle          = "style"
	webvttBlockNameText           = "text"
	webvttTimeBoundariesSeparator = " --> "
)

var BytesBOM = []byte{239, 187, 191}

func ReadFromWebVTT(i io.Reader, c chan<- SubtitleBlock) (err error) {
	// Init
	var scanner = bufio.NewScanner(i)
	var line string

	// Skip the header
	for scanner.Scan() {
		line = scanner.Text()
		line = strings.TrimPrefix(line, string(BytesBOM))
		line = strings.TrimSpace(line)
		if len(line) > 0 && line == "WEBVTT" {
			break
		}
	}

	// Scan
	var blockName string
	var currentBlock *SubtitleBlock
	var associated bytes.Buffer
	for scanner.Scan() {
		// Fetch line
		line = scanner.Text()
		// Check prefixes
		switch {
		case strings.HasPrefix(line, "NOTE "): // Comment
			blockName = webvttBlockNameComment
		case strings.HasPrefix(line, "Region: "): // Region
			blockName = webvttBlockNameRegion
		case strings.HasPrefix(line, "STYLE "): // Style
			blockName = webvttBlockNameStyle
		case strings.Contains(line, webvttTimeBoundariesSeparator): // Time boundaries
			blockName = webvttBlockNameText
			if currentBlock != nil {
				// Send last block
				c <- *currentBlock
			}
			// Init new item
			currentBlock = &SubtitleBlock{}
			// Add associated lines
			currentBlock.Lines.Write(associated.Bytes())
			associated = bytes.Buffer{}

			// Split line on time boundaries
			var parts = strings.Split(line, webvttTimeBoundariesSeparator)
			// Split line on space to catch inline styles as well
			var partsRight = strings.Split(parts[1], " ")
			// Parse time boundaries
			if currentBlock.StartTime, err = parseDurationWebVTT(parts[0]); err != nil {
				err = fmt.Errorf("parsing webvtt duration %q failed: %s", parts[0], err)
				log.Println(err)
				return
			}
			if currentBlock.EndTime, err = parseDurationWebVTT(partsRight[0]); err != nil {
				err = fmt.Errorf("parsing webvtt duration %q failed: %s", partsRight[0], err)
				log.Println(err)
				return
			}
		}
		// Switch on block name
		switch blockName {
		case webvttBlockNameText:
			currentBlock.Lines.WriteString(line + "\n")
		case webvttBlockNameComment, webvttBlockNameRegion, webvttBlockNameStyle:
			fallthrough
		default:
			associated.WriteString(line + "\n")
		}
		// Empty line
		if len(line) == 0 {
			// Reset block name
			blockName = ""
		}
	}
	if currentBlock != nil {
		// Send last block
		c <- *currentBlock
	}
	close(c)
	return
}

// parseDurationWebVTT parses a .vtt duration
func parseDurationWebVTT(i string) (time.Duration, error) {
	return parseDuration(i, ".", 3)
}

// parseDuration parses a duration in "00:00:00.000", "00:00:00,000" or "0:00:00:00" format
func parseDuration(i, millisecondSep string, numberOfMillisecondDigits int) (o time.Duration, err error) {
	// Split milliseconds
	var parts = strings.Split(i, millisecondSep)
	var milliseconds int
	var s string
	if len(parts) >= 2 {
		// Invalid number of millisecond digits
		s = strings.TrimSpace(parts[len(parts)-1])
		if len(s) > 3 {
			err = fmt.Errorf("astisub: Invalid number of millisecond digits detected in %s", i)
			return
		}

		// Parse milliseconds
		if milliseconds, err = strconv.Atoi(s); err != nil {
			err = fmt.Errorf("atoi of %q failed: %s", s, err)
			return
		}
		milliseconds *= int(math.Pow10(numberOfMillisecondDigits - len(s)))
		s = strings.Join(parts[:len(parts)-1], millisecondSep)
	} else {
		s = i
	}

	// Split hours, minutes and seconds
	parts = strings.Split(strings.TrimSpace(s), ":")
	var partSeconds, partMinutes, partHours string
	if len(parts) == 2 {
		partSeconds = parts[1]
		partMinutes = parts[0]
	} else if len(parts) == 3 {
		partSeconds = parts[2]
		partMinutes = parts[1]
		partHours = parts[0]
	} else {
		err = fmt.Errorf("astisub: No hours, minutes or seconds detected in %s", i)
		return
	}

	// Parse seconds
	var seconds int
	s = strings.TrimSpace(partSeconds)
	if seconds, err = strconv.Atoi(s); err != nil {
		err = fmt.Errorf("atoi of %q failed: %s", s, err)
		return
	}

	// Parse minutes
	var minutes int
	s = strings.TrimSpace(partMinutes)
	if minutes, err = strconv.Atoi(s); err != nil {
		err = fmt.Errorf("atoi of %q failed: %s", s, err)
		return
	}

	// Parse hours
	var hours int
	if len(partHours) > 0 {
		s = strings.TrimSpace(partHours)
		if hours, err = strconv.Atoi(s); err != nil {
			err = fmt.Errorf("atoi of %q failed: %s", s, err)
			return
		}
	}

	// Generate output
	o = time.Duration(milliseconds)*time.Millisecond + time.Duration(seconds)*time.Second + time.Duration(minutes)*time.Minute + time.Duration(hours)*time.Hour
	return
}
