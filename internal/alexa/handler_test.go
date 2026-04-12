package alexa

import (
	"context"
	"errors"
	"testing"
)

type mockSheetsService struct {
	ensureHeadersErr       error
	recordArrivalErr       error
	recordDepartureErr     error
	recordDepartureOnlyErr error
	recordedArrival        string
	recordedDeparture      string
	recordedDepartureOnly  string
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

func (m *mockSheetsService) RecordDepartureOnly(ctx context.Context, name string) error {
	m.recordedDepartureOnly = name
	return m.recordDepartureOnlyErr
}

func TestHandleAlexaRequest(t *testing.T) {
	tests := []struct {
		name              string
		expectedAppID     string
		request           Request
		mock              *mockSheetsService
		wantSpeech        string
		wantEnd           bool
		wantSessionAttr   string // key expected in SessionAttributes (empty = don't check)
		wantRecordedDepOn string // expected name passed to RecordDepartureOnly
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
			name:          "Departure Intent No Open Arrival asks confirmation",
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
			wantSpeech:    "Não encontrei uma chegada em aberto para Carlos Pereira. Deseja registrar apenas a saída?",
			wantEnd:       false,
			wantSessionAttr: "pendingDepartureName",
		},
		{
			name:          "YesIntent with pending departure records departure only",
			expectedAppID: "",
			request: Request{
				Session: Session{
					Attributes: map[string]interface{}{"pendingDepartureName": "Carlos Pereira"},
				},
				Body: RequestBody{
					Type:   RequestTypeIntent,
					Intent: Intent{Name: IntentYes},
				},
			},
			mock:              &mockSheetsService{},
			wantSpeech:        "Entendido. A saída de Carlos Pereira foi registrada.",
			wantEnd:           true,
			wantRecordedDepOn: "Carlos Pereira",
		},
		{
			name:          "YesIntent without pending departure",
			expectedAppID: "",
			request: Request{
				Body: RequestBody{
					Type:   RequestTypeIntent,
					Intent: Intent{Name: IntentYes},
				},
			},
			mock:       &mockSheetsService{},
			wantSpeech: "Não tenho nenhuma ação pendente.",
			wantEnd:    true,
		},
		{
			name:          "NoIntent with pending departure cancels",
			expectedAppID: "",
			request: Request{
				Session: Session{
					Attributes: map[string]interface{}{"pendingDepartureName": "Carlos Pereira"},
				},
				Body: RequestBody{
					Type:   RequestTypeIntent,
					Intent: Intent{Name: IntentNo},
				},
			},
			mock:       &mockSheetsService{},
			wantSpeech: "Tudo bem. Nenhum registro foi feito.",
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

			if tc.wantSessionAttr != "" {
				if resp.SessionAttributes == nil {
					t.Fatalf("expected SessionAttributes with key %q, got nil", tc.wantSessionAttr)
				}
				if _, ok := resp.SessionAttributes[tc.wantSessionAttr]; !ok {
					t.Errorf("SessionAttributes missing key %q, got %v", tc.wantSessionAttr, resp.SessionAttributes)
				}
			}

			if tc.wantRecordedDepOn != "" {
				if tc.mock.recordedDepartureOnly != tc.wantRecordedDepOn {
					t.Errorf("got recordedDepartureOnly = %q, want %q", tc.mock.recordedDepartureOnly, tc.wantRecordedDepOn)
				}
			}
		})
	}
}
