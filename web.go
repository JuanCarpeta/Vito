package main

// Busqueda web en DuckDuckGo con la biblioteca estandar.

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const urlBusqueda = "https://html.duckduckgo.com/html/?q="

var (
	reEnlace = regexp.MustCompile(
		`<a[^>]*class="result__a"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	reResumen = regexp.MustCompile(
		`<a[^>]*class="result__snippet"[^>]*>(.*?)</a>`)
	reEtiqueta = regexp.MustCompile(`<[^>]+>`)
)

// urlReal decodifica los enlaces que DuckDuckGo envuelve en /l/?uddg=...
func urlReal(cruda string) string {
	if cruda == "" {
		return ""
	}
	if u, err := url.Parse(cruda); err == nil {
		if d := u.Query().Get("uddg"); d != "" {
			if real, err := url.QueryUnescape(d); err == nil {
				return real
			}
		}
	}
	if strings.HasPrefix(cruda, "//") {
		return "https:" + cruda
	}
	return cruda
}

func limpiarHTML(s string) string {
	return html.UnescapeString(strings.TrimSpace(reEtiqueta.ReplaceAllString(s, "")))
}

// BuscarWeb devuelve texto formateado con los resultados o un error claro.
func BuscarWeb(consulta string) string {
	if strings.TrimSpace(consulta) == "" {
		return "Falta la consulta de busqueda web."
	}
	cliente := &http.Client{Timeout: 10 * time.Second}
	peticion, _ := http.NewRequest("GET", urlBusqueda+url.QueryEscape(consulta), nil)
	peticion.Header.Set("User-Agent",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/124.0 Safari/537.36")
	resp, err := cliente.Do(peticion)
	if err != nil {
		return fmt.Sprintf("No pude conectar con internet: %v", err)
	}
	defer resp.Body.Close()
	cuerpo, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return fmt.Sprintf("No pude leer la respuesta: %v", err)
	}
	pagina := string(cuerpo)

	enlaces := reEnlace.FindAllStringSubmatch(pagina, -1)
	resumenes := reResumen.FindAllStringSubmatch(pagina, -1)
	if len(enlaces) == 0 {
		return "No encontre resultados en la web para esa consulta."
	}
	lineas := []string{}
	maximo := 5
	if len(enlaces) < maximo {
		maximo = len(enlaces)
	}
	for i := 0; i < maximo; i++ {
		titulo := limpiarHTML(enlaces[i][2])
		enlace := urlReal(enlaces[i][1])
		resumen := ""
		if i < len(resumenes) {
			resumen = limpiarHTML(resumenes[i][1])
			if len(resumen) > 300 {
				resumen = resumen[:300]
			}
		}
		linea := fmt.Sprintf("%d. %s - %s", i+1, titulo, enlace)
		if resumen != "" {
			linea += " | " + resumen
		}
		lineas = append(lineas, linea)
	}
	return fmt.Sprintf("Encontre %d resultado(s) en la web:\n%s", maximo, strings.Join(lineas, "\n"))
}
