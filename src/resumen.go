package main

// Lectura de documentos (txt, docx) y resumenes con el modelo.

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// leerDocx extrae el texto de un Word: parrafo por parrafo.
func leerDocx(ruta string) (string, error) {
	archivo, err := zip.OpenReader(ruta)
	if err != nil {
		return "", err
	}
	defer archivo.Close()
	for _, entrada := range archivo.File {
		if entrada.Name != "word/document.xml" {
			continue
		}
		abierta, err := entrada.Open()
		if err != nil {
			return "", err
		}
		contenido, err := io.ReadAll(abierta)
		abierta.Close()
		if err != nil {
			return "", err
		}
		xml := string(contenido)
		// Cada parrafo es una linea; se quitan las etiquetas.
		xml = strings.ReplaceAll(xml, "</w:p>", "\n")
		etiquetas := regexp.MustCompile(`<[^>]+>`)
		texto := etiquetas.ReplaceAllString(xml, "")
		return decodificarEntidades(texto), nil
	}
	return "", fmt.Errorf("el documento no tiene word/document.xml")
}

func decodificarEntidades(s string) string {
	amp := string(rune(38))
	s = strings.ReplaceAll(s, amp+"lt;", "<")
	s = strings.ReplaceAll(s, amp+"gt;", ">")
	s = strings.ReplaceAll(s, amp+"quot;", "\"")
	s = strings.ReplaceAll(s, amp+"apos;", "'")
	s = strings.ReplaceAll(s, amp+"amp;", amp)
	return s
}

// LeerTextoDocumento lee txt o docx; para otros formatos devuelve error claro.
func LeerTextoDocumento(ruta string) (string, error) {
	ext := strings.ToLower(extDocumento(ruta))
	switch ext {
	case ".txt", ".md", ".log", ".csv", ".json", ".py", ".go", ".js", ".html", ".xml":
		contenido, err := os.ReadFile(ruta)
		if err != nil {
			return "", err
		}
		return string(contenido), nil
	case ".docx":
		return leerDocx(ruta)
	case ".pdf":
		return leerPDF(ruta)
	case ".xlsx":
		return leerXLSX(ruta)
	default:
		return "", fmt.Errorf("formato %s no soportado para resumir (usa .txt, .docx o .pdf)", ext)
	}
}

func extDocumento(ruta string) string {
	punto := -1
	for i := len(ruta) - 1; i >= 0; i-- {
		if ruta[i] == '.' {
			punto = i
			break
		}
		if ruta[i] == '\\' || ruta[i] == '/' {
			break
		}
	}
	if punto < 0 {
		return ""
	}
	return ruta[punto:]
}

// ResumirDocumento lee el documento y pide el resumen al modelo.
func ResumirDocumento(ruta string) string {
	texto, err := LeerTextoDocumento(ruta)
	if err != nil {
		return fmt.Sprintf("No pude leer %s: %v", ruta, err)
	}
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return fmt.Sprintf("%s esta vacio.", ruta)
	}
	maximo := 12000
	if len(texto) > maximo {
		texto = texto[:maximo] + "\n...(texto truncado)"
	}
	mensajes := []Mensaje{
		{Role: "system", Content: "Resumes documentos. El documento siempre viene dentro de las etiquetas <DOC> del mensaje del usuario. Responde solo con el resumen, en espanol, de 2 a 3 frases, sin listas y sin pedir mas informacion."},
		{Role: "user", Content: "Texto del documento:\n<DOC>\n" + texto + "\n</DOC>\nEscribe un resumen del contenido entre las etiquetas DOC."},
	}
	respuesta, err := Chatear(Modelo, mensajes, nil, 0.3)
	if err != nil {
		return fmt.Sprintf("No pude pedir el resumen al modelo: %v", err)
	}
	resumen := strings.TrimSpace(respuesta.Content)
	if resumen == "" {
		return "El modelo no devolvio resumen."
	}
	return "Resumen de " + ruta + ": " + resumen
}
