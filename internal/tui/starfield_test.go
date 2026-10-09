package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

// TestStarfieldAncho: el hiperespacio avanza frames y el canvas conserva el
// ancho exacto (cada línea = w columnas visibles) — mismo contrato que el
// framebuffer del modal, para que el render no salte.
func TestStarfieldAncho(t *testing.T) {
	s := newStarfield(120, 30)
	for f := 0; f < 300; f++ {
		s.tick()
	}
	lineas := s.frame()
	if len(lineas) != 30 {
		t.Fatalf("frame: %d líneas, esperado 30", len(lineas))
	}
	for i, l := range lineas {
		if got := runewidth.StringWidth(ansi.Strip(l)); got != 120 {
			t.Fatalf("línea %d: %d columnas, esperado 120", i, got)
		}
	}
}

// TestStarfieldPinta: tras varios frames hay estrellas dibujadas (truecolor).
func TestStarfieldPinta(t *testing.T) {
	s := newStarfield(120, 30)
	for f := 0; f < 200; f++ {
		s.tick()
	}
	pintado := false
	for _, l := range s.frame() {
		if strings.Contains(l, "\x1b[38;2;") {
			pintado = true
			break
		}
	}
	if !pintado {
		t.Errorf("el hiperespacio no pintó ninguna estrella tras 200 frames")
	}
}

// TestStarfieldToggle: activa()/toggle() reflejan el estado del fondo.
func TestStarfieldToggle(t *testing.T) {
	s := newStarfield(80, 24)
	if !s.activa() {
		t.Fatalf("el fondo debería nacer activo")
	}
	s.toggle()
	if s.activa() {
		t.Fatalf("tras toggle el fondo debería estar apagado")
	}
	s.toggle()
	if !s.activa() {
		t.Fatalf("tras segundo toggle el fondo debería volver a encender")
	}
}
