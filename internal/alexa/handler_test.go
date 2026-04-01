package alexa

import (
	"context"
	"errors"
	"testing"
)

type mockSheetsService struct {
	ensureHeadersErr   error
	recordArrivalErr   error
	recordDepartureErr error
	recordedArrival    string
	recordedDeparture  string
}

func (m *mockSheetsService) EnsureHeaders(ctx context.Context) error {
	return m.ensureHeadersErr
}

func (m *mockSheetsService) RecordArrival(ctx context.Context, name string) error {
	m.recordedArrival = name
	return m.recordArrivalErr
}

func (m *mockSheetsService) RecordDeparture(ctx context.Context, name string) error {
	m.recordedDeparture = name
	return m.recordDepartureErr
}

func TestHandleAlexaRequest(t *testing.T) {
	tests := []struct {
		name          string
		expectedAppID string
		request       Request
		mock          *mockSheetsService
		wantSpeech    string
		wantEnd       bool
	}{
		{
			name:          "Launch Request",
			expectedAppID: "",
			request: Request{
				Body: RequestBody{Type: RequestTypeLaunch},
			},
			mock:       &mockSheetsService{},
			wantSpeech: "Bem-vindo ao controle de presença. Você pode dizer que alguém chegou ou saiu.",
			wantEnd:    false,
		},
		{
			name:          "Invalid Application ID",
			expectedAppID: "amzn1.ask.skill.valid",
			request: Request{
				Session: Session{
					Application: Application{ApplicationID: "amzn1.ask.skill.invalid"},
				},
				Body: RequestBody{Type: RequestTypeLaunch},
			},
			mock:       &mockSheetsService{},
			wantSpeech: "Desculpe, este aplicativo não está autorizado.",
			wantEnd:    true,
		},
		{
			name:          "Missing Application ID in request is rejected",
			expectedAppID: "amzn1.ask.skill.valid",
			request: Request{
				Session: Session{
					Application: Application{ApplicationID: ""}, // absent — must still be rejected
				},
				Body: RequestBody{Type: RequestTypeLaunch},
			},
			mock:       &mockSheetsService{},
			wantSpeech: "Desculpe, este aplicativo não está autorizado.",
			wantEnd:    true,
		},
		{
			name:          "Valid Application ID - Launch",
			expectedAppID: "amzn1.ask.skill.valid",
			request: Request{
				Session: Session{
					Application: Application{ApplicationID: "amzn1.ask.skill.valid"},
				},
				Body: RequestBody{Type: RequestTypeLaunch},
			},
			mock:       &mockSheetsService{},
			wantSpeech: "Bem-vindo ao controle de presença. Você pode dizer que alguém chegou ou saiu.",
			wantEnd:    false,
		},
		{
			name:          "Help Intent",
			expectedAppID: "",
			request: Request{
				Body: RequestBody{
					Type:   RequestTypeIntent,
					Intent: Intent{Name: IntentHelp},
				},
			},
			mock:       &mockSheetsService{},
			wantSpeech: "Você pode dizer algo como: João Silva chegou, ou João Silva saiu.",
			wantEnd:    false,
		},
		{
			name:          "Stop Intent",
			expectedAppID: "",
			request: Request{
				Body: RequestBody{
					Type:   RequestTypeIntent,
					Intent: Intent{Name: IntentStop},
				},
			},
			mock:       &mockSheetsService{},
			wantSpeech: "Até logo!",
			wantEnd:    true,
		},
		{
			name:          "Ensure Headers Error",
			expectedAppID: "",
			request: Request{
				Body: RequestBody{
					Type:   RequestTypeIntent,
					Intent: Intent{Name: IntentArrival},
				},
			},
			mock: &mockSheetsService{
				ensureHeadersErr: errors.New("sheets api down"),
			},
			wantSpeech: "Desculpe, tive um problema ao preparar a planilha. Tente novamente.",
			wantEnd:    true,
		},
		{
			name:          "Arrival Intent Success",
			expectedAppID: "",
			request: Request{
				Body: RequestBody{
					Type: RequestTypeIntent,
					Intent: Intent{
						Name: IntentArrival,
						Slots: map[string]Slot{
							"name": {Name: "name", Value: "Maria Souza"},
						},
					},
				},
			},
			mock:       &mockSheetsService{},
			wantSpeech: "Entendido. Maria Souza foi registrado como presente.",
			wantEnd:    true,
		},
		{
			name:          "Departure Intent Success",
			expectedAppID: "",
			request: Request{
				Body: RequestBody{
					Type: RequestTypeIntent,
					Intent: Intent{
						Name: IntentDeparture,
						Slots: map[string]Slot{
							"name": {Name: "name", Value: "Carlos Pereira"},
						},
					},
				},
			},
			mock:       &mockSheetsService{},
			wantSpeech: "Entendido. A saída de Carlos Pereira foi registrada.",
			wantEnd:    true,
		},
		{
			name:          "Departure Intent No Open Arrival",
			expectedAppID: "",
			request: Request{
				Body: RequestBody{
					Type: RequestTypeIntent,
					Intent: Intent{
						Name: IntentDeparture,
						Slots: map[string]Slot{
							"name": {Name: "name", Value: "Carlos Pereira"},
						},
					},
				},
			},
			mock: &mockSheetsService{
				recordDepartureErr: errors.New("no open arrival found"),
			},
			wantSpeech: "Não encontrei uma chegada em aberto para Carlos Pereira.",
			wantEnd:    true,
		},
		{
			name:          "Session Ended Request",
			expectedAppID: "",
			request: Request{
				Body: RequestBody{Type: RequestTypeSessionEnded},
			},
			mock:       &mockSheetsService{},
			wantSpeech: "",
			wantEnd:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := HandleAlexaRequest(t.Context(), tc.request, tc.mock, tc.expectedAppID)

			if resp.Response.ShouldEndSession != tc.wantEnd {
				t.Errorf("got ShouldEndSession = %v, want %v", resp.Response.ShouldEndSession, tc.wantEnd)
			}

			if tc.wantSpeech != "" {
				if resp.Response.OutputSpeech == nil {
					t.Fatalf("OutputSpeech is nil, want text: %q", tc.wantSpeech)
				}
				if resp.Response.OutputSpeech.Text != tc.wantSpeech {
					t.Errorf("got text %q, want %q", resp.Response.OutputSpeech.Text, tc.wantSpeech)
				}
			} else if resp.Response.OutputSpeech != nil {
				t.Errorf("expected no OutputSpeech, got %v", resp.Response.OutputSpeech)
			}
		})
	}
}
