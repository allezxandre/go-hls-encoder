package iframe_playlist_generator

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
)

type ProbeFrame struct {
	MediaType               string `json:"media_type"`
	StreamIndex             int    `json:"stream_index"`
	KeyFrame                bool   `json:"key_frame"`
	PktPts                  int    `json:"pkt_pts"`
	PktPtsTime              string `json:"pkt_pts_time"`
	PktDts                  int    `json:"pkt_dts"`
	PktDtsTime              string `json:"pkt_dts_time"`
	BestEffortTimestamp     int    `json:"best_effort_timestamp"`
	BestEffortTimestampTime string `json:"best_effort_timestamp_time"`
	PktPos                  string `json:"pkt_pos"`
	PktSize                 string `json:"pkt_size"`
	Width                   int    `json:"width"`
	Height                  int    `json:"height"`
	PixFmt                  string `json:"pix_fmt"`
	SampleAspectRatio       string `json:"sample_aspect_ratio"`
	PictType                string `json:"pict_type"`
	CodedPictureNumber      int    `json:"coded_picture_number"`
	DisplayPictureNumber    int    `json:"display_picture_number"`
	InterlacedFrame         int    `json:"interlaced_frame"`
	TopFieldFirst           int    `json:"top_field_first"`
	RepeatPict              int    `json:"repeat_pict"`
	ColorRange              string `json:"color_range"`
	ColorSpace              string `json:"color_space"`
	ColorPrimaries          string `json:"color_primaries"`
	ColorTransfer           string `json:"color_transfer"`
	ChromaLocation          string `json:"chroma_location"`
}

type ProbePacket struct {
	PtsTime      float64 `json:"pts_time,string"`
	DtsTime      float64 `json:"dts_time,string"`
	DurationTime float64 `json:"duration_time,string"`
	Size         uint    `json:"size,string"`
	Pos          uint    `json:"pos,string"`
	Flags        string  `json:"flags"`
}

// isFromKeyFrame Returns `true` if the packet
// comes from a key-frame.
func (p *ProbePacket) isFromKeyFrame() bool {
	if len(p.Flags) < 1 {
		log.Println("Assertion Failed: Flags length is 0. Should be at least 1")
		return false
	}
	return p.Flags[0] == 'K'
}

// probeKeyFrames Probes a file at path `filename` for its key-frames.
func probeKeyFrames(filename string) ([]*ProbeFrame, error) {
	type ProbeFrames struct {
		Frames []*ProbeFrame `json:"frames"`
	}

	rf, errf := exec.Command("ffprobe",
		"-skip_frame", "nokey",
		"-select_streams", "v",
		"-show_frames", filename,
		"-print_format", "json",
	).Output()
	if errf != nil {
		return nil, errf
	}

	var v ProbeFrames
	// Unmarshal Frames data
	err := json.Unmarshal(rf, &v)
	if err != nil {
		return v.Frames, err
	}

	return v.Frames, err
}

// probePackets Probes a file at path `filename` for its packets.
func probePackets(initfilename string, filename string) ([]*ProbePacket, error) {
	type ProbePackets struct {
		Packets []*ProbePacket `json:"packets"`
	}

	var cmd *exec.Cmd
	var rp []byte
	var errp error
	if len(initfilename) > 0 {
		// FIXME: use automatic init
		cmd = exec.Command("ffprobe",
			"-hide_banner", "-loglevel", "warning",
			"-show_packets",
			"-select_streams", "v",
			"-show_entries", "packet=pts_time,dts_time,size,pos,flags,duration_time",
			"-print_format", "json",
			"-",
		)
		cmd.Stderr = os.Stderr
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		var err error
		// FIXME: use bytes instead of cat
		log.Println("DEBUG: cat", initfilename, filename)
		cmdCat := exec.Command("cat", initfilename, filename)
		cmd.Stdin, err = cmdCat.StdoutPipe()
		if err != nil {
			log.Println("DEBUG: cannot create pipe")
			return []*ProbePacket{}, err
		}
		err = cmd.Start()
		if err != nil {
			log.Println("DEBUG: Cannot start ffprobe:", err)
			return []*ProbePacket{}, err
		}
		err = cmdCat.Run()
		if err != nil {
			log.Println("DEBUG: Error running cat:", err)
			return []*ProbePacket{}, err
		}
		errp = cmd.Wait()
		rp = stdout.Bytes()
	} else {
		cmd = exec.Command("ffprobe", "-hide_banner",
			"-show_packets",
			"-select_streams", "v",
			"-show_entries", "packet=pts_time,dts_time,size,pos,flags,duration_time",
			"-print_format", "json",
			filename,
		)
		cmd.Stderr = os.Stderr
		rp, errp = cmd.Output()
	}
	if errp != nil {
		return nil, errp
	}

	var v ProbePackets
	// Unmarshal Frames data
	err := json.Unmarshal(rp, &v)
	if err != nil {
		return v.Packets, err
	}

	return v.Packets, err
}
