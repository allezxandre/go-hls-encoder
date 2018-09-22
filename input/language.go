package input

type Language string

const ( // https://tools.ietf.org/html/rfc5646
	Unknown         Language = ""
	FrenchLanguage  Language = "fr"
	QuebecLanguage  Language = "fr-CA"
	TrueFrench      Language = "fr-FR"
	EnglishLanguage Language = "en"
)
