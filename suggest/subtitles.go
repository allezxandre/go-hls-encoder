package suggest

import (
	"github.com/allezxandre/go-hls-encoder/input"
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
	Language        input.Language // Primary language https://tools.ietf.org/html/rfc5646
}

var DefaultSubtitlesGroupID = "subtitles"

func SuggestSubtitlesVariants(probeDataInputs []*probe.ProbeData, additionalInputs []input.SubtitleInput, removeVFQ bool) []SubtitleVariant {
	// Create a map of languages to their subtitles
	languages := make(map[input.Language][]SubtitleVariant)

	// First using the probe data...
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

	// Then using the additionalInputs, if any
	nbInputs := len(probeDataInputs) // The number of streams already mapped
	for inputIndex, subtitleInput := range additionalInputs {
		realInputIndex := nbInputs + inputIndex
		variant := SubtitleVariant{
			MapInput:        strconv.Itoa(realInputIndex) + ":" + strconv.Itoa(int(subtitleInput.StreamIndex)),
			Language:        subtitleInput.Language,
			Name:            subtitleInput.Name,
			HearingImpaired: subtitleInput.HearingImpaired,
			Forced:          subtitleInput.Forced,
		}
		languages[subtitleInput.Language] = append(languages[subtitleInput.Language], variant)
	}

	// Only keep one per language
	variants := cleanVariants(languages, removeVFQ)
	return variants
}

func cleanVariants(languages map[input.Language][]SubtitleVariant, removeVFQ bool) (variants []SubtitleVariant) {
	// For each language...
	for language, subtitleVariants := range languages {
		if removeVFQ && language == input.QuebecLanguage {
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
	return variants
}
