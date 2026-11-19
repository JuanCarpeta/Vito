package main

// Cerebro: prompt del sistema, historial y bucle de function calling.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	if inventario := ResumenInventario(); inventario != "" {
		b.WriteString("Conoces este equipo:\n" + inventario + "\n\n")
	}
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
		esquema("abrir_archivo", "Abre un archivo, carpeta o aplicacion instalada con el programa predeterminado de Windows. Para aplicaciones pasa SOLO el nombre (por ejemplo 'steam', 'firefox', 'spotify'), nunca inventes rutas. Solo con 'abre', 'abreme', 'ejecuta' o 'lanza'.",
			map[string]EsquemaProp{"ruta": prop("string", "Ruta completa del archivo, o solo el nombre de la aplicacion")}, []string{"ruta"}),
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

// Intenciones directas: si la orden es clara, se ejecuta sin consultar al
// modelo (llama3.1 a veces responde de memoria en vez de usar herramientas).

var (
	reNavegar = regexp.MustCompile(
		`(?i)^\s*(?:ve|vete|vaya|navega|navegar|dirigete|dirígete|llevame|llévame|ponme|muevete|muévete|entra|cambia)\s+(?:a|al|hacia|en|para|me)?\s*(?:a|al|en|la|el)?\s*(.+?)\s*[.?!]?\s*$`)
	reListar = regexp.MustCompile(
		`(?i)\b(que hay|qué hay|lista|listame|listar|muestrame|muéstrame|contenido|que contiene|qué contiene)\b`)
	reAqui = regexp.MustCompile(
		`(?i)\b(aqui|aquí|actual|esta carpeta|este directorio|donde estoy|disco local|disco c)\b`)
	reWeb = regexp.MustCompile(
		`(?i)\b(internet|google|web|online|en linea|línea)\b`)
)

// destinosConocidos traduce apodos comunes a rutas reales.
var destinosConocidos = map[string]string{
	"disco local":    `C:\`,
	"disco c":        `C:\`,
	"raiz":           `C:\`,
	"raíz":           `C:\`,
	"documentos":     "",
	"descargas":      "",
	"escritorio":     "",
	"imagenes":       "",
	"imágenes":       "",
	"musica":         "",
	"música":         "",
	"videos":         "",
}

func rutaConocida(nombre string) (string, bool) {
	clave := strings.ToLower(strings.TrimSpace(nombre))
	if ruta, ok := destinosConocidos[clave]; ok {
		if ruta != "" {
			return ruta, true
		}
		tipo, err := tipoEspecial(clave)
		if err != nil {
			return "", false
		}
		return tipo, true
	}
	return "", false
}

// tipoEspecial resuelve carpetas de usuario (Documentos, Descargas, etc.).
func tipoEspecial(clave string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	nombres := map[string][]string{
		"documentos": {"Documents", "Documentos", "Mis Documentos"},
		"descargas":  {"Downloads", "Descargas"},
		"escritorio": {"Desktop", "Escritorio"},
		"imagenes":   {"Pictures", "Imágenes"},
		"imágenes":   {"Pictures", "Imágenes"},
		"musica":     {"Music", "Música"},
		"música":     {"Music", "Música"},
		"videos":     {"Videos", "Vídeos"},
	}
	subcarpetas := nombres[clave]
	if len(subcarpetas) == 0 {
		return "", fmt.Errorf("carpeta desconocida")
	}
	candidatas := []string{}
	for _, sub := range subcarpetas {
		candidatas = append(candidatas,
			filepath.Join(home, sub),
			filepath.Join(home, "OneDrive", sub))
	}
	for _, c := range candidatas {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("no encontre la carpeta %s", clave)
}

// resolverDestino convierte el texto de destino en una ruta existente.
func resolverDestino(destino string) (string, bool) {
	destino = strings.Trim(destino, `"' `)
	if destino == "" {
		return "", false
	}
	if ruta, ok := rutaConocida(destino); ok {
		return ruta, true
	}
	if filepath.IsAbs(destino) {
		if info, err := os.Stat(destino); err == nil && info.IsDir() {
			return destino, true
		}
		return "", true // destino absoluto mencionado pero inexistente
	}
	candidata := filepath.Join(directorioActual, destino)
	if info, err := os.Stat(candidata); err == nil && info.IsDir() {
		return candidata, true
	}
	return "", false
}

// responderDirecto ejecuta navegar y listar sin modelo; "" = que decida el modelo.
func responderDirecto(texto string) string {
	if reWeb.MatchString(texto) {
		return ""
	}
	if coincidencias := reNavegar.FindStringSubmatch(texto); coincidencias != nil {
		ruta, mencionado := resolverDestino(coincidencias[1])
		if ruta != "" {
			return navegar(ruta)
		}
		if mencionado {
			return fmt.Sprintf("No existe: %s", strings.TrimSpace(coincidencias[1]))
		}
		return ""
	}
	if reListar.MatchString(texto) {
		// "que hay en X": destino despues de 'en/de'; sin destino = carpeta actual.
		destino := ""
		if m := regexp.MustCompile(`(?i)\b(?:en|de)\s+(.+?)\s*[.?!]?$`).FindStringSubmatch(texto); m != nil {
			destino = m[1]
		}
		if destino != "" && !reAqui.MatchString(destino) {
			if ruta, ok := resolverDestino(destino); ok {
				if ruta != "" {
					return listar(ruta)
				}
				// Destino mencionado pero no resuelto: que decida el modelo.
				return ""
			}
		}
		return listar(directorioActual)
	}
	return ""
}
func Preguntar(texto string) string {
	recortarHistorial(25, 12)
	refrescarHistorial()

	// Orden clara: se responde sin modelo (mas confiable y rapido).
	if respuesta := responderDirecto(texto); respuesta != "" {
		historial = append(historial,
			Mensaje{Role: "user", Content: texto},
			Mensaje{Role: "assistant", Content: respuesta})
		return respuesta
	}

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
