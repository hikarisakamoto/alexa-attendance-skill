package alexa

type Response struct {
	Version           string                 `json:"version"`
	SessionAttributes map[string]interface{} `json:"sessionAttributes,omitempty"`
	Response          ResponseBody           `json:"response"`
}

type ResponseBody struct {
	OutputSpeech     *OutputSpeech `json:"outputSpeech,omitempty"`
	ShouldEndSession bool          `json:"shouldEndSession"`
}

type OutputSpeech struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
