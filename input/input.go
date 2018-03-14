package input

// SubtitleInput A SubtitleInput is an input stream or a file with a
// subtitle to take inside. For instance an .srt file
type SubtitleInput struct {
	Name        string // A Unique name to represent the subtitle
	InputURL    string // The input URL. Can be a stream or a file
	StreamIndex uint   // The index of the subtitle stream in the file. For an .srt file for instance, it's 0

	// Metadata
	HearingImpaired bool
	Language        Language
	Forced          bool
}
