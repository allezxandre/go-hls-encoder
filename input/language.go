package input

type Language string

const ( // https://tools.ietf.org/html/rfc5646
	Unknown         Language = ""
	FrenchLanguage           = "fr"
	QuebecLanguage           = "fr-CA"
	TrueFrench               = "fr-FR"
	EnglishLanguage          = "en"
)
