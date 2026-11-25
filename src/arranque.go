package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	colorReset = "\x1b[0m"
	colorCian  = "\x1b[36m"
	colorVerde = "\x1b[32m"
	colorGris  = "\x1b[90m"
)

var bannerVito = []string{
	`██╗   ██╗██╗████████╗ ██████╗ `,
	`██║   ██║██║╚══██╔══╝██╔═══██╗`,
	`██║   ██║██║   ██║   ██║   ██║`,
	`╚██╗ ██╔╝██║   ██║   ██║   ██║`,
	` ╚████╔╝ ██║   ██║   ╚██████╔╝`,
	`  ╚═══╝  ╚═╝   ╚═╝    ╚═════╝ `,
}

var consolaVT bool

func prepararConsola() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	kernel32.NewProc("SetConsoleOutputCP").Call(65001)
	handle := syscall.Handle(os.Stdout.Fd())
	var modo uint32
	r, _, _ := kernel32.NewProc("GetConsoleMode").Call(uintptr(handle), uintptr(unsafe.Pointer(&modo)))
	if r == 0 {
		return
	}

	r, _, _ = kernel32.NewProc("SetConsoleMode").Call(uintptr(handle), uintptr(modo|0x0004))
	consolaVT = r != 0
}

func saludoPorHora() string {
	ahora := time.Now()
	saludo := "Buenas noches"
	if h := ahora.Hour(); h >= 5 && h < 12 {
		saludo = "Buenos dias"
	} else if h >= 12 && h < 19 {
		saludo = "Buenas tardes"
	}

	if usuario := os.Getenv("USERNAME"); usuario != "" {
		saludo += ", " + usuario
	}
	return fmt.Sprintf("%s. Son las %s", saludo, ahora.Format("15:04"))
}

func dibujarBanner() {
	if !consolaVT {
		fmt.Println(strings.Join(bannerVito, "\n"))
		return
	}
	lineas := make([][]rune, len(bannerVito))
	ancho := 0
	for i, l := range bannerVito {
		lineas[i] = []rune(l)
		if len(lineas[i]) > ancho {
			ancho = len(lineas[i])
		}
	}
	for i := range lineas {
		for len(lineas[i]) < ancho {
			lineas[i] = append(lineas[i], ' ')
		}
	}
	for col := 1; col <= ancho; col++ {
		for _, l := range lineas {
			fmt.Printf("%s%s%s\n", colorCian, string(l[:col]), colorReset)
		}
		fmt.Printf("\x1b[%dA", len(lineas))
		time.Sleep(18 * time.Millisecond)
	}
	fmt.Printf("\x1b[%dB\n", len(lineas))
}

func dibujarBarra(ancho int, pct float64, texto string) {
	llenas := int(pct * float64(ancho))
	barra := strings.Repeat("█", llenas) + strings.Repeat("░", ancho-llenas)
	fmt.Printf("\r\x1b[2K%s[%s]%s %3d%%  %s%s%s",
		colorVerde, barra, colorReset, int(pct*100), colorGris, texto, colorReset)
}

func animarProgreso(listo chan struct{}) {
	if !consolaVT {
		fmt.Println("Indexando en segundo plano...")
		<-listo
		return
	}
	total := ConteoFirmas()
	inicio := time.Now()
	const ancho = 30
	for {
		select {
		case <-listo:
			dibujarBarra(ancho, 1.0, fmt.Sprintf("indice listo (%d archivos)", ConteoFilas()))
			fmt.Println()
			return
		case <-time.After(6 * time.Second):
			// No bloquear la terminal: el escaneo sigue en su goroutine.
			dibujarBarra(ancho, 0.99, "sigo indexando en segundo plano...")
			fmt.Println()
			return
		default:
		}
		escaneadas := carpetasEscaneadas.Load()
		var pct float64
		if total > 0 {
			pct = float64(escaneadas) / float64(total)
			if pct > 0.99 {
				pct = 0.99
			}
		} else {
			pct = 1 - math.Exp(-time.Since(inicio).Seconds()/20)
			if pct > 0.95 {
				pct = 0.95
			}
		}
		dibujarBarra(ancho, pct, fmt.Sprintf("indexando... %d carpetas", escaneadas))
		time.Sleep(80 * time.Millisecond)
	}
}

func AnimarArranque(escanear func()) {
	prepararConsola()
	dibujarBanner()
	fmt.Println()
	fmt.Println(saludoPorHora())
	fmt.Println()
	if escanear == nil {
		fmt.Println(fmt.Sprintf("Vito v%s listo. Escribe una orden, 'ayuda' o 'apagar' para salir.", VersionActual))
		return
	}
	listo := make(chan struct{})
	go func() {
		escanear()
		close(listo)
	}()
	animarProgreso(listo)
	fmt.Println(fmt.Sprintf("Vito v%s listo. Escribe una orden, 'ayuda' o 'apagar' para salir.", VersionActual))
}
