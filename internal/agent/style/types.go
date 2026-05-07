package style

// StyleFeatures contains quantitative writing style metrics.
type StyleFeatures struct {
	AvgSentenceLength  float64            `json:"avg_sentence_length"`
	AvgParagraphLength float64            `json:"avg_paragraph_length"`
	SentenceLengthStd  float64            `json:"sentence_length_std"`
	DialogueRatio      float64            `json:"dialogue_ratio"`
	VocabularyRichness float64            `json:"vocabulary_richness"`
	TotalChars         int                `json:"total_chars"`
	TotalSentences     int                `json:"total_sentences"`
	TotalParagraphs    int                `json:"total_paragraphs"`
	TopChars           []CharFreq         `json:"top_chars"`
	TopBigrams         []CharFreq         `json:"top_bigrams"`
	PunctuationFreq    map[string]float64 `json:"punctuation_freq"`
	RhetoricDensity    float64            `json:"rhetoric_density"`
}

type CharFreq struct {
	Char string  `json:"char"`
	Freq float64 `json:"freq"`
}

// StyleProfileSummary is returned when listing profiles.
type StyleProfileSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// PolishRequest is the input for a polish operation.
type PolishRequest struct {
	Text       string `json:"text"`
	ProfileID  string `json:"profile_id"`
	Intensity  string `json:"intensity"` // 轻微 / 中等 / 较大
}

// PolishResult is returned from a polish operation.
type PolishResult struct {
	Original string `json:"original"`
	Polished string `json:"polished"`
}
