package suggest

import (
	"github.com/allezxandre/go-hls-encoder/probe"
	"strconv"
)

type SubtitleVariant struct {
	MapInput string // The map value: in the form of $input:$stream

	// M3U8 Playlist options: https://tools.ietf.org/html/draft-pantos-http-live-streaming-23
	Name            string  // Unique name for variant. Required.
	GroupID         *string // Optional group ID. "subtitles" will be used if `nil`
	HearingImpaired bool
	Forced          bool
	Language        Language // Primary language https://tools.ietf.org/html/rfc5646
}

var DefaultSubtitlesGroupID = "subtitles"

func SuggestSubtitlesVariants(probeDataInputs []*probe.ProbeData, removeVFQ bool) (variants []SubtitleVariant) {
	languages := make(map[Language][]SubtitleVariant)
	// Create a map of languages to their subtitles
	for inputIndex, probeData := range probeDataInputs {
		for streamIndex, stream := range probeData.Streams {
			if stream.CodecType == "subtitle" && stream.CodecName != "hdmv_pgs_subtitle" {
				language := matchLanguage(stream)
				variant := SubtitleVariant{
					MapInput:        strconv.Itoa(inputIndex) + ":" + strconv.Itoa(streamIndex),
					Language:        language,
					Name:            "Subtitle " + strconv.Itoa(streamIndex),
					HearingImpaired: matchHearingImpairedTag(stream),
					Forced:          matchForcedTag(stream),
				}
				languages[language] = append(languages[language], variant)
			}
		}
	}
	// For each language...
	for language, subtitleVariants := range languages {
		if removeVFQ && language == QuebecLanguage {
			// Skip VFQ
			continue
		}
		if len(subtitleVariants) > 0 {
			gotForced := false
			gotFull := false
			for _, subVariant := range subtitleVariants {
				// Pick one Forced version
				if !gotForced && subVariant.Forced {
					variants = append(variants, subVariant)
					gotForced = true
				}
				// Pick one Full version
				if !gotFull && !subVariant.Forced {
					variants = append(variants, subVariant)
					gotFull = true
				}
				if gotForced && gotFull {
					break
				}
			}
		}
	}
	return
}
