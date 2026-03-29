package alexa

import (
	"context"
	"log/slog"
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

func HandleAlexaRequest(ctx context.Context, req Request, sheetsService SheetsService, expectedAppID string) Response {
	slog.Info("handling Alexa request", "type", req.Body.Type)

	if expectedAppID != "" {
		if req.Session.Application.ApplicationID != expectedAppID {
			slog.Warn("invalid application ID", "expected", expectedAppID, "actual", req.Session.Application.ApplicationID)
			return buildResponse("Desculpe, este aplicativo não está autorizado.", true)
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
	panic("unimplemented")
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
