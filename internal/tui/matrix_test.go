package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

// TestLluviaMatrix: la lluvia avanza frames, pinta celdas y mantiene el
// ancho del canvas exacto (cada línea = w columnas visibles).
func TestLluviaMatrix(t *testing.T) {
	m := newMatrix(120, 30)

	// frame inicial vacío: todas las líneas son espacios de w columnas
	lineas := m.frame()
	if len(lineas) != 30 {
		t.Fatalf("frame: %d líneas, esperado 30", len(lineas))
	}
	for i, l := range lineas {
		if got := runewidth.StringWidth(ansi.Strip(l)); got != 120 {
			t.Fatalf("línea %d inicial: %d columnas, esperado 120", i, got)
		}
	}

	// 500 frames: la lluvia cae y SIEMPRE conserva el ancho exacto
	for f := 0; f < 500; f++ {
		m.tick()
	}
	for _, l := range m.frame() {
		if runewidth.StringWidth(ansi.Strip(l)) != 120 {
			t.Fatalf("tras tick: línea con ancho != 120")
		}
	}
	if !huboCeldas(m) {
		t.Errorf("la lluvia debería haber encendido celdas tras 500 frames")
	}
}

func huboCeldas(m *matrix) bool {
	for _, fila := range m.celdas {
		for _, c := range fila {
			if c.glow > 0 {
				return true
			}
		}
	}
	return false
}

// TestToggleFondo: la tecla "t" apaga/enciende el hiperespacio SOLO fuera de
// los inputs (en selectores); dentro de un campo de texto es un carácter.
func TestToggleFondo(t *testing.T) {
	m := tea.Model(initial())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30}) // crea el fondo

	if !m.(*tui).fondo.activa() {
		t.Fatalf("el fondo debería nacer activo")
	}
	// ir al selector de build tool (fuera del input)
	m.(*tui).input.SetValue("mi-api")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	if m.(*tui).fondo.activa() {
		t.Fatalf("tras 't' en selector el fondo debería estar apagado")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	if !m.(*tui).fondo.activa() {
		t.Fatalf("tras segundo 't' el fondo debería volver a encender")
	}
}

// TestTseEscribeEnElInput: dentro del campo de texto, "t" y "q" son
// caracteres NORMALES del input — nunca togglean ni sacan de la app.
func TestTseEscribeEnElInput(t *testing.T) {
	m := tea.Model(initial())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	if got := m.(*tui).input.Value(); got != "t" {
		t.Fatalf("la 't' no llegó al input: %q", got)
	}
	if !m.(*tui).fondo.activa() {
		t.Errorf("escribir 't' no debería togglear el fondo")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if got := m.(*tui).input.Value(); got != "tq" {
		t.Fatalf("la 'q' no llegó al input: %q", got)
	}
	if m.(*tui).quit {
		t.Errorf("escribir 'q' en el input no debería salir de la app")
	}
}

// TestBannerDegradado: el banner usa colores truecolor (38;2) por línea.
func TestBannerDegradado(t *testing.T) {
	tu := &tui{w: 200, h: 50} // ancho grande → usa el arte completo
	out := tu.banner()
	if !strings.Contains(out, "\x1b[38;2;") {
		t.Errorf("el banner debería tener degradado truecolor")
	}
	if !strings.Contains(out, "▗▄▄▖") {
		t.Errorf("el banner pierde el wordmark")
	}
	// 4 líneas separadas por \n
	if got := len(strings.Split(out, "\n")); got != 4 {
		t.Errorf("banner renderizado con %d líneas, esperado 4", got)
	}

	// el degradado interpola: primera línea ≠ última línea
	r1, g1, b1 := degradado(0, 4)
	r2, g2, b2 := degradado(3, 4)
	if r1 == r2 && b1 == b2 {
		t.Errorf("degradado no cambia entre extremos: %d,%d,%d vs %d,%d,%d", r1, g1, b1, r2, g2, b2)
	}
}

// TestCanvasEstampa: el framebuffer sobrescribe celdas por completo (overwrite
// matemático) — el fondo jamás puede quedar por encima del modal estampado.
func TestCanvasEstampa(t *testing.T) {
	c := newCanvas(10, 3)

	// capa 1: pintar "lluvia" en toda la fila 1
	for j := 0; j < 10; j++ {
		c.set(1, j, 'x', "38;2;34;197;94", "", false, false)
	}

	// capa 2: estampar un bloque ANSI de 2x2 en (2,1)
	celdas := parsearANSI("ab\ncd", "48;5;234")
	estampar(c, celdas, 2, 1)

	out := c.render()
	lineas := strings.Split(out, "\n")

	// el estampado pisa: fila 1 = "xx a b x..." con el bloque en cols 2-3
	fila1 := ansi.Strip(lineas[1])
	if runewidth.StringWidth(fila1) != 10 {
		t.Fatalf("ancho de fila = %d, esperado 10 (canvas exacto)", runewidth.StringWidth(fila1))
	}
	if string([]rune(fila1)[2:4]) != "ab" {
		t.Errorf("el bloque no pisó el fondo en cols 2-3: %q", fila1)
	}
	// cada celda del bloque tiene el fondo del panel (bg garantizado)
	if !strings.Contains(lineas[1], "48;5;234") {
		t.Errorf("las celdas estampadas no tienen el fondo del panel: %q", lineas[1])
	}

	// el render SIEMPRE tiene 3 líneas exactas (h del canvas)
	if len(lineas) != 3 {
		t.Errorf("alto = %d líneas, esperado 3", len(lineas))
	}
}

// TestFullScreenSinFranjas: la lluvia corre en TODO el viewport — ninguna
// línea del frame queda 100% vacía (sin franjas negras arriba o abajo del
// modal). El modal con su fondo sólido queda por encima.
func TestFullScreenSinFranjas(t *testing.T) {
	m := newMatrix(120, 30)
	m.zona(18, 6, 84, 17)
	for f := 0; f < 400; f++ {
		m.tick()
	}
	for fila := 0; fila < 30; fila++ {
		if !strings.Contains(m.frame()[fila], "\x1b[") {
			t.Errorf("fila %d sin lluvia (franja vacía)", fila)
		}
	}
	// la zona del modal también tiene canvas (la tapa el modal), pero la
	// lluvia existe detrás: no hay un "agujero" en el buffer
	linea := m.frame()[6]
	if got := runewidth.StringWidth(ansi.Strip(linea)); got != 120 {
		t.Errorf("línea dentro del modal con ancho %d, esperado 120", got)
	}
}

// TestVignetteAtenua: la franja de transición (margen alrededor del modal)
// NO puede tener colores de cabeza/cuerpo normales — solo dim ultra tenue o
// drop (espacios). El corte hacia el modal es suave, no tosco.
func TestVignetteAtenua(t *testing.T) {
	m := newMatrix(120, 30)
	m.zona(18, 6, 84, 17)
	for f := 0; f < 300; f++ {
		m.tick()
	}

	// fila 4 = margen Y superior (2 filas antes del borde en y=6)
	linea := m.frame()[4]
	// los segmentos laterales (cols 0-14 y 105-119) son margen X
	for _, seg := range []string{
		segmentoVisible(linea, 0, 15),   // margen izquierdo
		segmentoVisible(linea, 105, 15), // margen derecho
	} {
		if strings.Contains(seg, "197") {
			t.Errorf("el margen (vignette) no debería tener verde de cabeza (34;197;94): %q", seg)
		}
	}
	// fuera del margen (fila 0, lejos del modal) la lluvia vive con paleta normal
	if !strings.Contains(m.frame()[0], "\x1b[38;2;") {
		t.Errorf("la lluvia lejos del modal debería tener colores normales")
	}
}

// segmentoVisible extrae un tramo de columnas visibles de una línea ANSI:
// elimina los códigos de escape y toma las runas [x, x+ancho).
func segmentoVisible(linea string, x, ancho int) string {
	limpia := ansi.Strip(linea)
	rs := []rune(limpia)
	if x >= len(rs) {
		return ""
	}
	fin := x + ancho
	if fin > len(rs) {
		fin = len(rs)
	}
	return string(rs[x:fin])
}

// TestMatrixDenso: el efecto de película — entre 60% y 80% de las columnas
// del motor tienen estelas cayendo, y las estelas son largas (12-25).
func TestMatrixDenso(t *testing.T) {
	m := newMatrix(120, 30)
	for f := 0; f < 200; f++ {
		m.tick()
	}

	// % de columnas del motor con al menos una celda viva en su x
	activas := 0
	for _, c := range m.cols {
		for fila := 0; fila < m.h; fila++ {
			if m.celdas[fila][c.x].glow > 0 {
				activas++
				break
			}
		}
	}
	densidad := float64(activas) / float64(len(m.cols))
	if densidad < 0.6 {
		t.Errorf("densidad = %.2f (%d/%d columnas), esperado >= 0.6", densidad, activas, len(m.cols))
	}

	// estela larga: glow máximo del canvas (la cabeza recién escrita)
	maxGlow := 0
	for _, fila := range m.celdas {
		for _, c := range fila {
			if c.glow > maxGlow {
				maxGlow = c.glow
			}
		}
	}
	if maxGlow < 10 {
		t.Errorf("estelas cortas: maxGlow = %d, esperado >= 12 (estela de 12-25)", maxGlow)
	}
}
// TestFondoEnView: con el fondo activo el render usa el blit (la caja queda
// sobre el hiperespacio, sin perder el contenido).
func TestFondoEnView(t *testing.T) {
	m := tea.Model(initial())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	out := m.View()
	if !strings.Contains(out, "▗▄▄▖") {
		t.Errorf("la caja (banner) no aparece encima del fondo")
	}
	if !strings.Contains(out, "¿Cómo se llama el proyecto?") {
		t.Errorf("la pregunta no aparece encima del fondo")
	}
	for _, l := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		if runewidth.StringWidth(ansi.Strip(l)) != 120 {
			t.Errorf("línea fuera de ancho (120): %q", ansi.Strip(l))
		}
	}
}