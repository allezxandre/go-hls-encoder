package converter

import (
	"fmt"
	"os/exec"

	"github.com/allezxandre/go-hls-encoder/suggest"
	"github.com/allezxandre/go-hls-encoder/webvtt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type subtitleConversionCommand struct {
	EncoderCommand  *exec.Cmd
	OutputDir, Name string
	Logfile         *os.File // The logfile to use, or Nil to use Stderr
}

type SubtitleVariantConversion struct {
	Variant  suggest.SubtitleVariant
	commands *subtitleConversionCommand
}

// start Starts the conversion of the subtitles in a new goroutine,
// and returns a channel that will be closed when the conversion is done.
func (sCmds subtitleConversionCommand) start() error {
	// Pipe Stderr to logfile
	if sCmds.Logfile != nil {
		sCmds.EncoderCommand.Stderr = sCmds.Logfile
	} else {
		sCmds.EncoderCommand.Stderr = os.Stderr
	}
	// Pipe Stdout to segmenter
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

// callSubtitleConversions Starts all subtitle conversions asynchroneously.
func callSubtitleConversions(variants []suggest.SubtitleVariant, outputDir string) (conversions []SubtitleVariantConversion) {
	for _, v := range variants {
		cmds := convertSubtitle(v, outputDir)
		err := cmds.start()
		if err != nil {
			log.Println("Cannot convert subtitle variant", v.Name, "\nError:", err)
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

	// Set output file
	logFilename := filepath.Join(outputDir, fmt.Sprint("conversion-%s.log", variant.Name))
	logFile, err := os.Create(logFilename)
	if err != nil {
		log.Println("Cannot create logfile for subtitle conversion command:", err)
		// FIXME: return error
		logFile = nil
	}

	subtitleCmds := subtitleConversionCommand{
		EncoderCommand: encode,
		OutputDir:      outputDir,
		Name:           variant.Name,
		Logfile:        logFile,
	}

	return subtitleCmds
}
