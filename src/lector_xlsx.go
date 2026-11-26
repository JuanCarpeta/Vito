package main

// Lector de Excel (.xlsx) con la biblioteca estandar: mismo formato OOXML
// que los Word (un ZIP de XML). Soporta celdas con texto en linea, numeros
// y texto compartido (sharedStrings), que es lo que generan Excel, Vito y
// Google Sheets al exportar.

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	reFilaXlsx    = regexp.MustCompile(`<row[^>]*>(.*?)</row>`)
	reCeldaXlsx   = regexp.MustCompile(`<c\b([^>]*)>(.*?)</c>|<c\b([^>]*)/>`)
	reTextoCelda  = regexp.MustCompile(`<t[^>]*>(.*?)</t>`)
	reValorCelda  = regexp.MustCompile(`<v>(.*?)</v>`)
	reColumnaXlsx = regexp.MustCompile(`r="([A-Z]+)(\d+)"`)
)

// leerXLSX extrae el texto de la primera hoja, una linea por fila y las
// celdas separadas por punto y coma (el mismo formato que entiende CrearXLSX).
func leerXLSX(ruta string) (string, error) {
	archivo, err := zip.OpenReader(ruta)
	if err != nil {
		return "", err
	}
	defer archivo.Close()

	var compartidos []string
	var hojas []string
	for _, entrada := range archivo.File {
		switch {
		case strings.EqualFold(entrada.Name, "xl/sharedStrings.xml"):
			contenido, err := contenidoEntrada(entrada)
			if err != nil {
				return "", err
			}
			// Cada <si> puede tener varios <t> (texto con saltos).
			for _, bloque := range regexp.MustCompile(`<si>.*?</si>`).FindAllString(contenido, -1) {
				var trozos []string
				for _, m := range reTextoCelda.FindAllStringSubmatch(bloque, -1) {
					trozos = append(trozos, decodificarEntidades(m[1]))
				}
				compartidos = append(compartidos, strings.Join(trozos, ""))
			}
		case strings.HasPrefix(strings.ToUpper(entrada.Name), "XL/WORKSHEETS/"):
			contenido, err := contenidoEntrada(entrada)
			if err != nil {
				return "", err
			}
			hojas = append(hojas, contenido)
		}
	}
	if len(hojas) == 0 {
		return "", fmt.Errorf("el archivo no tiene hojas")
	}
	// Solo la primera hoja: es la que Excel abre por defecto.
	return extraerHojas(hojas[0], compartidos), nil
}

func contenidoEntrada(entrada *zip.File) (string, error) {
	abierta, err := entrada.Open()
	if err != nil {
		return "", err
	}
	defer abierta.Close()
	contenido, err := io.ReadAll(io.LimitReader(abierta, 20<<20))
	if err != nil {
		return "", err
	}
	return string(contenido), nil
}

func extraerHojas(hoja string, compartidos []string) string {
	var lineas []string
	for _, filaXML := range reFilaXlsx.FindAllStringSubmatch(hoja, -1) {
		celdas := map[int]string{}
		mayor := -1
		for _, celda := range reCeldaXlsx.FindAllStringSubmatch(filaXML[1], -1) {
			atributos, cuerpo := celda[1]+celda[3], celda[2]
			posicion := reColumnaXlsx.FindStringSubmatch(atributos)
			if posicion == nil {
				continue
			}
			indice := letraAColumna(posicion[1])
			valor := valorDeCelda(atributos, cuerpo, compartidos)
			celdas[indice] = valor
			if indice > mayor {
				mayor = indice
			}
		}
		if mayor < 0 {
			continue
		}
		fila := make([]string, mayor+1)
		vacias := 0
		for i := 0; i <= mayor; i++ {
			fila[i] = celdas[i]
			if fila[i] == "" {
				vacias++
			}
		}
		if vacias == len(fila) {
			continue // fila completamente vacia
		}
		lineas = append(lineas, strings.Join(fila, ";"))
	}
	if len(lineas) == 0 {
		return ""
	}
	return strings.Join(lineas, "\n")
}

// valorDeCelda resuelve el contenido de una celda segun su tipo.
func valorDeCelda(atributos, cuerpo string, compartidos []string) string {
	texto := ""
	if m := reValorCelda.FindStringSubmatch(cuerpo); m != nil {
		texto = strings.TrimSpace(m[1])
	}
	if m := reTextoCelda.FindStringSubmatch(cuerpo); m != nil {
		texto = decodificarEntidades(m[1])
	}
	if strings.Contains(atributos, `t="s"`) {
		indice, err := strconv.Atoi(texto)
		if err == nil && indice < len(compartidos) {
			return compartidos[indice]
		}
		return ""
	}
	if strings.Contains(atributos, `t="str"`) || strings.Contains(atributos, `t="e"`) {
		return decodificarEntidades(texto)
	}
	// Numeros y demas: tal cual.
	return decodificarEntidades(texto)
}

// letraAColumna convierte "A"->0, "B"->1, ... "AA"->26.
func letraAColumna(letra string) int {
	indice := 0
	for _, c := range letra {
		indice = indice*26 + int(c-'A'+1)
	}
	return indice - 1
}

var _ = filepath.Ext // reservado para futuras verificaciones de extension
