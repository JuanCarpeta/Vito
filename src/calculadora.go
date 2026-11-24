package main

// Calculadora segura: evalua expresiones aritmeticas escritas en lenguaje
// natural o simbolico, sin evaluar codigo arbitrario.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	rePalabras = strings.NewReplacer(
		"mas", "+", "menos", "-", "por", "*", "equis", "*",
		"dividido", "/", "entre", "/", "elevado a", "^", "elevado", "^",
		"raiz de", "raiz", "(", "(", ")", ")",
	)
	reValida = regexp.MustCompile(`^[0-9+\-*/^().,\s]+$`)
)

// EvaluarExpresion resuelve una expresion aritmetica y devuelve el texto.
func EvaluarExpresion(expresion string) string {
	limpia := strings.ToLower(strings.TrimSpace(expresion))
	limpia = rePalabras.Replace(limpia)
	limpia = strings.ReplaceAll(limpia, ",", ".")
	limpia = strings.Join(strings.Fields(limpia), "")
	if limpia == "" || !reValida.MatchString(limpia) {
		return ""
	}
	if !strings.ContainsAny(limpia, "+-*/^") {
		return "" // no es una operacion
	}
	valor, err := evaluar(limpia)
	if err != nil {
		return ""
	}
	return formatearNumero(valor)
}

func formatearNumero(valor float64) string {
	if valor == float64(int64(valor)) {
		return fmt.Sprintf("%d", int64(valor))
	}
	return fmt.Sprintf("%.4g", valor)
}

// analizador descendente: expresion -> termino (+|- termino)*
type analizador struct {
	texto string
	pos   int
}

func evaluar(texto string) (float64, error) {
	a := &analizador{texto: texto}
	resultado, err := a.expresion()
	if err != nil {
		return 0, err
	}
	if a.pos < len(a.texto) {
		return 0, fmt.Errorf("caracter inesperado en la posicion %d", a.pos)
	}
	return resultado, nil
}

func (a *analizador) expresion() (float64, error) {
	valor, err := a.termino()
	if err != nil {
		return 0, err
	}
	for a.pos < len(a.texto) {
		switch a.texto[a.pos] {
		case '+':
			a.pos++
			derecha, err := a.termino()
			if err != nil {
				return 0, err
			}
			valor += derecha
		case '-':
			a.pos++
			derecha, err := a.termino()
			if err != nil {
				return 0, err
			}
			valor -= derecha
		default:
			return valor, nil
		}
	}
	return valor, nil
}

func (a *analizador) termino() (float64, error) {
	valor, err := a.potencia()
	if err != nil {
		return 0, err
	}
	for a.pos < len(a.texto) {
		switch a.texto[a.pos] {
		case '*':
			a.pos++
			derecha, err := a.potencia()
			if err != nil {
				return 0, err
			}
			valor *= derecha
		case '/':
			a.pos++
			derecha, err := a.potencia()
			if err != nil {
				return 0, err
			}
			if derecha == 0 {
				return 0, fmt.Errorf("division por cero")
			}
			valor /= derecha
		default:
			return valor, nil
		}
	}
	return valor, nil
}

func (a *analizador) potencia() (float64, error) {
	base, err := a.primario()
	if err != nil {
		return 0, err
	}
	if a.pos < len(a.texto) && a.texto[a.pos] == '^' {
		a.pos++
		exponente, err := a.potencia()
		if err != nil {
			return 0, err
		}
		return potenciaEntera(base, exponente), nil
	}
	return base, nil
}

func (a *analizador) primario() (float64, error) {
	if a.pos < len(a.texto) && a.texto[a.pos] == '(' {
		a.pos++
		valor, err := a.expresion()
		if err != nil {
			return 0, err
		}
		if a.pos >= len(a.texto) || a.texto[a.pos] != ')' {
			return 0, fmt.Errorf("falta cerrar parentesis")
		}
		a.pos++
		return valor, nil
	}
	if a.pos < len(a.texto) && (a.texto[a.pos] == '+' || a.texto[a.pos] == '-') {
		signo := 1.0
		if a.texto[a.pos] == '-' {
			signo = -1
		}
		a.pos++
		valor, err := a.primario()
		return signo * valor, err
	}
	inicio := a.pos
	for a.pos < len(a.texto) && (a.texto[a.pos] >= '0' && a.texto[a.pos] <= '9' ||
		a.texto[a.pos] == '.') {
		a.pos++
	}
	if inicio == a.pos {
		return 0, fmt.Errorf("esperaba un numero en la posicion %d", a.pos)
	}
	return strconv.ParseFloat(a.texto[inicio:a.pos], 64)
}

func potenciaEntera(base, exponente float64) float64 {
	if exponente != float64(int(exponente)) {
		return base // sin decimales raros: devolvemos la base
	}
	resultado := 1.0
	for i := 0; i < int(abs(exponente)); i++ {
		resultado *= base
	}
	if exponente < 0 && base != 0 {
		return 1 / resultado
	}
	return resultado
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// reCalculo detecta pedidos de calculo en lenguaje natural.
var reCalculo = regexp.MustCompile(
	`(?i)^\s*(?:cuanto es|cuanto es el resultado de|calcula(?:r)?(?:me)?(?: el)?(?: resultado de)?|resultado de|resuelve)\s+(.+?)\s*[?=]?\s*$`)

// responderCalculo intenta resolver una cuenta; "" si no parece calculo.
func responderCalculo(texto string) string {
	coincidencia := reCalculo.FindStringSubmatch(texto)
	if coincidencia == nil {
		return ""
	}
	// Solo si la expresion contiene numeros y una operacion.
	expresion := coincidencia[1]
	if !regexp.MustCompile(`\d`).MatchString(expresion) {
		return ""
	}
	resultado := EvaluarExpresion(expresion)
	if resultado == "" {
		return ""
	}
	return fmt.Sprintf("%s = %s", expresion, resultado)
}
