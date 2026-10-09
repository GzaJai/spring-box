package tui

import (
	"fmt"
	"strings"
)

// bannerASCII es el título principal (wordmark de bloques redondeados).
// Las líneas pueden traer espacios al final: se limpian al renderizar.
const bannerASCII = `
  ▗▄▄▖▗▄▄▖ ▗▄▄▖ ▗▄▄▄▖▗▖  ▗▖ ▗▄▄▖        ▗▄▄▖  ▗▄▖ ▗▖  ▗▖
▐▌   ▐▌ ▐▌▐▌ ▐▌  █  ▐▛▚▖▐▌▐▌           ▐▌ ▐▌▐▌ ▐▌ ▝▚▞▘
▝▀▚▖▐▛▀▘ ▐▛▀▚▖  █  ▐▌ ▝▜▌▐▌▝▜▌        ▐▛▀▚▖▐▌ ▐▌  ▐▌
 ▗▄▄▞▘▐▌   ▐▌ ▐▌▗▄█▄▖▐▌  ▐▌▝▚▄▞▘        ▐▙▄▞▘▝▚▄▞▘▗▞▘▝▚▖`

// anchoMinimoModal: ancho mínimo de la columna de contenido — la tabla del
// Resumen manda (etiqueta + separador + valor). El modal nunca baja de acá.
const anchoMinimoModal = anchoEtiqueta + 2 + anchoValor

// anchoModal: ancho interior del modal = el mayor entre el banner y el
// contenido mínimo (así el banner nunca recorta la tabla del Resumen).
func (t *tui) anchoModal() int {
	return max(anchoBanner(), anchoMinimoModal)
}

// banner renderiza el wordmark con el degradé truecolor verde → blanco por
// línea. Cierre fg-only (39): el fondo del panel sobrevive (fix bloques negros).
func (t *tui) banner() string {
	lineas := strings.Split(limpiarBanner(bannerASCII), "\n")
	n := len(lineas)
	var sb strings.Builder
	for i, l := range lineas {
		r, g, b := degradado(i, n)
		sb.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[39m", r, g, b, l))
		if i < n-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
