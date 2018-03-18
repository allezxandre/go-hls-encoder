package suggest

import (
	"github.com/allezxandre/go-hls-encoder/input"
	"github.com/allezxandre/go-hls-encoder/probe"
	"path/filepath"
	"strconv"
)

type SubtitleVariant struct {
	InputURL    string // The stream URL where the subtitle should be found
	StreamIndex uint   // The stream index of the subtitle in the input from InputURL

	// M3U8 Playlist options: https://tools.ietf.org/html/draft-pantos-http-live-streaming-23
	Name            string  // Unique name for variant. Required.
	GroupID         *string // Optional group ID. "subtitles" will be used if `nil`
	HearingImpaired bool
	Forced          bool
	Language        input.Language // Primary language https://tools.ietf.org/html/rfc5646

	// A unique output index for the subtitle file.
	// Each subtitle variant should have its own.
	OutputIndex uint
}

var DefaultSubtitlesGroupID = "subtitles"

func SuggestSubtitlesVariants(probeDataInputsURLs []string, probeDataInputs []*probe.ProbeData, additionalInputs []input.SubtitleInput, removeVFQ bool) []SubtitleVariant {
	// Create a map of languages to their subtitles
	languages := make(map[input.Language][]SubtitleVariant)
	var outputIndex uint = 0

	// First using the probe data...
	for inputIndex, probeData := range probeDataInputs {
		for streamIndex, stream := range probeData.Streams {
			if stream.CodecType == "subtitle" && stream.CodecName != "hdmv_pgs_subtitle" {
				outputIndex += 1
				language := matchLanguage(stream)
				variant := SubtitleVariant{
					InputURL:        probeDataInputsURLs[inputIndex],
					StreamIndex:     uint(streamIndex),
					Language:        language,
					Name:            "Subtitle" + strconv.Itoa(streamIndex),
					HearingImpaired: matchHearingImpairedTag(stream),
					Forced:          matchForcedTag(stream),
					OutputIndex:     outputIndex,
				}
				languages[language] = append(languages[language], variant)
			}
		}
	}

	// Then using the additionalInputs, if any
	for _, subtitleInput := range additionalInputs {
		outputIndex += 1
		variant := SubtitleVariant{
			InputURL:        subtitleInput.InputURL,
			StreamIndex:     subtitleInput.StreamIndex,
			Language:        subtitleInput.Language,
			Name:            subtitleInput.Name,
			HearingImpaired: subtitleInput.HearingImpaired,
			Forced:          subtitleInput.Forced,
			OutputIndex:     outputIndex,
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

// PlaylistName Returns the name of the m3u8 playlist.
// If `outputDir` is not "", joins the filename with the outputDir
func (v SubtitleVariant) PlaylistName(outputDir string) string {
	if len(outputDir) > 0 {
		return filepath.Join(outputDir, v.PlaylistName(""))
	} else {
		return v.Name + ".m3u8"
	}
}
