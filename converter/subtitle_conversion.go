package converter

import (
	"fmt"
	"os/exec"

	"github.com/allezxandre/go-hls-encoder/suggest"
	"github.com/allezxandre/go-hls-encoder/webvtt"
	"log"
	"strings"
	"time"
)

type subtitleConversionCommand struct {
	EncoderCommand  *exec.Cmd
	OutputDir, Name string
}

type SubtitleVariantConversion struct {
	Variant  suggest.SubtitleVariant
	commands *subtitleConversionCommand
}

// start Starts the conversion of the subtitles in a new goroutine,
// and returns a channel that will be closed when the conversion is done.
func (sCmds subtitleConversionCommand) start() error {
	// TODO: use a logfile to output conversion
	// Pipe
	webvttPipe, err := sCmds.EncoderCommand.StdoutPipe()
	if err != nil {
		return err
	}

	// Debug
	fmt.Println("\nDEBUG: FFMPEG Subtitle command:\n \"" + strings.Join(sCmds.EncoderCommand.Args, "\" \""))

	// Launch segmenter
	go webvtt.Segment(webvttPipe, 6*time.Second, sCmds.OutputDir, sCmds.Name)

	err = sCmds.EncoderCommand.Start()
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	return nil
}

func convertSubtitles(variants []suggest.SubtitleVariant, outputDir string) (conversions []SubtitleVariantConversion) {
	for _, v := range variants {
		cmds := convertSubtitle(v, outputDir)
		err := cmds.start()
		if err != nil {
			log.Println("Cannot convert variant", v.Name, "\nError:", err)
			continue
		}
		conversions = append(conversions, SubtitleVariantConversion{
			Variant:  v,
			commands: &cmds,
		})
	}
	return
}

func convertSubtitle(variant suggest.SubtitleVariant, outputDir string) subtitleConversionCommand {
	// Subtitle encoding // TODO: issue a ticket on FFMPEG: you can't encode & segment with the same command
	args := ffmpegDefaultArguments()
	// Add input
	args = append(args, "-i", variant.InputURL)
	// Map & codec
	args = append(args,
		"-map", fmt.Sprintf("0:%d", variant.StreamIndex),
		"-c:s:0", "webvtt", "-f", "webvtt", "-")

	encode := exec.Command("ffmpeg", args...)

	subtitleCmds := subtitleConversionCommand{EncoderCommand: encode, OutputDir: outputDir, Name: variant.Name}

	return subtitleCmds
}
