package main

// Lector de PDF con la biblioteca estandar: busca los streams de contenido,
// los descomprime si usan FlateDecode y extrae el texto de los operadores
// Tj y TJ. Soporta PDFs simples (los que genera Vito, y muchos exportados
// sin comprimir); PDFs complejos (fuentes embebidas CID) pueden salir con
// texto parcial.

import (
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	reStream   = regexp.MustCompile(`(?s)stream\r?\n(.*?)endstream`)
	reTextoPDF = regexp.MustCompile(`\((?:[^()\\]|\\.)*\)\s*Tj|\[((?:[^()\\]|\\.)*)\]\s*TJ`)
	reLiteral  = regexp.MustCompile(`\(((?:[^()\\]|\\.)*)\)`)
)

// aWinAnsiInverso convierte los octales \ddd del PDF a caracteres reales
// (los mismos acentos que CrearPDF codifica).
var octalesPDF = map[string]rune{
	"341": 'á', "351": 'é', "355": 'í', "363": 'ó', "372": 'ú',
	"301": 'Á', "311": 'É', "315": 'Í', "323": 'Ó', "332": 'Ú',
	"361": 'ñ', "321": 'Ñ', "374": 'ü', "334": 'Ü',
	"277": '¿', "241": '¡', "260": '°',
}

func decodificarLiteralPDF(literal string) string {
	var b strings.Builder
	for i := 0; i < len(literal); i++ {
		c := literal[i]
		switch c {
		case '\\':
			i++
			if i >= len(literal) {
				break
			}
			siguiente := literal[i]
			if siguiente >= '0' && siguiente <= '7' {
				// Escape octal de hasta 3 digitos.
				fin := i
				for fin < len(literal) && fin < i+3 && literal[fin] >= '0' && literal[fin] <= '7' {
					fin++
				}
				octal := literal[i:fin]
				if caracter, ok := octalesPDF[octal]; ok {
					b.WriteRune(caracter)
				} else {
					if valor, err := strconv.ParseInt(octal, 8, 16); err == nil {
						b.WriteRune(rune(valor))
					}
				}
				i = fin - 1
			} else {
				b.WriteByte(siguiente)
			}
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// descomprimirStream infla un stream FlateDecode; si no es zlib, lo
// devuelve tal cual (stream sin comprimir).
func descomprimirStream(datos []byte) []byte {
	lector, err := zlib.NewReader(strings.NewReader(string(datos)))
	if err != nil {
		return datos
	}
	original, err := io.ReadAll(io.LimitReader(lector, 10<<20))
	if err != nil {
		return datos
	}
	return original
}

// extraerTextoPDF saca el texto de un contenido ya descomprimido.
func extraerTextoPDF(contenido string) []string {
	lineas := []string{}
	for _, coincidencia := range reTextoPDF.FindAllStringSubmatch(contenido, -1) {
		var trozos []string
		if coincidencia[1] == "" {
			// Operador Tj: un solo literal.
			if m := reLiteral.FindStringSubmatch(coincidencia[0]); m != nil {
				trozos = append(trozos, decodificarLiteralPDF(m[1]))
			}
		} else {
			// Operador TJ: varios literales en un arreglo.
			for _, m := range reLiteral.FindAllStringSubmatch(coincidencia[1], -1) {
				trozos = append(trozos, decodificarLiteralPDF(m[1]))
			}
		}
		linea := strings.Join(trozos, "")
		if strings.TrimSpace(linea) != "" {
			lineas = append(lineas, linea)
		}
	}
	return lineas
}

// leerPDF extrae el texto de un archivo PDF.
func leerPDF(ruta string) (string, error) {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return "", err
	}
	texto := string(datos)
	if !strings.Contains(texto, "%PDF-") {
		return "", fmt.Errorf("no parece un PDF")
	}
	lineas := []string{}
	for _, coincidencia := range reStream.FindAllStringSubmatch(texto, -1) {
		sinComprimir := descomprimirStream([]byte(coincidencia[1]))
		lineas = append(lineas, extraerTextoPDF(string(sinComprimir))...)
	}
	if len(lineas) == 0 {
		return "", fmt.Errorf("no pude extraer texto (puede tener fuentes embebidas o estar escaneado)")
	}
	return strings.Join(lineas, "\n"), nil
}
