package probe

import (
	"encoding/json"
	"gitlab.com/joutube/joutube-server/jt-error"
	"os/exec"
)

// ffmpeg probe

type ProbeFormat struct {
	Filename         string      `json:"filename,omitempty"`
	NBStreams        int         `json:"nb_streams,omitempty"`
	NBPrograms       int         `json:"nb_programs,omitempty"`
	FormatName       string      `json:"format_name,omitempty"`
	FormatLongName   string      `json:"format_long_name,omitempty"`
	StartTimeSeconds string      `json:"start_time,omitempty"`
	DurationSeconds  string      `json:"duration,omitempty"`
	Size             string      `json:"size,omitempty"`
	BitRate          string      `json:"bit_rate,omitempty"`
	ProbeScore       int         `json:"probe_score,omitempty"`
	Tags             *FormatTags `json:"tags,omitempty"`
}

type FormatTags struct {
	MajorBrand       string `json:"major_brand"`
	MinorVersion     string `json:"minor_version"`
	CompatibleBrands string `json:"compatible_brands"`
	CreationTime     string `json:"creation_time"`
}

type StreamDisposition struct {
	Default         int `json:"default"`
	Dub             int `json:"dub"`
	Original        int `json:"original"`
	Comment         int `json:"comment"`
	Lyrics          int `json:"lyrics"`
	Karaoke         int `json:"karaoke"`
	Forced          int `json:"forced"`
	HearingImpaired int `json:"hearing_impaired"`
	VisualImpaired  int `json:"visual_impaired"`
	CleanEffects    int `json:"clean_effects"`
	AttachedPic     int `json:"attached_pic"`
}

type StreamTags struct {
	CreationTime string `json:"creation_time,omitempty"`
	Language     string `json:"language,omitempty"`
	Encoder      string `json:"encoder,omitempty"`
	Title        string `json:"title,omitempty"`
}

type ProbeStream struct {
	Index              int               `json:"index"`
	CodecName          string            `json:"codec_name"`
	CodecLongName      string            `json:"codec_long_name"`
	CodecType          string            `json:"codec_type"`
	CodecTimeBase      string            `json:"codec_time_base"`
	CodecTagString     string            `json:"codec_tag_string"`
	CodecTag           string            `json:"codec_tag"`
	RFrameRate         string            `json:"r_frame_rate"`
	AvgFrameRate       string            `json:"avg_frame_rate"`
	TimeBase           string            `json:"time_base"`
	StartPts           int               `json:"start_pts"`
	StartTime          string            `json:"start_time"`
	DurationTs         uint64            `json:"duration_ts"`
	Duration           float64           `json:"duration,string"`
	BitRate            int               `json:"bit_rate,string"`
	BitsPerRawSample   string            `json:"bits_per_raw_sample"`
	NbFrames           string            `json:"nb_frames"`
	Disposition        StreamDisposition `json:"disposition,omitempty"`
	Tags               StreamTags        `json:"tags,omitempty"`
	Profile            string            `json:"profile,omitempty"`
	Width              int               `json:"width"`
	Height             int               `json:"height"`
	HasBFrames         int               `json:"has_b_frames,omitempty"`
	SampleAspectRatio  string            `json:"sample_aspect_ratio,omitempty"`
	DisplayAspectRatio string            `json:"display_aspect_ratio,omitempty"`
	PixFmt             string            `json:"pix_fmt,omitempty"`
	Level              int               `json:"level,omitempty"`
	ColorRange         string            `json:"color_range,omitempty"`
	ColorSpace         string            `json:"color_space,omitempty"`

	SampleFmt     string `json:"sample_fmt,omitempty"`
	SampleRate    string `json:"sample_rate,omitempty"`
	Channels      int    `json:"channels,omitempty"`
	ChannelLayout string `json:"channel_layout,omitempty"`
	BitsPerSample int    `json:"bits_per_sample,omitempty"`
}

type ProbeData struct {
	Format  *ProbeFormat   `json:"format,omitempty"`
	Streams []*ProbeStream `json:"streams,omitempty"`
}

func Probe(filename string) (*ProbeData, error) {
	rf, errf := exec.Command("ffprobe", "-show_format", filename, "-print_format", "json").Output()
	if errf != nil {
		return nil, errf
	}

	var v ProbeData
	// Unmarshal Format data
	err := json.Unmarshal(rf, &v)
	if err != nil {
		return &v, err
	}

	rs, errs := exec.Command("ffprobe", "-show_streams", filename, "-print_format", "json").Output()
	if errs != nil {
		return &v, errs
	}

	// Unmarshal Streams Data
	err = json.Unmarshal(rs, &v)

	return &v, err
}

func GetProbeData(streamURLs ...string) (inputProbes []*ProbeData, errFinal error) {
	for _, streamURL := range streamURLs {
		probeData, err := Probe(streamURL)
		if err != nil {
			return nil, jt_error.JoutubeError{
				ErrorType:       jt_error.ConversionError,
				Origin:          "probing file '" + streamURL + "'",
				AssociatedError: err,
			}
		}
		inputProbes = append(inputProbes, probeData)
	}
	return
}
