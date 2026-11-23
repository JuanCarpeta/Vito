package main

// Creacion de documentos reales (.docx, .xlsx, .pdf) con la biblioteca
// estandar: OOXML es un ZIP de XML y el PDF minimo se escribe a mano.

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func escribirZip(ruta string, archivos map[string]string) error {
	archivo, err := os.Create(ruta)
	if err != nil {
		return err
	}
	defer archivo.Close()
	compresor := zip.NewWriter(archivo)
	for nombre, contenido := range archivos {
		entrada, err := compresor.Create(nombre)
		if err != nil {
			return err
		}
		if _, err := entrada.Write([]byte(contenido)); err != nil {
			return err
		}
	}
	return compresor.Close()
}

func escaparXML(s string) string {
	return strings.NewReplacer(
		"&", "&amp;", "<", "&lt;", ">", "&gt;",
	).Replace(s)
}

// CrearDOCX genera un Word minimo con titulo y parrafos.
func CrearDOCX(ruta, titulo, texto string) error {
	if !strings.EqualFold(filepath.Ext(ruta), ".docx") {
		ruta += ".docx"
	}
	cuerpo := ""
	if titulo != "" {
		cuerpo += `<w:p><w:r><w:rPr><w:b/><w:sz w:val="32"/></w:rPr>` +
			`<w:t>` + escaparXML(titulo) + `</w:t></w:r></w:p>`
	}
	for _, linea := range strings.Split(strings.ReplaceAll(texto, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(linea) == "" {
			cuerpo += `<w:p/>`
			continue
		}
		cuerpo += `<w:p><w:r><w:t xml:space="preserve">` + escaparXML(linea) + `</w:t></w:r></w:p>`
	}
	archivos := map[string]string{
		"[Content_Types].xml": `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`,
		"_rels/.rels":         `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`,
		"word/document.xml": `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
			cuerpo + `</w:body></w:document>`,
	}
	return escribirZip(ruta, archivos)
}

// CrearXLSX genera un Excel minimo: cada linea es una fila y las celdas
// se separan por punto y coma o tabulacion.
func CrearXLSX(ruta, texto string) error {
	if !strings.EqualFold(filepath.Ext(ruta), ".xlsx") {
		ruta += ".xlsx"
	}
	filas := ""
	numero := 1
	lineas := strings.Split(strings.ReplaceAll(texto, "\r\n", "\n"), "\n")
	if strings.TrimSpace(texto) == "" {
		lineas = []string{""}
	}
	for _, linea := range lineas {
		celdasXML := ""
		for i, celda := range strings.Split(strings.ReplaceAll(linea, "\t", ";"), ";") {
			celdasXML += fmt.Sprintf(`<c r="%s%d" t="inlineStr"><is><t>%s</t></is></c>`,
				columnaExcel(i), numero, escaparXML(celda))
		}
		filas += fmt.Sprintf(`<row r="%d">%s</row>`, numero, celdasXML)
		numero++
	}
	archivos := map[string]string{
		"[Content_Types].xml":        `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`,
		"_rels/.rels":                `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":            `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Hoja1" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/worksheets/sheet1.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>` +
			filas + `</sheetData></worksheet>`,
	}
	return escribirZip(ruta, archivos)
}

func columnaExcel(i int) string {
	letra := string(rune('A' + i%26))
	for i >= 26 {
		i = i/26 - 1
		letra = string(rune('A'+i%26)) + letra
	}
	return letra
}

// aPDF convierte acentos a octales WinAnsi y escapa los literales de PDF.
var tablaCP1252 = map[rune]string{
	'á': "\\341", 'é': "\\351", 'í': "\\355", 'ó': "\\363", 'ú': "\\372",
	'Á': "\\301", 'É': "\\311", 'Í': "\\315", 'Ó': "\\323", 'Ú': "\\332",
	'ñ': "\\361", 'Ñ': "\\321", 'ü': "\\374", 'Ü': "\\334",
	'¿': "\\277", '¡': "\\241", '°': "\\260",
}

func aPDF(s string) string {
	var b strings.Builder
	for _, r := range s {
		if octal, ok := tablaCP1252[r]; ok {
			b.WriteString(octal)
		} else if r < 128 {
			b.WriteRune(r)
		} else {
			b.WriteRune('?')
		}
	}
	return strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)").
		Replace(b.String())
}

// CrearPDF genera un PDF minimo de una pagina con titulo y parrafos.
func CrearPDF(ruta, titulo, texto string) error {
	if !strings.EqualFold(filepath.Ext(ruta), ".pdf") {
		ruta += ".pdf"
	}
	lineas := []string{}
	if titulo != "" {
		lineas = append(lineas, "BT /F2 18 Tf 72 760 Td ("+aPDF(titulo)+") Tj ET")
	}
	y := 720.0
	for _, linea := range strings.Split(strings.ReplaceAll(texto, "\r\n", "\n"), "\n") {
		if y < 50 {
			break // una pagina por ahora
		}
		lineas = append(lineas, fmt.Sprintf("BT /F1 11 Tf 72 %.0f Td (%s) Tj ET", y, aPDF(linea)))
		y -= 16
	}
	contenido := strings.Join(lineas, "\n")
	pdf := fmt.Sprintf("%%PDF-1.4\n"+
		"1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n"+
		"2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n"+
		"3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] "+
		"/Resources << /Font << /F1 4 0 R /F2 5 0 R >> >> /Contents 6 0 R >> endobj\n"+
		"4 0 obj << /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >> endobj\n"+
		"5 0 obj << /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >> endobj\n"+
		"6 0 obj << /Length %d >> stream\n%s\nendstream endobj\n"+
		"trailer << /Root 1 0 R /Size 7 >>\n", len(contenido), contenido)
	return os.WriteFile(ruta, []byte(pdf), 0o644)
}
