package alexa

const (
	RequestTypeLaunch       = "LaunchRequest"
	RequestTypeIntent       = "IntentRequest"
	RequestTypeSessionEnded = "SessionEndedRequest"

	IntentArrival      = "ArrivalIntent"
	IntentDeparture    = "DepartureIntent"
	IntentHelp         = "AMAZON.HelpIntent"
	IntentStop         = "AMAZON.SpotIntent"
	IntentCancel       = "AMAZON.CancelIntent"
	IntentFallback     = "AMAZON.FallbackIntent"
	IntentNavigateHome = "AMAZON.NavigateHomeIntent"
)

type Request struct {
	Version string      `json:"version"`
	Session Session     `json:"session"`
	Body    RequestBody `json:"request"`
}

type Session struct {
	Application Application `json:"application"`
}

type Application struct {
	ApplicationID string `json:"applicationId"`
}

type RequestBody struct {
	Type   string `json:"type"`
	Intent Intent `json:"intent"`
}

type Intent struct {
	Name  string          `json:"name"`
	Slots map[string]Slot `json:"slots"`
}

type Slot struct {
	Name string `json:"name"`
}
