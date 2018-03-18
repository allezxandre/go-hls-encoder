package suggest

import (
	"github.com/allezxandre/go-hls-encoder/input"
	"github.com/allezxandre/go-hls-encoder/probe"
	"regexp"
	"strings"
)

func matchLanguage(stream *probe.ProbeStream) input.Language {
	currentGuess := input.Unknown
	// Match title
	if len(stream.Tags.Title) > 0 {
		matchVFF := regexp.MustCompile(`\b(vff|vfi|true(\b)*french)\b`)
		matchVFQ := regexp.MustCompile(`\bvfq\b|\bqu[eé]bec[a-z]*\b`)
		matchFrench := regexp.MustCompile(`(fre|french|fran[cç]ais)`)
		matchEnglish := regexp.MustCompile(`(ang|angl|eng|engl|anglais|english|vo)`)
		titleString := strings.ToLower(stream.Tags.Title)
		switch {
		// If VFQ or VFF, just return right away
		case matchVFF.MatchString(titleString):
			return input.TrueFrench
		case matchVFQ.MatchString(titleString):
			return input.QuebecLanguage
		case matchFrench.MatchString(titleString):
			currentGuess = input.FrenchLanguage // Just a guess for now
		case matchEnglish.MatchString(titleString):
			currentGuess = input.EnglishLanguage
		}
	}
	// Match language tag
	if len(stream.Tags.Language) == 0 {
		return currentGuess
	}
	matchFrench := regexp.MustCompile(`^(fre|french|fran[cç]ais)$`)
	matchEnglish := regexp.MustCompile(`^(ang|angl|eng|engl|anglais|english)$`)
	languageString := strings.ToLower(stream.Tags.Language)
	switch {
	case matchFrench.MatchString(languageString):
		return input.FrenchLanguage
	case matchEnglish.MatchString(languageString):
		return input.EnglishLanguage
	default:
		return currentGuess
	}
}

func matchForcedTag(stream *probe.ProbeStream) bool {
	return stream.Disposition.Forced == 1
}

func matchHearingImpairedTag(stream *probe.ProbeStream) bool {
	return stream.Disposition.HearingImpaired == 1
}
