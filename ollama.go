package main

// Cliente de la API de Ollama (POST /api/chat, sin streaming).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const urlOllama = "http://localhost:11434/api/chat"

type Mensaje struct {
	Role      string          `json:"role"`
	Content   string          `json:"content,omitempty"`
	ToolCalls []LlamadaHerr   `json:"tool_calls,omitempty"`
	ToolName  string          `json:"tool_name,omitempty"`
	Images    []string        `json:"-"`
}

type FuncionLlamada struct {
	Nombre    string          `json:"name"`
	Argumentos json.RawMessage `json:"arguments"`
}

type LlamadaHerr struct {
	Function FuncionLlamada `json:"function"`
}

type Herramienta struct {
	Type     string       `json:"type"`
	Function EsquemaFunc  `json:"function"`
}

type EsquemaFunc struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  EsquemaParams  `json:"parameters"`
}

type EsquemaParams struct {
	Type       string                    `json:"type"`
	Properties map[string]EsquemaProp    `json:"properties"`
	Required   []string                  `json:"required,omitempty"`
}

type EsquemaProp struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type RespuestaChat struct {
	Message Mensaje `json:"message"`
	Error   string  `json:"error"`
}

// Chatear envia la conversacion y devuelve el mensaje del modelo.
func Chatear(modelo string, historial []Mensaje, herramientas []Herramienta, temperatura float64) (Mensaje, error) {
	peticion := map[string]any{
		"model":      modelo,
		"messages":   historial,
		"stream":     false,
		"options":    map[string]any{"temperature": temperatura},
	}
	if len(herramientas) > 0 {
		peticion["tools"] = herramientas
	}
	cuerpo, err := json.Marshal(peticion)
	if err != nil {
		return Mensaje{}, err
	}
	cliente := &http.Client{Timeout: 180 * time.Second}
	resp, err := cliente.Post(urlOllama, "application/json", bytes.NewReader(cuerpo))
	if err != nil {
		return Mensaje{}, fmt.Errorf("no pude conectar con Ollama: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Mensaje{}, fmt.Errorf("Ollama respondio con codigo %d", resp.StatusCode)
	}
	var datos RespuestaChat
	if err := json.NewDecoder(resp.Body).Decode(&datos); err != nil {
		return Mensaje{}, err
	}
	if datos.Error != "" {
		return Mensaje{}, fmt.Errorf("%s", datos.Error)
	}
	return datos.Message, nil
}
