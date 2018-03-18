package suggest

import (
	"fmt"
	"github.com/allezxandre/go-hls-encoder/input"
	"log"
	"strings"
)

// Generates the entry for the m3u8 playlist
// #EXT-X-STREAM-INF:BANDWIDTH=1500000,RESOLUTION=1920x796,CODECS="avc1.42e00a",AUDIO="audio"
func (v VideoVariant) Stanza(streamPlaylistFilename string, audioGroup *string, subtitleGroup *string) string {
	// From https://tools.ietf.org/html/draft-pantos-http-live-streaming-23#section-4.3.4.2
	var optionsList []string // The list of options to create the entry
	optionsList = append(optionsList,
		fmt.Sprintf("BANDWIDTH=%v", v.Bandwidth),
		fmt.Sprintf("RESOLUTION=%v", v.Resolution))
	if audioGroup != nil && len(*audioGroup) > 0 {
		optionsList = append(optionsList,
			fmt.Sprintf("AUDIO=\"%v\"", *audioGroup))
	}
	if subtitleGroup != nil && len(*subtitleGroup) > 0 {
		optionsList = append(optionsList,
			fmt.Sprintf("SUBTITLES=\"%v\"", *subtitleGroup))
	}
	// TODO: Add CODECS
	return fmt.Sprintf("#EXT-X-STREAM-INF:%v\n%v",
		strings.Join(optionsList, ","), streamPlaylistFilename)
}

func (v AudioVariant) Stanza(streamPlaylistFilename string) string {
	// From https://tools.ietf.org/html/draft-pantos-http-live-streaming-23#section-4.3.4.1
	var optionsList []string // The list of options to create the entry
	groupID := DefaultAudioGroupID
	if v.GroupID != nil {
		groupID = *v.GroupID
	}
	// Required attributes
	optionsList = append(optionsList,
		"TYPE=AUDIO",
		"AUTOSELECT=YES",
		fmt.Sprintf("GROUP-ID=\"%v\"", groupID),
		fmt.Sprintf("NAME=\"%v\"", v.Name))
	// Channel number
	switch v.Type {
	case SurroundSound, StereoSound:
		optionsList = append(optionsList, fmt.Sprintf("CHANNELS=\"%d\"", v.Type))
	default:
		log.Println("WARNING: Unknown number of channels")
	}
	// Language
	if v.Language != input.Unknown {
		optionsList = append(optionsList,
			fmt.Sprintf("LANGUAGE=\"%v\"", v.Language))
	}
	// Characteristics
	var characteristics []string
	if v.DescribesVideo != nil && *v.DescribesVideo {
		characteristics = append(characteristics, "public.accessibility.describes-video")
	}
	if len(characteristics) > 0 {
		optionsList = append(optionsList,
			fmt.Sprintf("CHARACTERISTICS=\"%v\"", strings.Join(characteristics, ",")))
	}
	optionsList = append(optionsList,
		fmt.Sprintf("URI=\"%v\"", streamPlaylistFilename))

	return "#EXT-X-MEDIA:" + strings.Join(optionsList, ",")
}

func (v SubtitleVariant) Stanza() string {
	streamPlaylistFilename := v.PlaylistName("")

	var optionsList []string // The list of options to create the entry
	groupID := DefaultSubtitlesGroupID
	if v.GroupID != nil {
		groupID = *v.GroupID
	}
	optionsList = append(optionsList,
		"TYPE=SUBTITLES",
		"AUTOSELECT=YES",
		fmt.Sprintf("GROUP-ID=\"%v\"", groupID),
		fmt.Sprintf("NAME=\"%v\"", v.Name))
	if v.Language != input.Unknown {
		optionsList = append(optionsList,
			fmt.Sprintf("LANGUAGE=\"%v\"", v.Language))
	}
	forced := "NO"
	if v.Forced {
		forced = "YES"
	}
	optionsList = append(optionsList,
		fmt.Sprintf("FORCED=%v", forced))
	// Characteristics
	var characteristics []string
	if v.HearingImpaired {
		characteristics = append(characteristics,
			"public.accessibility.transcribes-spoken-dialog",
			"public.accessibility.describes-music-and-sound")
	}
	if len(characteristics) > 0 {
		optionsList = append(optionsList,
			fmt.Sprintf("CHARACTERISTICS=\"%v\"", strings.Join(characteristics, ",")))
	}
	optionsList = append(optionsList,
		fmt.Sprintf("URI=\"%v\"", streamPlaylistFilename))

	return "#EXT-X-MEDIA:" + strings.Join(optionsList, ",")
}
