package converter

import (
	"fmt"
	"github.com/allezxandre/go-hls-encoder/iframe-playlist-generator"
	"github.com/allezxandre/go-hls-encoder/suggest"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type Conversion struct {
	StreamURLs                 []string
	mainCommand                *exec.Cmd
	SubtitleConversionCommands []SubtitleVariantConversion
	OutputDirectory            string
}

// Applies function f to all commands related to the conversion
func (c Conversion) do(f func(cmd *exec.Cmd)) {
	f(c.mainCommand)
	for _, subConv := range c.SubtitleConversionCommands {
		f(subConv.commands.EncoderCommand)
	}
}

func (c Conversion) Signal(sig syscall.Signal) {
	c.do(func(cmd *exec.Cmd) {
		cmd.Process.Signal(sig)
	})
}

func (c Conversion) SigInt() {
	c.Signal(syscall.SIGINT)
}

// Exit Kills all remaining ongoing conversion
func (c Conversion) Exit() {
	c.do(func(cmd *exec.Cmd) {
		if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
			// Process is not done
			cmd.Process.Kill()
		}
	})
}

var hlsSettings = []string{
	"-f", "hls",
	"-hls_time", "6",
	"-hls_list_size", "0",
	//"-hls_playlist_type", "event",
	"-hls_segment_type", "fmp4",
	//"-movflags", "frag_keyframe",
	"-g", "60",
	"-hls_flags", "split_by_time",
}

func ffmpegDefaultArguments() []string {
	return []string{"-hide_banner", "-y", "-stats", "-loglevel", "warning"}
}

func LaunchConversion(outputDir, masterPlaylistName, streamPlaylistName string,
	videoVariants []suggest.VideoVariant, audioVariants []suggest.AudioVariant, subtitleVariantsCh <-chan []suggest.SubtitleVariant,
	inputs ...string) (*Conversion, error) {

	// Generate FFMPEG command
	args := ffmpegDefaultArguments()
	// ... add inputs
	for _, input := range inputs {
		args = append(args, "-i", input)
	}
	// Additional subtitle inputs will be added later

	// ... add video and audio variants
	args = append(args, videoConversionArgs(videoVariants)...)
	args = append(args, audioConversionArgs(audioVariants)...)
	// ... add HLS options
	args = append(args, hlsSettings...)
	// ... add HLS variants mapping
	args = append(args, "-var_stream_map", variantsMapArg(videoVariants, audioVariants))

	// Create stream playlist
	if err := os.MkdirAll(outputDir, 0700); err != nil {
		log.Println("Cannot create conversion dir at path '"+outputDir+"':", err)
		return nil, err // FIXME: return better error
	}
	outputFile := filepath.Join(outputDir, streamPlaylistName+"_%v.m3u8")

	// HLS options
	args = append(args, "-max_muxing_queue_size", "1024", outputFile)

	// Start video and audio conversion
	masterCh := make(chan string)
	cmd, err := callFFmpeg(filepath.Join(outputDir, "conversion.log"), args, masterCh)
	if err != nil {
		close(masterCh)
		return nil, err
	}

	// Start subtitles conversion
	subtitleVariants := <-subtitleVariantsCh
	convertedSubtitles := callSubtitleConversions(subtitleVariants, outputDir)

	// Generate master playlist
	masterFilename := filepath.Join(outputDir, masterPlaylistName+".m3u8")
	masterCh <- masterFilename
	close(masterCh)
	// ... open file
	f, err := os.OpenFile(masterFilename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		panic(err) // FIXME
	}
	// .. write
	f.WriteString("#EXTM3U\n" +
		"#EXT-X-VERSION:7\n")
	streamIndex := 0
	// ... find audio groups
	var audioGroup *string = nil
	var subtitlesGroup *string = nil
	if len(audioVariants) > 0 {
		audioGroup = &suggest.DefaultAudioGroupID
		if audioVariants[0].GroupID != nil {
			audioGroup = audioVariants[0].GroupID
		}
	}
	// ... find subtitles groups
	if len(convertedSubtitles) > 0 {
		subtitlesGroup = &suggest.DefaultSubtitlesGroupID
		if convertedSubtitles[0].Variant.GroupID != nil {
			subtitlesGroup = convertedSubtitles[0].Variant.GroupID
		}
	}
	// ... write audio
	streamIndex = len(videoVariants) // Audio playlists start after the last video variant
	for _, variant := range audioVariants {
		f.WriteString(variant.Stanza(playlistFilenameForStream(streamPlaylistName, streamIndex)) + "\n")
		streamIndex += 1
	}
	f.WriteString("\n")
	// ... write subtitles
	streamIndex = 0 // Subtitle playlists restart at 0
	for _, c := range convertedSubtitles {
		fmt.Printf("DEBUG: Adding subtitle %q to Master\n", c.Variant.Name)
		f.WriteString(c.Variant.Stanza() + "\n")
		streamIndex += 1
	}
	f.WriteString("\n\n")
	// ... write video variants
	streamIndex = 0 // Video playlists are the first
	for _, variant := range videoVariants {
		vAudioGroup := audioGroup
		if variant.AudioGroup != nil {
			vAudioGroup = variant.AudioGroup
		}
		vSubtitlesGroup := subtitlesGroup
		if variant.SubtitleGroup != nil {
			vAudioGroup = variant.SubtitleGroup
		}
		f.WriteString(variant.Stanza(playlistFilenameForStream(streamPlaylistName, streamIndex), vAudioGroup, vSubtitlesGroup) + "\n")
		streamIndex += 1
	}
	f.Close()

	return &Conversion{
		StreamURLs:                 inputs,
		mainCommand:                cmd,
		SubtitleConversionCommands: convertedSubtitles,
		OutputDirectory:            outputDir,
	}, nil
}

// Launch FFMPEG command on args and returns if it launched succesfully.
// This function does not wait for FFMPEG to complete.
func callFFmpeg(logFilename string, args []string, masterCh <-chan string) (*exec.Cmd, error) {
	logFile, err := os.Create(logFilename)
	if err != nil {
		log.Println("Cannot create logfile:", err)
		return nil, err // FIXME: return better error
	}

	cmd := exec.Command("ffmpeg", args...)
	//Debug
	fmt.Println("\nDEBUG: Running FFMPEG command:\n \"" + strings.Join(cmd.Args, "\" \"") + "\"")
	fmt.Println("DEBUG:\tUse \n\t\ttail -f " + logFilename + "\n\n\tto see output.")

	cmd.Stdout = logFile
	// TODO: Use a buffer like follows:
	// var errorBuffer bytes.Buffer
	// cmd.Stderr = io.MultiWriter(logFile, &errorBuffer)
	cmd.Stderr = logFile

	err = cmd.Start()
	if err != nil {
		log.Println("FFmpeg execution had the following error:", err)
		return cmd, err // FIXME: return better error
	}
	go func() {
		masterFilename, ok := <-masterCh
		if !ok {
			return
		}
		err := cmd.Wait()
		if err != nil {
			log.Println("Error running FFMPEG:", err)
			return
		}
		logFile.Close()
		dir, filename := filepath.Split(masterFilename)
		fmt.Printf("DEBUG: Everything is fine, but we're not generating iFrame Playlist...")
		return // FIXME: remove this
		fmt.Printf("DEBUG: Everything is fine. \n"+
			"DEBUG: Generating I-FRAME-ONLY playlists on master in directory \"%v\"\n", dir)

		err = iframe_playlist_generator.GeneratePlaylist(dir, filename)
		if err != nil {
			log.Println("An error happened genrating I-FRAME-ONLY playlist:", err)
		}
	}()
	return cmd, nil
}

func playlistFilenameForStream(streamPlaylistName string, index int) string {
	return streamPlaylistName + "_" + strconv.Itoa(index) + ".m3u8"
}
