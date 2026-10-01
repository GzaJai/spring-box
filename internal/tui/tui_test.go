package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"

	"springy/internal/model"
)

// press envía una tecla y devuelve el modelo con la assertion de tipo hecha.
func press(m tea.Model, k tea.KeyMsg) *tui {
	next, _ := m.Update(k)
	return next.(*tui)
}

// TestViewCentrada: el render ocupa todo el viewport (pantalla completa) y el
// contenido se re-centra al redimensionar la terminal.
func TestViewCentrada(t *testing.T) {
	m := tea.Model(initial())

	// resize: la terminal pasa a 120x30 → el viewport debe ser de 120x30
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	out := m.View()
	lineas := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lineas) != 30 {
		t.Fatalf("alto del render = %d líneas, esperado 30", len(lineas))
	}
	for i, l := range lineas {
		if got := runewidth.StringWidth(ansi.Strip(l)); got != 120 {
			t.Errorf("línea %d: ancho visible = %d columnas, esperado 120", i, got)
		}
	}

	// la jerarquía visual se preserva dentro del viewport: banner rosa arriba,
	// subtítulo tenue, pregunta naranja, input con prompt "> " y footer fuera
	// del contenedor
	for _, esperado := range []string{
		"▄███████▄", // primera línea del banner ASCII (SPRINGY)
		"arrancá tu proyecto Spring en local",
		"¿Cómo se llama el proyecto?",
		"> mi-api", // prompt + placeholder del input
		"enter: continuar · ctrl+c: salir", // en el input NO se anuncia "t" (es un carácter escribible)
	} {
		if !strings.Contains(out, esperado) {
			t.Errorf("falta en el render: %q", esperado)
		}
	}
}

// TestBannerAscii: el bloque del banner tiene las 6 líneas y define el ancho
// de la caja (≈80 columnas).
func TestBannerAscii(t *testing.T) {
	lineas := strings.Split(bannerASCII, "\n")
	if len(lineas) != 6 {
		t.Fatalf("el banner debe tener 6 líneas, tiene %d", len(lineas))
	}
	ancho := anchoBanner()
	if ancho < 70 {
		t.Errorf("banner demasiado angosto: %d columnas, esperado ≈80", ancho)
	}
	// sin espacios finales: la caja queda exacta y simétrica
	limpio := limpiarBanner(bannerASCII)
	if limpio != strings.TrimRight(limpio, " \n") {
		t.Errorf("el banner limpio no debería tener espacios finales")
	}
	for i, l := range strings.Split(limpio, "\n") {
		if strings.HasSuffix(l, " ") {
			t.Errorf("línea %d del banner limpio tiene espacio final", i)
		}
	}
}

// TestViewDefaultSinResize: sin WindowSizeMsg los defaults 80x24 no rompen.
func TestViewDefaultSinResize(t *testing.T) {
	m := initial()
	out := m.View()
	if !strings.Contains(out, "enter: continuar") {
		t.Errorf("footer ausente en el render por defecto")
	}
}

// TestFlujoTeclado cubre las transiciones de fase del asistente
// (Patrón: state transitions directas sobre Model.Update).
func TestFlujoTeclado(t *testing.T) {
	m := tea.Model(initial())

	// 1) nombre inválido no avanza
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Mi Proyecto")})
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.(*tui).fase != faseNombre {
		t.Fatalf("nombre inválido: fase = %v, esperado faseNombre", m.(*tui).fase)
	}

	// 2) nombre válido → faseBuild
	m.(*tui).input.SetValue("mi-api")
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.(*tui).fase != faseBuild {
		t.Fatalf("tras nombre válido: fase = %v, esperado faseBuild", m.(*tui).fase)
	}

	// 3) bajar + enter en build tool → Gradle → faseJava
	m = press(m, tea.KeyMsg{Type: tea.KeyDown})
	if got := m.(*tui).sel.actual(); got != "gradle" {
		t.Fatalf("build tool = %q, esperado gradle", got)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.(*tui).fase != faseJava {
		t.Fatalf("tras build tool: fase = %v, esperado faseJava", m.(*tui).fase)
	}

	// 4) bajar + enter → Java 21 → fasePackaging
	m = press(m, tea.KeyMsg{Type: tea.KeyDown})
	if got := m.(*tui).sel.actual(); got != "21" {
		t.Fatalf("java = %q, esperado 21", got)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.(*tui).fase != fasePackaging {
		t.Fatalf("tras java: fase = %v, esperado fasePackaging", m.(*tui).fase)
	}

	// 5) bajar + enter → War → faseBootVer (paso de versión)
	m = press(m, tea.KeyMsg{Type: tea.KeyDown})
	if got := m.(*tui).sel.actual(); got != "war" {
		t.Fatalf("packaging = %q, esperado war", got)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.(*tui).fase != faseBootVer {
		t.Fatalf("tras packaging: fase = %v, esperado faseBootVer", m.(*tui).fase)
	}

	// 5b) enter → primera versión (GA default) → faseMetadatos
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.(*tui).fase != faseMetadatos {
		t.Fatalf("tras boot: fase = %v, esperado faseMetadatos", m.(*tui).fase)
	}
	if boot := m.(*tui).boot; boot != "4.1.1" {
		t.Fatalf("boot = %q, esperado 4.1.1 (GA default)", boot)
	}

	// 5c) metadatos: tipear group propio, enter×2 → faseDeps (package es
	// solo lectura y se deriva en vivo)
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("com.app")})
	if got := m.(*tui).groupField.Value(); got != "com.app" {
		t.Fatalf("group = %q, esperado com.app", got)
	}
	if got := m.(*tui).pkgPreview(); got != "com.app.miapi" {
		t.Fatalf("pkgPreview = %q, esperado com.app.miapi (artifact sanitizado)", got)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter}) // → Artifact
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter}) // aplicar → faseDeps
	if m.(*tui).fase != faseDeps {
		t.Fatalf("tras metadatos: fase = %v, esperado faseDeps", m.(*tui).fase)
	}
	if got := m.(*tui).groupName; got != "com.app" {
		t.Fatalf("groupName = %q, esperado com.app", got)
	}

	// 5d) el preview sanea guiones del artifact
	if got := (&tui{name: "mi-api"}).pkgPreview(); got != "com.example.miapi" {
		t.Fatalf("preview con artifact con guiones = %q, esperado com.example.miapi", got)
	}

	// 6) espacio marca la primera dependencia (Security); enter → faseResumen
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	marcadas := m.(*tui).deps.seleccionadas()
	if len(marcadas) != 1 || marcadas[0] != "security" {
		t.Fatalf("deps marcadas = %v, esperado [security]", marcadas)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.(*tui).fase != faseResumen {
		t.Fatalf("tras deps: fase = %v, esperado faseResumen", m.(*tui).fase)
	}

	// 7) resumen: n vuelve a dependencias
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.(*tui).fase != faseDeps {
		t.Fatalf("tras 'n': fase = %v, esperado faseDeps", m.(*tui).fase)
	}
}

// TestResumenTabla: la pantalla final es una tabla key-value de 2 columnas
// centrada en bloque, con footer de acciones fuera del contenedor.
func TestResumenTabla(t *testing.T) {
	m := initial()
	m.fase = faseResumen
	m.name = "mi-api"
	m.build = "maven"
	m.java = "21"
	m.pack = "jar"
	m.deps.con()
	m.deps.toggle() // Security marcada

	m.w, m.h = 120, 30
	out := m.View()

	for _, esperado := range []string{
		"Resumen",
		"Proyecto:", "mi-api",
		"Build tool:", "maven",
		"Java (SDK):", "21",
		"Empaquetado:", "jar",
		"Spring Boot:", "4.1.1 (default)",
		"Group:", "com.example",
		"Artifact:", "mi-api",
		"Package:", "com.example.miapi",
		"Dependencias:", "Web + Test (base) + Spring Security",
		"Destino:",
		"enter / y: generar · n: volver · ctrl+c: salir · t: matrix",
	} {
		if !strings.Contains(out, esperado) {
			t.Errorf("falta en el resumen: %q", esperado)
		}
	}
}

// TestWrapPalabras: el wrap no parte palabras y respeta el ancho máximo.
func TestWrapPalabras(t *testing.T) {
	texto := "Web + Test (base) + Spring Security, Actuator, PostgreSQL, Redis"
	lineas := wrapPalabras(texto, 20)
	if len(lineas) < 2 {
		t.Fatalf("debería envolver en varias líneas, salió: %v", lineas)
	}
	for _, l := range lineas {
		if runewidth.StringWidth(l) > 20 {
			t.Errorf("línea excede 20 columnas: %q", l)
		}
		// una línea envuelta no puede empezar con un espacio (el indent lo da la tabla)
		if strings.HasPrefix(l, " ") {
			t.Errorf("línea envuelta con espacio inicial: %q", l)
		}
	}
}

// TestTruncarMedio: textos largos se cortan con elipsis sin romper el ancho.
func TestTruncarMedio(t *testing.T) {
	if got := truncarMedio("corto", 60); got != "corto" {
		t.Errorf("texto corto no debería truncarse: %q", got)
	}
	largo := strings.Repeat("a", 120)
	got := truncarMedio(largo, 20)
	if !strings.Contains(got, "…") {
		t.Errorf("falta la elipsis: %q", got)
	}
	if runewidth.StringWidth(got) > 20 {
		t.Errorf("truncado excede 20 columnas: %q", got)
	}
	if !strings.HasPrefix(got, "aaa") || !strings.HasSuffix(got, "aaa") {
		t.Errorf("debe conservar inicio y fin: %q", got)
	}
}

// TestLineasVaciasConFondo: ninguna línea vacía del contenido es un "\n"
// plano — todas llevan el fondo del panel (fix de la franja negra en los
// respiros de Metadatos y demás).
func TestLineasVaciasConFondo(t *testing.T) {
	out := rellenarVacias("a\n\nb", 10)
	lineas := strings.Split(out, "\n")
	if len(lineas) != 3 {
		t.Fatalf("líneas = %d, esperado 3", len(lineas))
	}
	medio := lineas[1]
	if got := runewidth.StringWidth(ansi.Strip(medio)); got != 10 {
		t.Errorf("línea vacía con ancho %d, esperado 10 (rellena de espacios)", got)
	}
	if !strings.Contains(medio, "48;5;234") {
		t.Errorf("la línea vacía no tiene el fondo del panel: %q", medio)
	}
	// y en la vista real: la línea entre Artifact y el preview no queda negra
	m := initial()
	m.fase = faseMetadatos
	m.name = "mi-api"
	m.w, m.h = 120, 30
	out2 := m.View()
	negras := 0
	for _, l := range strings.Split(out2, "\n") {
		if runewidth.StringWidth(ansi.Strip(l)) == 0 {
			negras++
		}
	}
	if negras > 0 {
		t.Errorf("%d líneas vacías planas en la vista de metadatos (franjas negras)", negras)
	}
}

// TestToggleSinDesplazamiento: alternar `t` NO mueve la caja — el toggle solo
// cambia los verdes del fondo por espacios; las coordenadas del modal son
// fijas (fix del salto vertical).
func TestToggleSinDesplazamiento(t *testing.T) {
	m := tea.Model(initial())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})

	indiceDe := func(out string) int {
		for i, l := range strings.Split(out, "\n") {
			if strings.Contains(l, "¿Cómo se llama el proyecto?") {
				return i
			}
		}
		return -1
	}

	on := m.View()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")}) // apagar
	off := m.View()

	iOn, iOff := indiceDe(on), indiceDe(off)
	if iOn != iOff {
		t.Fatalf("la caja salta verticalmente: índice %d (matrix) vs %d (off)", iOn, iOff)
	}
	if strings.Count(off, "\n") != strings.Count(on, "\n") {
		t.Errorf("alto distinto entre modos: %d vs %d", strings.Count(off, "\n"), strings.Count(on, "\n"))
	}
}

// TestToggleDeps: marcar/desmarcar del multiselect contra el catálogo.
func TestToggleDeps(t *testing.T) {
	m := &tui{}
	m.deps.con()

	if len(m.deps.items) != len(model.Dependencies) {
		t.Fatalf("catálogo = %d items, esperado %d", len(m.deps.items), len(model.Dependencies))
	}

	// toggle de la primera (Security)
	m.deps.toggle()
	if !m.deps.items[0].marcada {
		t.Fatalf("la primera dep debería quedar marcada")
	}
	if got := m.deps.seleccionadas(); len(got) != 1 || got[0] != "security" {
		t.Fatalf("seleccionadas = %v, esperado [security]", got)
	}

	// desmarcar → lista vacía
	m.deps.toggle()
	if got := m.deps.seleccionadas(); len(got) != 0 {
		t.Fatalf("tras desmarcar: seleccionadas = %v, esperado vacío", got)
	}
}

// TestNavegacionLimites: el cursor no se sale de rango (Patrón 1).
func TestNavegacionLimites(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		desde int
		hasta int
		n     int
	}{
		{"down desde 0", "j", 0, 1, 2},
		{"down en el fondo", "j", 1, 1, 2},
		{"up en el tope", "k", 0, 0, 2},
		{"up desde 1", "k", 1, 0, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &selector{items: make([]opcion, tt.n), idx: tt.desde}
			nav(s, tt.key)
			if s.idx != tt.hasta {
				t.Errorf("idx = %d, esperado %d", s.idx, tt.hasta)
			}
		})
	}
}