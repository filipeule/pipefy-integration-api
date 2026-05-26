package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"pipefy-integration/internal/models"
	"strings"
	"time"
)

type PipefyService struct {
	url   string
	token string
}

func NewPipefyService(url string, token string) (*PipefyService, error) {
	if url == "" {
		return nil, fmt.Errorf("pipefy url must be not empty")
	}

	if token == "" {
		return nil, fmt.Errorf("pipefy token must be not empty")
	}

	return &PipefyService{
		url:   url,
		token: token,
	}, nil
}

func (s *PipefyService) CreateCard(ctx context.Context, data CreateUserData) (string, error) {
	mutationReq := makeCreateCardMutation(data)

	pipefyRes, err := s.doRequest(ctx, mutationReq)
	if err != nil {
		return "", err
	}

	var result struct {
		Data struct {
			CreateCard struct {
				Card struct {
					ID string `json:"id"`
				} `json:"card"`
			} `json:"createCard"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(pipefyRes, &result); err != nil {
        return "", err
    }

    if len(result.Errors) > 0 {
        msgs := make([]string, 0, len(result.Errors))

		for _, e := range result.Errors {
			msgs = append(msgs, e.Message)
		}

		return "", fmt.Errorf("pipefy error: %s", strings.Join(msgs, ", "))
    }

	return result.Data.CreateCard.Card.ID, nil
}

func (s *PipefyService) UpdateCard(
	ctx context.Context, cardID string, priority *models.PrioridadeCliente,
) error {
	mutationReq := makeUpdateCardFieldMutation(cardID, priority)

	pipefyRes, err := s.doRequest(ctx, mutationReq)
	if err != nil {
		return err
	}

	var result struct {
		Data struct {
			UpdateFieldsValues struct {
				Success    bool `json:"success"`
				UserErrors []struct {
					Field   string `json:"field"`
					Message string `json:"message"`
				} `json:"userErrors"`
			} `json:"updateFieldsValues"`
		} `json:"data"`
		Errors []struct {
			Message    string `json:"message"`
			Extensions struct {
				Code string `json:"code"`
			} `json:"extensions"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(pipefyRes, &result); err != nil {
        return err
    }

	if !result.Data.UpdateFieldsValues.Success {
		errs := result.Data.UpdateFieldsValues.UserErrors
		if len(errs) > 0 {
			msgs := make([]string, 0, len(errs))
			for _, e := range errs {
				msgs = append(msgs, fmt.Sprintf("%s: %s", e.Field, e.Message))
			}

			return fmt.Errorf("pipefy error: failed to update fields: %s", strings.Join(msgs, ", "))
		}

		return fmt.Errorf("pipefy error: failed to update card")
	}

    if len(result.Errors) > 0 {
        msgs := make([]string, 0, len(result.Errors))

		for _, e := range result.Errors {
			msgs = append(msgs, e.Message)
		}

		return fmt.Errorf("pipefy error: %s", strings.Join(msgs, ", "))
    }

	return nil
}

func (s *PipefyService) doRequest(ctx context.Context, body string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewBufferString(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.token)

	client := http.Client{
		Timeout: 5 * time.Millisecond,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, fmt.Errorf("pipefy error: unauthorized")
	case resp.StatusCode >= 500:
		return nil, fmt.Errorf("pipefy error: internal error")
	}

	bodyRes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bodyRes, nil
}

func makeCreateCardMutation(data CreateUserData) string {
	clientName := data.NomeCliente
	clientEmail := data.EmailCliente
	requestType := data.TipoSolicitacao
	netWorth := data.ValorPatrimonio

	mutationCreateReq := fmt.Sprintf(`
		query {
			mutation {
				createCard(input: {
					pipe_id: 001
					fields_attributes: [
						{ field_id: "cliente_nome", field_value: "%s" },
						{ field_id: "cliente_email", field_value: "%s" },
						{ field_id: "tipo_solicitacao", field_value: "%s" },
						{ field_id: "valor_patrimonio", field_value: "%.2f" }
						{ field_id: "prioridade", field_value: "%s" },
						{ field_id: "status", field_value: "%s" },
					]
				}) {
					card {
						id
					}
				}
			}
		}
	`, clientName, clientEmail, requestType, netWorth, models.PrioridadeNormal, models.StatusAguardandoAnalise)

	return mutationCreateReq
}

func makeUpdateCardFieldMutation(cardId string, priority *models.PrioridadeCliente) string {
	mutationUpdateReq := fmt.Sprintf(`
		query {
			mutation {
				updateFieldsValues(input: {
					nodeId: %s
					values: [
						{ fieldId: "prioridade", field_value: "%s" },
						{ fieldId: "status", field_value: "%s" },
					]
				}) {
					success
					userErrors {
						field
						message
					}
				}
			}
		}
	`, cardId, *priority, models.StatusProcessado)

	return mutationUpdateReq
}
