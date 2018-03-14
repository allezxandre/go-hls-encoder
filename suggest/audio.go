package suggest

import (
	"errors"
	"fmt"
	"log"

	"github.com/allezxandre/go-hls-encoder/input"
	"github.com/allezxandre/go-hls-encoder/probe"
	"gitlab.com/joutube/joutube-server/jt-error"
	"strconv"
)

func checkforAACsecondaryAudio(fileStreams []*probe.ProbeStream) (streamIndex int, err error) {
	// see if there are any ac3 5.1 surround sound streams we can use.
	for _, stream := range fileStreams {
		if stream.CodecType == "audio" {
			if stream.Channels == 2 {
				if stream.CodecName == "aac" {
					streamIndex = stream.Index
					fmt.Printf("Found a 2 channel aac stream")
					return
					//FOUND IT
				}
			}
		}
	} //end of search for master audio that we can just copy over and not transcode.
	err = jt_error.JoutubeError{
		ErrorType:       jt_error.ConversionError,
		Origin:          "looking for AAC audio",
		AssociatedError: errors.New("could not find secondary audio"),
	}
	return -1, err
}

type AudioVariantType int

const (
	StereoSound   AudioVariantType = 2
	SurroundSound AudioVariantType = 6
)

type AudioVariant struct {
	MapInput        string           // The map value: in the form of $input:$stream
	Codec           string           // Codec to use, or "copy". Required.
	Type            AudioVariantType // Required (for naming purposes)
	Bitrate         *string          // Optional
	ConvertToStereo bool             // If true, this variant is downsampling Surround to Stereo

	// M3U8 Playlist options: https://tools.ietf.org/html/draft-pantos-http-live-streaming-23
	GroupID        *string        // Optional group ID. "audio" will be used if `nil`
	Name           string         // Unique name for variant. Required.
	Language       input.Language // Primary language https://tools.ietf.org/html/rfc5646
	DescribesVideo *bool
}

var DefaultAudioGroupID = "audio"

func SuggestAudioVariants(probeDataInputs []*probe.ProbeData, createAlternateStereo bool, removeVFQ bool) (variants []AudioVariant) {
	for inputIndex, probeData := range probeDataInputs { // Loop through inputs
		for streamIndex, stream := range probeData.Streams {
			// Find tags
			language := matchLanguage(stream)
			mapInput := strconv.Itoa(inputIndex) + ":" + strconv.Itoa(streamIndex)
			if stream.CodecType == "audio" {
				if stream.Channels <= 2 { // TODO: Handle Mono
					audioType := StereoSound
					switch stream.CodecName {
					case "aac":
						// Copy AAC audio
						variants = append(variants, AudioVariant{
							MapInput:        mapInput,
							Type:            audioType,
							Codec:           "copy",
							Name:            "Audio " + strconv.Itoa(streamIndex) + " (AAC Stereo)",
							Language:        language,
							ConvertToStereo: false,
						})
					default:
						// Convert audio to AAC
						bitrate := "256k"
						variants = append(variants, AudioVariant{
							MapInput:        mapInput,
							Type:            audioType,
							Codec:           "libfdk_aac",
							Bitrate:         &bitrate,
							Name:            "Audio " + strconv.Itoa(streamIndex),
							Language:        language,
							ConvertToStereo: false,
						})
					} // end of switch on codec
				} else {
					audioType := SurroundSound
					log.Println("Surround sound detected. Format:", stream.CodecName)
					// The Master Audio has surround sound
					switch stream.CodecName {
					case "ac3":
						_, err := checkforAACsecondaryAudio(probeData.Streams)
						if err != nil {
							// We didn't find an aac alternate
							// Copy Dolby Digital
							variants = append(variants, AudioVariant{
								MapInput:        mapInput,
								Type:            audioType,
								Codec:           "copy",
								Name:            "Audio " + strconv.Itoa(streamIndex) + " (Dolby Surround)",
								Language:        language,
								ConvertToStereo: false,
							})
							if createAlternateStereo {
								// Convert to AAC 2.0
								bitrate := "256k"
								variants = append(variants, AudioVariant{
									MapInput:        mapInput,
									Type:            StereoSound,
									Codec:           "libfdk_aac",
									Bitrate:         &bitrate,
									Name:            "Audio " + strconv.Itoa(streamIndex) + " (AAC Stereo)",
									Language:        language,
									ConvertToStereo: true,
								})
							}
						} else {
							// we found the aac 2 channel stream, no need to converter
							// Copy Dolby Digital
							variants = append(variants, AudioVariant{
								MapInput:        mapInput,
								Type:            audioType,
								Codec:           "copy",
								Name:            "Audio " + strconv.Itoa(streamIndex) + " (Dolby Surround)",
								Language:        language,
								ConvertToStereo: false,
							})
							// AAC was copied already
						}
					case "aac", "truehd", "dca", "dts":
						fallthrough
					default:
						// Convert to AAC
						bitrate1 := "384k"
						variants = append(variants, AudioVariant{
							MapInput: mapInput,
							Type:     audioType,
							Codec:    "libfdk_aac",
							Bitrate:  &bitrate1,
							Name:     "Audio " + strconv.Itoa(streamIndex) + " (AAC Surround)",
							Language: language,
						})
						if createAlternateStereo {
							// Convert to AAC 2.0
							bitrate2 := "256k"
							variants = append(variants, AudioVariant{
								MapInput:        mapInput,
								Type:            StereoSound,
								Codec:           "libfdk_aac",
								Bitrate:         &bitrate2,
								Name:            "Audio " + strconv.Itoa(streamIndex) + " (AAC Stereo)",
								Language:        language,
								ConvertToStereo: true,
							})
						}
					} // end of switch

				} //end of if channels > 2
			}
		}
	}
	if removeVFQ {
		variants = removeVFQAudio(variants)
	}
	return
}

func removeVFQAudio(variants []AudioVariant) []AudioVariant {
	// Check if French is present
	hasFrench := false
	filteredVariants := make([]AudioVariant, 0)
	for _, variant := range variants {
		hasFrench = hasFrench || variant.Language == input.FrenchLanguage || variant.Language == input.TrueFrench
		if variant.Language != input.QuebecLanguage {
			filteredVariants = append(filteredVariants, variant)
		}
	}

	if hasFrench {
		return filteredVariants
	} else {
		// We don't remove VFQ because it is the only French
		return variants
	}
}
