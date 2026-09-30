package tui

import (
	"strings"
)

// cell es una celda del framebuffer: un carácter con sus atributos de color.
// fg/bg guardan la secuencia ANSI lista para concatenar ("38;5;214",
// "48;5;234", "38;2;34;197;94"...); "" = default.
type cell struct {
	ch    rune
	fg    string
	bg    string
	bold  bool
	faint bool
}

func (cl cell) defaultStyle() bool {
	return cl.fg == "" && cl.bg == "" && !cl.bold && !cl.faint
}

func (cl cell) mismoEstilo(o cell) bool {
	return cl.fg == o.fg && cl.bg == o.bg && cl.bold == o.bold && cl.faint == o.faint
}

// apertura devuelve la secuencia ANSI que abre el estilo ("" si es default).
func (cl cell) apertura() string {
	if cl.defaultStyle() {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\x1b[")
	if cl.bold {
		sb.WriteString("1;")
	}
	if cl.faint {
		sb.WriteString("2;")
	}
	if cl.fg != "" {
		sb.WriteString(cl.fg + ";")
	}
	if cl.bg != "" {
		sb.WriteString(cl.bg + ";")
	}
	return strings.TrimSuffix(sb.String(), ";") + "m"
}

// canvas es el framebuffer 2D de tamaño EXACTO termWidth x termHeight.
// Todo el render pasa por acá: fondo → estampado del modal → render ANSI.
type canvas struct {
	w, h   int
	celdas [][]cell
}

func newCanvas(w, h int) *canvas {
	c := &canvas{w: w, h: h}
	c.celdas = make([][]cell, h)
	for i := range c.celdas {
		c.celdas[i] = make([]cell, w)
		for j := range c.celdas[i] {
			c.celdas[i][j] = cell{ch: ' '}
		}
	}
	return c
}

// set escribe una celda (sobreescribiendo cualquier cosa previa: la lluvia
// jamás puede quedar por encima de lo que se estampa después).
func (c *canvas) set(y, x int, ch rune, fg, bg string, bold, faint bool) {
	if y < 0 || y >= c.h || x < 0 || x >= c.w {
		return
	}
	c.celdas[y][x] = cell{ch: ch, fg: fg, bg: bg, bold: bold, faint: faint}
}

// render: fila por fila, sin márgenes dinámicos. El alto del output es
// SIEMPRE h y el ancho SIEMPRE w — cero saltos al alternar el toggle.
//
// Render por CORRIDAS (run-length): celdas consecutivas con el MISMO estilo
// se escriben contiguas (el ANSI se abre una vez y cierra al final del run).
// Así el texto interior del modal permanece contiguo — los substrings no se
// rompen con secuencias intercaladas entre carácter y carácter.
func (c *canvas) render() string {
	lineas := make([]string, c.h)
	for i, fila := range c.celdas {
		var sb strings.Builder
		var abierta cell
		hayAbierta := false
		for _, cl := range fila {
			// mismo estilo que el run actual → solo el carácter, sin ANSI nuevo
			if hayAbierta && cl.mismoEstilo(abierta) {
				sb.WriteRune(cl.ch)
				continue
			}
			// cambio de estilo: cerrar el run anterior y abrir el nuevo
			if hayAbierta {
				sb.WriteString("\x1b[0m")
			}
			sb.WriteString(cl.apertura())
			sb.WriteRune(cl.ch)
			abierta, hayAbierta = cl, true
		}
		if hayAbierta {
			sb.WriteString("\x1b[0m")
		}
		lineas[i] = sb.String()
	}
	return strings.Join(lineas, "\n")
}

// pCell es una celda ya parseada de un render ANSI (modal/footer).
type pCell struct {
	ch    rune
	fg    string
	bg    string
	bold  bool
	faint bool
}

// parsearANSI convierte un render ANSI multilínea en celdas. Tolera
// secuencias 256/truecolor y los atributos que usamos (1, 2, 22, 23, 39,
// 49, 0). Las celdas sin background explícito reciben bgDefault — garantiza
// que TODAS las celdas interiores del modal tengan el gris del panel.
func parsearANSI(s string, bgDefault string) [][]pCell {
	fg, bg := "", ""
	bold, faint := false, false
	lineas := strings.Split(s, "\n")
	out := make([][]pCell, len(lineas))

	for i, l := range lineas {
		rs := []rune(l)
		celdas := make([]pCell, 0, len(rs))
		for k := 0; k < len(rs); {
			if rs[k] == '\x1b' {
				// encontrar el 'm'
				j := k + 1
				for j < len(rs) && rs[j] != 'm' {
					j++
				}
				sec := strings.Split(string(rs[k+1:j]), ";")
				procesarSec(sec, &fg, &bg, &bold, &faint)
				k = j + 1
				continue
			}
			b := bg
			if b == "" {
				b = bgDefault
			}
			celdas = append(celdas, pCell{ch: rs[k], fg: fg, bg: b, bold: bold, faint: faint})
			k++
		}
		out[i] = celdas
	}
	return out
}

// procesarSec aplica una secuencia ANSI (sin el ESC pero CON el '[') al
// estado. El primer token suele venir como "[38..." — se descarta el '['.
func procesarSec(toks []string, fg, bg *string, bold, faint *bool) {
	if len(toks) > 0 && strings.HasPrefix(toks[0], "[") {
		toks[0] = toks[0][1:]
	}
	i := 0
	for i < len(toks) {
		switch toks[i] {
		case "0":
			*fg, *bg = "", ""
			*bold, *faint = false, false
		case "1":
			*bold = true
		case "2":
			*faint = true
		case "22", "23":
			*bold, *faint = false, false
		case "39":
			*fg = ""
		case "49":
			*bg = ""
		case "38", "48": // 38;5;N o 38;2;r;g;b (o 38;5;N con atributos antes)
			if i+1 < len(toks) && toks[i+1] == "5" && i+2 < len(toks) {
				sec := toks[i] + ";5;" + toks[i+2]
				if toks[i] == "38" {
					*fg = sec
				} else {
					*bg = sec
				}
				i += 2
			} else if i+1 < len(toks) && toks[i+1] == "2" && i+4 < len(toks) {
				// truecolor: 38;2;r;g;b = EXACTAMENTE 5 tokens (índices 0..4)
				// → condición i+4 < len (antes i+5 < len cortaba el caso exacto
				// y el logo quedaba gris: los truecolor nunca se parseaban).
				sec := toks[i] + ";2;" + toks[i+2] + ";" + toks[i+3] + ";" + toks[i+4]
				if toks[i] == "38" {
					*fg = sec
				} else {
					*bg = sec
				}
				i += 4
			}
		}
		i++
	}
}

// estampar pisa el rectángulo del modal (o footer) sobre el canvas, celda a
// celda, con fondo de panel garantizado. Es una sobreescritura total: lo que
// había debajo (lluvia) desaparece matemáticamente.
func estampar(c *canvas, celdas [][]pCell, x, y int) {
	for by, fila := range celdas {
		for bx, pc := range fila {
			c.set(y+by, x+bx, pc.ch, pc.fg, pc.bg, pc.bold, pc.faint)
		}
	}
}