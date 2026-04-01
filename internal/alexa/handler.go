package alexa

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	requestTimeout = 10 * time.Second
	maxNameLength  = 100
)

type SheetsService interface {
	EnsureHeaders(ctx context.Context) error
	RecordArrival(ctx context.Context, name string) error
	RecordDeparture(ctx context.Context, name string) error
}

func NewHttpHandler(sheetsService SheetsService, expectedAppID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "hethod not aalowed", http.StatusMethodNotAllowed)
			return
		}

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			slog.Error("error decoding request", "error", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()

		resp := HandleAlexaRequest(ctx, req, sheetsService, expectedAppID)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Error("failed to encode response", "error", err)
		}
	}
}

func HandleAlexaRequest(ctx context.Context, req Request, sheetsService SheetsService, expectedAppID string) Response {
	slog.Info("handling Alexa request", "type", req.Body.Type)

	if expectedAppID != "" {
		if req.Session.Application.ApplicationID != expectedAppID {
			slog.Warn("invalid application ID", "expected", expectedAppID, "actual", req.Session.Application.ApplicationID)
			return buildResponse("Desculpe, este aplicativo não está autorizado.", true)
		}
	}

	if req.Body.Type == RequestTypeIntent {
		slog.Info("intet received", "intent", req.Body.Intent.Name)
		if err := sheetsService.EnsureHeaders(ctx); err != nil {
			slog.Error("error ensuring today's sheet", "error", err)
			return buildResponse("Desculpe, tive um problema ao preparar a planilha. Tente novamente.", true)
		}
	}

	switch req.Body.Type {
	case RequestTypeLaunch:
		return buildResponse("Bem-vindo ao controle de presença. Você pode dizer que alguém chegou ou saiu.", false)
	case RequestTypeIntent:
		return handleIntent(ctx, req.Body.Intent, sheetsService)
	case RequestTypeSessionEnded:
		return buildResponse("", true)
	default:
		return buildResponse("Não tenho certeza de como lidar com isso.", true)
	}
}

func handleIntent(ctx context.Context, intent Intent, sheetsService SheetsService) Response {
	switch intent.Name {
	case IntentArrival:
		return handleArrival(ctx, intent, sheetsService)
	case IntentDeparture:
		return handleDeparture(ctx, intent, sheetsService)
	case IntentHelp:
		return buildResponse("Você pode dizer algo como: João Silva chegou, ou João Silva saiu.", false)
	case IntentStop, IntentCancel:
		return buildResponse("Até logo!", true)
	case IntentFallback:
		return buildResponse("Desculpe, não entendi. Você pode dizer algo como: João Silva chegou, ou João Silva saiu.", false)
	case IntentNavigateHome:
		return buildResponse("Bem-vindo ao controle de presença. Você pode dizer que alguém chegou ou saiu.", false)
	default:
		return buildResponse("Não entendi. Tente dizer um nome seguido de chegou ou saiu.", false)
	}
}

func handleDeparture(ctx context.Context, intent Intent, sheetsService SheetsService) Response {
	name := extractName(intent)
	if name == "" {
		return buildResponse("Não consegui entender o nome. Por favor, tente novamente.", false)
	}

	if err := sheetsService.RecordDeparture(ctx, name); err != nil {
		slog.Error("error recording departure", "error", err)
		if strings.Contains(err.Error(), "no open arrival found") {
			return buildResponse(fmt.Sprintf("Não encontrei uma chegada em aberto para %s.", name), true)
		}
		return buildResponse("Desculpe, tive um problema ao registrar a saída. Tente novamente.", false)
	}

	return buildResponse(fmt.Sprintf("Entendido. A saída de %s foi registrada.", name), true)
}

func handleArrival(ctx context.Context, intent Intent, sheetsService SheetsService) Response {
	name := extractName(intent)
	if name == "" {
		return buildResponse("Não consegui entender o nome. Por favor, tente novamente.", false)
	}

	if err := sheetsService.RecordArrival(ctx, name); err != nil {
		slog.Error("error recording arrival", "error", err)
		return buildResponse("Desculpe, tive um problema ao registrar a chegada. Tente novamente.", false)
	}

	return buildResponse(fmt.Sprintf("Entendido. %s foi registrado como presente.", name), true)
}

func extractName(intent Intent) string {
	slot, ok := intent.Slots["name"]
	if !ok {
		return ""
	}

	name := strings.TrimSpace(slot.Value)
	if len(name) > maxNameLength {
		return ""
	}
	return name
}

func buildResponse(text string, endSession bool) Response {
	resp := Response{
		Version: "1.0",
		Response: ResponseBody{
			ShouldEndSession: endSession,
		},
	}

	if text != "" {
		resp.Response.OutputSpeech = &OutputSpeech{
			Type: "PlainText",
			Text: text,
		}
	}
	return resp
}
