package converter

import (
	"fmt"
	"github.com/allezxandre/go-hls-encoder/iframe-playlist-generator"
	"github.com/allezxandre/go-hls-encoder/input"
	"github.com/allezxandre/go-hls-encoder/probe"
	"github.com/allezxandre/go-hls-encoder/suggest"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Conversion struct {
	StreamURLs      []string
	Command         *exec.Cmd
	OutputDirectory string
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

func ConvertFile(outputDir, masterPlaylistName, streamPlaylistName string, additionalSubtitleInputs []input.SubtitleInput, inputs ...string) (*Conversion, error) {
	// Probe data
	probeData, err := probe.GetProbeData(inputs...)
	if err != nil {
		return nil, err
	}

	// Figure out variants
	videoVariants := suggest.SuggestVideoVariants(probeData)
	audioVariants := suggest.SuggestAudioVariants(probeData, false, true)
	subtitleVariants := suggest.SuggestSubtitlesVariants(probeData, additionalSubtitleInputs, true)
	maxSubs := len(videoVariants) + len(audioVariants) // FIXME
	if maxSubs < len(subtitleVariants) {
		subtitleVariants = subtitleVariants[:maxSubs]
		log.Println("Warning: some subtitles won't be copied as you can't have " +
			"subtitle-only variants and there aren't enough other video and audio variants.")
	}

	// Generate FFMPEG command
	args := []string{"-hide_banner", "-y", "-stats", "-loglevel", "warning"}
	// ... add inputs
	for _, input := range inputs {
		args = append(args, "-i", input)
	}
	for _, additionalSubtitleInput := range additionalSubtitleInputs {
		args = append(args, "-i", additionalSubtitleInput.InputURL)
	}

	// ... add variants
	args = append(args, videoConversionArgs(videoVariants)...)
	args = append(args, audioConversionArgs(audioVariants)...)
	args = append(args, subtitlesConversionArgs(subtitleVariants)...)
	// ... add HLS options
	args = append(args, hlsSettings...)
	// ... add HLS variants mapping
	args = append(args, "-var_stream_map", variantsMapArg(videoVariants, audioVariants, subtitleVariants))

	// Create stream playlist

	if err := os.MkdirAll(outputDir, 0700); err != nil {
		log.Println("Cannot create conversion dir at path '"+outputDir+"':", err)
		return nil, err // FIXME: return better error
	}
	outputFile := filepath.Join(outputDir, streamPlaylistName+"_%v.m3u8")

	// HLS options
	args = append(args, "-max_muxing_queue_size", "1024", outputFile)

	// Start conversion
	var cmd *exec.Cmd
	masterCh := make(chan string)
	cmd, err = callFFmpeg(filepath.Join(outputDir, "conversion.log"), args, masterCh)
	if err != nil {
		close(masterCh)
		return nil, err
	}

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
	if len(subtitleVariants) > 0 {
		subtitlesGroup = &suggest.DefaultSubtitlesGroupID
		if subtitleVariants[0].GroupID != nil {
			subtitlesGroup = subtitleVariants[0].GroupID
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
	for _, variant := range subtitleVariants {
		f.WriteString(variant.Stanza(playlistFilenameForSubtitlesStream(streamPlaylistName, streamIndex)) + "\n")
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
		StreamURLs:      inputs,
		Command:         cmd,
		OutputDirectory: outputDir,
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
	cmd.Stderr = io.MultiWriter(logFile, os.Stderr)

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

func playlistFilenameForSubtitlesStream(streamPlaylistName string, index int) string {
	return streamPlaylistName + "_" + strconv.Itoa(index) + "_vtt.m3u8"
}
