package tui

import "testing"

// TestParseTruecolor: el parser ANSI DEBE capturar los truecolor de 5 tokens
// (38;2;r;g;b) — bug previo doble: (1) i+5 < len cortaba el caso exacto y
// (2) el '[' del inicio de la secuencia no se descartaba ("[38" != "38") →
// el logo spring-box terminaba gris. La secuencia se pasa CON el '[' (caso real).
func TestParseTruecolor(t *testing.T) {
	fg, bg := "", ""
	bold, faint := false, false

	// degradado del banner tal como lo emite banner(): \x1b[38;2;140;255;120
	procesarSec([]string{"[38", "2", "140", "255", "120"}, &fg, &bg, &bold, &faint)
	if fg != "38;2;140;255;120" {
		t.Fatalf("truecolor FG = %q, esperado 38;2;140;255;120", fg)
	}

	// el cierre "39" resetea el foreground
	procesarSec([]string{"[39", "22", "23"}, &fg, &bg, &bold, &faint)
	if fg != "" {
		t.Fatalf("tras '39' el fg debería quedar vacío, quedó %q", fg)
	}
}

// TestBannerMantieneTruecolor: el banner parseado conserva el degradado en
// las celdas del canvas (no queda gris).
func TestBannerMantieneTruecolor(t *testing.T) {
	celdas := parsearANSI((&tui{w: 200, h: 50}).banner(), panelBG)
	// el banner son 4 líneas; la primera línea tiene chars con truecolor
	primera := celdas[0]
	algunaConColor := false
	for _, c := range primera {
		if c.fg != "" {
			algunaConColor = true
			if c.fg[0:5] != "38;2;" {
				t.Fatalf("color del banner no es truecolor: %q", c.fg)
			}
		}
	}
	if !algunaConColor {
		t.Errorf("el banner no tiene ningún carácter con color (logo gris)")
	}
}