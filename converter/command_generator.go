package converter

import (
	"github.com/allezxandre/go-hls-encoder/suggest"
	"strconv"
	"strings"
)

func videoConversionArgs(variants []suggest.VideoVariant) (args []string) {
	for outputIndex, variant := range variants {
		indexS := strconv.Itoa(outputIndex)
		// Map & codec
		args = append(args, "-map", variant.MapInput,
			"-c:v:"+indexS, variant.Codec, "-g", "60", "-sc_threshold", "0")
		if variant.Codec == "libx264" {
			// Additional X264 parameters
			args = append(args,
				"-bsf:v:"+indexS, "h264_mp4toannexb",
				"-pix_fmt", "yuv420p")
		}
		// Resolution
		if variant.ResolutionHeight != nil {
			args = append(args, "-filter:v:"+indexS,
				"scale=\"trunc(oh*a/2)*2:"+strconv.Itoa(*variant.ResolutionHeight)+"\"")
		}
		// Bitrate
		if variant.Bitrate != nil {
			args = append(args, "-b:v:"+indexS, *variant.Bitrate)
		}
		// CRF
		if variant.CRF != nil {
			args = append(args, "-crf", strconv.Itoa(*variant.CRF))
		}
		// Profile & Level
		if variant.Profile != nil && variant.Level != nil {
			args = append(args,
				"-profile:v:"+indexS, *variant.Profile,
				"-level", *variant.Level,
			)
		}
	}
	return
}

func audioConversionArgs(variants []suggest.AudioVariant) (args []string) {
	for outputIndex, variant := range variants {
		indexS := strconv.Itoa(outputIndex)
		// Map & codec
		args = append(args, "-map", variant.MapInput,
			"-c:a:"+indexS, variant.Codec, "-g", "60", "-sc_threshold", "0")
		// Bitrate
		if variant.Bitrate != nil {
			args = append(args, "-b:a:"+indexS, *variant.Bitrate)
		}
		// Convert to stereo
		if variant.ConvertToStereo {
			args = append(args,
				"-ac:a:"+indexS, "2",
				// From https://superuser.com/questions/852400/properly-downmix-5-1-to-stereo-using-ffmpeg
				"-filter:a:"+indexS, "pan=stereo|FL < 1.0*FL + 0.707*FC + 0.707*BL|FR < 1.0*FR + 0.707*FC + 0.707*BR",
			)
		}
	}

	return
}

func variantsMapArg(videoVariants []suggest.VideoVariant, audioVariants []suggest.AudioVariant) string {
	mapArray := make([]string, 0, len(videoVariants)+len(audioVariants))
	for variantIndex := range videoVariants {
		mapArray = append(mapArray, "v:"+strconv.Itoa(variantIndex))
	}
	for variantIndex := range audioVariants {
		mapArray = append(mapArray, "a:"+strconv.Itoa(variantIndex))
	}
	return strings.Join(mapArray, " ")
}
