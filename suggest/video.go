package suggest

import (
	"errors"
	"github.com/allezxandre/go-hls-encoder/probe"
	"log"
	"strconv"
	"strings"
)

//
//
// Find the masterVideo stream.  At this point it's usually just
// the only video stream, but may need to add code here for the
// situation where we have more than one video
//
//
func masterVideo(fileStreams []*probe.ProbeStream) (streamIndex int, err error) {
	for _, stream := range fileStreams {
		if stream.CodecType == "video" {
			streamIndex := stream.Index
			return streamIndex, nil
		}
	}
	err = errors.New("could not find a video stream to use as master")
	return
}

type VideoVariant struct {
	MapInput   string  // The map value: in the form of $input:$stream
	Codec      string  // Codec to use, or "copy". Required.
	CRF        *int    // Optional. CRF Value.
	Profile    *string // Optional
	Level      *string // Required if `Profile` is provided.
	Bitrate    *string // Optional
	AddHVC1Tag bool    // Add tag `-tag:v hvc1`
	// Associated Media
	AudioGroup    *string // Optional Audio Group
	SubtitleGroup *string // Optional Subtitle Group
	// M3U8 Playlist options
	Resolution       string // Resolution for variant in M3U8 playlist
	Bandwidth        string
	ResolutionHeight *int // Optional. To use as -filter:v scale="trunc(oh*a/2)*2:HEIGHT"
}

func SuggestVideoVariants(probeDataInputs []*probe.ProbeData) (variants []VideoVariant) {
	for inputIndex, probeData := range probeDataInputs { // Loop through inputs
		if masterVideoIndex, err := masterVideo(probeData.Streams); err == nil {
			// Found a video in this input
			videoStream := probeData.Streams[masterVideoIndex]
			bandwidth := 700000 // FIXME: Handle unknown bandwidth
			if videoStream.BitRate > 0 {
				bandwidth = videoStream.BitRate
			}
			// Match codec
			switch videoStream.CodecName {
			case "h264":
				if videoStream.Height > 540 {
					// Additional low-quality variant
					// h264Width, h264Height := computeNewRatio(videoStream, 420)
					// crf := 28
					// variants = append(variants, VideoVariant{
					// 	MapInput:         strconv.Itoa(inputIndex) + ":" + strconv.Itoa(masterVideoIndex),
					// 	Codec:            "libx264",
					// 	CRF:              &crf,
					// 	ResolutionHeight: &h264Height,
					// 	Resolution:       strconv.Itoa(h264Width) + "x" + strconv.Itoa(h264Height),
					// 	Bandwidth:        strconv.Itoa(bandwidth / 10),
					// })
				}
				// Only one variant: copy
				variants = append(variants, VideoVariant{
					MapInput:   strconv.Itoa(inputIndex) + ":" + strconv.Itoa(masterVideoIndex),
					Codec:      "copy",
					Resolution: strconv.Itoa(videoStream.Width) + "x" + strconv.Itoa(videoStream.Height),
					Bandwidth:  strconv.Itoa(bandwidth),
				})
			case "h265", "hevc":
				// HEVC -> 2 variants: copy and x264
				log.Println("High efficiency stream detected. Copying...")
				variants = append(variants, VideoVariant{
					MapInput:   strconv.Itoa(inputIndex) + ":" + strconv.Itoa(masterVideoIndex),
					Codec:      "copy",
					Resolution: strconv.Itoa(videoStream.Width) + "x" + strconv.Itoa(videoStream.Height),
					Bandwidth:  strconv.Itoa(bandwidth * 2),
					AddHVC1Tag: true,
				})
				/*
					// For the x264 variant, compute height setting
					h264Width, h264Height := computeNewRatio(videoStream, 360)
					crf := 18
					variants = append(variants, VideoVariant{
						MapInput:         strconv.Itoa(inputIndex) + ":" + strconv.Itoa(masterVideoIndex),
						Codec:            "libx264",
						CRF:              &crf,
						ResolutionHeight: &h264Height,
						Resolution:       strconv.Itoa(h264Width) + "x" + strconv.Itoa(h264Height),
						Bandwidth:        strconv.Itoa(730000),
					})
				*/
			default:
				// One variant: converter to x264, after computing height setting
				h264Width, h264Height := computeNewRatio(videoStream, 1080)
				crf := 18
				variants = append(variants, VideoVariant{
					MapInput:         strconv.Itoa(inputIndex) + ":" + strconv.Itoa(masterVideoIndex),
					Codec:            "libx264",
					CRF:              &crf,
					ResolutionHeight: &h264Height,
					Resolution:       strconv.Itoa(h264Width) + "x" + strconv.Itoa(h264Height),
					Bandwidth:        strconv.Itoa(bandwidth),
				})
			}

		}
	}
	return
}

func computeNewRatio(videoStream *probe.ProbeStream, maximumHeight int) (int, int) {
	h264Height := videoStream.Height // The height of the h264 stream to use
	if h264Height > maximumHeight {
		h264Height = maximumHeight
		// Find aspect ratio
		ratio := 1.777778 // (16/9)
		ratioStrings := strings.Split(videoStream.DisplayAspectRatio, ":")
		if len(ratioStrings) == 2 {
			a, err1 := strconv.ParseFloat(ratioStrings[0], 64)
			b, err2 := strconv.ParseFloat(ratioStrings[1], 64)
			if err1 == nil && err2 == nil {
				ratio = a / b
			} else {
				log.Println("WARNING: Cannot parse aspect ratio (" + videoStream.DisplayAspectRatio + "). Defaulting to 16/9")
			}
		} else {
			log.Println("WARNING: Unexpected aspect ratio format (" + videoStream.DisplayAspectRatio + "). Defaulting to 16/9")
		}
		// Return final resolution
		return int(float64(h264Height) * ratio), h264Height
	} else {
		return videoStream.Width, videoStream.Height
	}
}
