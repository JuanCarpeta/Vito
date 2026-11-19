package main

// Cerebro: prompt del sistema, historial y bucle de function calling.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	modelo      = "llama3.1:8b"
	maxPasos    = 4
	temperatura = 0.0
)

var historial []Mensaje

func construirPrompt() string {
	var b strings.Builder
	b.WriteString("Eres Vito, un asistente de terminal. Hablas como una persona en una conversación natural: ")
	b.WriteString("respuestas cortas de una a tres frases, tono cercano y amable, sin listas, sin asteriscos y sin emojis. Responde siempre en español.\n\n")
	if resumen := ResumenMemoria(); resumen != "" {
		b.WriteString(resumen)
		b.WriteString("\n\n")
	}
	b.WriteString("Estas ubicado en: " + directorioActual + ".\n")
	b.WriteString("Usa SIEMPRE las herramientas cuando el usuario pida algo del computador; nunca inventes rutas ni resultados. ")
	b.WriteString("Para buscar archivos usa 'buscar_archivos' con palabras clave del nombre. Para ubicarte en una carpeta usa 'navegar_carpeta' (solo te mueves, no abres ventanas). ")
	b.WriteString("Abre con 'abrir_archivo' solo cuando el usuario diga 'abre', 'ejecuta' o 'lanza'. Para leer usa 'leer_archivo'; para ver una carpeta 'listar_carpeta'; para tamano o fecha 'info_archivo'. ")
	b.WriteString("Para crear usa 'crear_archivo' y 'crear_carpeta'; si no te dan nombre o carpeta, pregunta antes de crear y sugiere Documentos. ")
	b.WriteString("Para preguntas de internet, noticias o datos actuales usa 'buscar_web' y cuenta lo que encontraste sin inventar. ")
	b.WriteString("Cuando el usuario pida que recuerdes algo usa 'guardar_memoria'; para saber que recuerdas 'consultar_memoria'; para borrar 'olvidar_memoria'. ")
	b.WriteString("Nunca digas que hiciste algo si no llamaste a la herramienta correspondiente. Si una herramienta devuelve un error, dilo tal cual. ")
	b.WriteString("Cuando ya hiciste la accion, confirmalo en una frase corta con la ruta exacta que devolvio la herramienta.")
	return b.String()
}

func definirHerramientas() []Herramienta {
	prop := func(tipo, desc string) EsquemaProp {
		return EsquemaProp{Type: tipo, Description: desc}
	}
	esquema := func(nombre, desc string, props map[string]EsquemaProp, req []string) Herramienta {
		return Herramienta{Type: "function", Function: EsquemaFunc{
			Name: nombre, Description: desc,
			Parameters: EsquemaParams{Type: "object", Properties: props, Required: req},
		}}
	}
	return []Herramienta{
		esquema("buscar_archivos", "Busca archivos por nombre en el computador. Usala cuando el usuario diga 'busca', 'encuentra', 'donde esta' o 'tengo un archivo de'. Pasa solo palabras clave del nombre.",
			map[string]EsquemaProp{"consulta": prop("string", "Palabras del nombre del archivo")}, []string{"consulta"}),
		esquema("navegar_carpeta", "Ubica a Jarvis en una carpeta SIN abrir ventanas. Usala con 've a', 'navega a', 'llevame a', 'entra en'.",
			map[string]EsquemaProp{"ruta": prop("string", "Ruta completa de la carpeta")}, []string{"ruta"}),
		esquema("abrir_archivo", "Abre un archivo o carpeta con el programa predeterminado de Windows. Solo con 'abre', 'abreme', 'ejecuta' o 'lanza'.",
			map[string]EsquemaProp{"ruta": prop("string", "Ruta completa")}, []string{"ruta"}),
		esquema("leer_archivo", "Lee el contenido de un archivo de texto. Si el resultado sale truncado, avisa que es muy grande.",
			map[string]EsquemaProp{"ruta": prop("string", "Ruta completa del archivo")}, []string{"ruta"}),
		esquema("crear_carpeta", "Crea una carpeta en la ruta indicada, incluyendo carpetas padre.",
			map[string]EsquemaProp{"ruta": prop("string", "Ruta completa de la carpeta")}, []string{"ruta"}),
		esquema("crear_archivo", "Crea un archivo de texto con el contenido dado (puede ser vacio).",
			map[string]EsquemaProp{
				"ruta":      prop("string", "Ruta completa del archivo nuevo"),
				"contenido": prop("string", "Contenido inicial (opcional)"),
			}, []string{"ruta"}),
		esquema("listar_carpeta", "Lista el contenido real de una carpeta. Usa '.' para la carpeta actual.",
			map[string]EsquemaProp{"ruta": prop("string", "Ruta de la carpeta")}, []string{"ruta"}),
		esquema("info_archivo", "Devuelve tamano, fecha de modificacion y tipo de un archivo o carpeta.",
			map[string]EsquemaProp{"ruta": prop("string", "Ruta completa")}, []string{"ruta"}),
		esquema("buscar_web", "Busca informacion actual en internet (noticias, clima, precios). Devuelve titulos, enlaces y resumenes.",
			map[string]EsquemaProp{"consulta": prop("string", "La consulta de busqueda")}, []string{"consulta"}),
		esquema("guardar_memoria", "Guarda un dato para recordarlo en futuras conversaciones ('recuerda que...').",
			map[string]EsquemaProp{
				"clave": prop("string", "Nombre corto del dato"),
				"valor": prop("string", "El dato a recordar"),
			}, []string{"clave", "valor"}),
		esquema("consultar_memoria", "Consulta lo que recuerdas del usuario.",
			map[string]EsquemaProp{"clave": prop("string", "Dato a consultar (opcional)")}, nil),
		esquema("olvidar_memoria", "Borra un dato de la memoria ('olvida eso').",
			map[string]EsquemaProp{"clave": prop("string", "Nombre del dato a olvidar")}, []string{"clave"}),
	}
}

func refrescarHistorial() {
	if len(historial) == 0 {
		historial = []Mensaje{{Role: "system", Content: construirPrompt()}}
		return
	}
	historial[0].Content = construirPrompt()
}

func recortarHistorial(maximo, conservar int) {
	if len(historial) <= maximo {
		return
	}
	cola := historial[len(historial)-conservar:]
	for len(cola) > 0 && (cola[0].Role == "tool" || cola[0].Role == "assistant") {
		cola = cola[1:]
	}
	historial = append([]Mensaje{historial[0]}, cola...)
}

// Preguntar procesa una orden del usuario y devuelve la respuesta en texto.
func Preguntar(texto string) string {
	recortarHistorial(25, 12)
	refrescarHistorial()
	historial = append(historial, Mensaje{Role: "user", Content: texto})

	for paso := 0; paso < maxPasos; paso++ {
		mensaje, err := Chatear(modelo, historial, definirHerramientas(), temperatura)
		if err != nil {
			respuesta := fmt.Sprintf("No pude consultar el modelo: %v", err)
			historial = append(historial, Mensaje{Role: "assistant", Content: respuesta})
			return respuesta
		}
		historial = append(historial, mensaje)

		if len(mensaje.ToolCalls) == 0 {
			contenido := strings.TrimSpace(mensaje.Content)
			if contenido == "" {
				contenido = "No te entendi, ¿me lo repites?"
			}
			historial[len(historial)-1].Content = contenido
			return contenido
		}

		accion := ""
		for _, llamada := range mensaje.ToolCalls {
			args := llamada.Function.Argumentos
			if len(args) == 0 {
				args = json.RawMessage("{}")
			}
			resultado := EjecutarHerramienta(llamada.Function.Nombre, args)
			historial = append(historial, Mensaje{
				Role: "tool", Content: resultado, ToolName: llamada.Function.Nombre,
			})
			switch llamada.Function.Nombre {
			case "crear_archivo", "crear_carpeta", "abrir_archivo", "navegar_carpeta":
				accion = resultado
			}
		}
		if accion != "" {
			historial = append(historial, Mensaje{Role: "assistant", Content: accion})
			return accion
		}
	}
	respuesta := "No pude completar la accion."
	historial = append(historial, Mensaje{Role: "assistant", Content: respuesta})
	return respuesta
}

func dirDatos() string {
	ejecutable, err := os.Executable()
	if err != nil {
		d, _ := os.Getwd()
		return d
	}
	return filepath.Dir(ejecutable)
}
