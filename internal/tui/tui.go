package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/GzaJai/spring-box/internal/model"
)

// fase es el paso actual del asistente. Es el "calco" de start.spring.io:
// Project (build tool) → Java version → Packaging → Spring Boot versión →
// Metadatos (group/artifact/package) → Dependencias → resumen.
type fase int

const (
	faseNombre fase = iota
	faseBuild
	faseJava
	fasePackaging
	faseBootVer
	faseMetadatos
	faseDeps
	faseResumen
)

// slugRe valida el nombre del proyecto: minúsculas, números y guiones.
var slugRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// opcion es una entrada de los selectores (↑↓ + enter).
type opcion struct {
	label string // lo que se ve
	valor string // lo que se guarda en Options
}

// selector es un picker genérico reutilizado en todos los pasos de opción.
type selector struct {
	titulo string
	items  []opcion
	idx    int
	ayuda  string
}

func (s *selector) actual() string        { return s.items[s.idx].valor }
func (s *selector) subir()                { if s.idx > 0 { s.idx-- } }
func (s *selector) bajar()                { if s.idx < len(s.items)-1 { s.idx++ } }
func (s *selector) con(t string, a string, items ...opcion) {
	s.titulo, s.ayuda, s.items, s.idx = t, a, items, 0
}

// depItem es una fila del multiselect de dependencias.
type depItem struct {
	dep     model.Dependency
	marcada bool
}

// multiselect permite marcar varias dependencias (↑↓ + espacio + enter).
type multiselect struct {
	titulo string
	items  []depItem
	idx    int
	ayuda  string
}

func (m *multiselect) subir() { if m.idx > 0 { m.idx-- } }
func (m *multiselect) bajar() { if m.idx < len(m.items)-1 { m.idx++ } }
func (m *multiselect) toggle() { m.items[m.idx].marcada = !m.items[m.idx].marcada }
func (m *multiselect) seleccionadas() []string {
	ids := []string{}
	for _, it := range m.items {
		if it.marcada {
			ids = append(ids, it.dep.ID)
		}
	}
	return ids
}
func (m *multiselect) textos() []string {
	labels := []string{}
	for _, it := range m.items {
		if it.marcada {
			labels = append(labels, it.dep.Label)
		}
	}
	return labels
}
func (m *multiselect) con() {
	m.titulo = "¿Qué dependencias querés sumar?"
	m.ayuda = "↑↓: navegar · espacio: marcar · enter: continuar · ctrl+c: salir"
	m.items = make([]depItem, len(model.Dependencies))
	for i, d := range model.Dependencies {
		m.items[i] = depItem{dep: d}
	}
	m.idx = 0
}

type tui struct {
	fase   fase
	input  textinput.Model
	sel    selector
	deps   multiselect
	name   string
	build  string
	java   string
	pack   string
	boot   string
	// metadatos del proyecto (group / artifact editables; package se deriva)
	groupField    textinput.Model
	artifactField textinput.Model
	metaIdx       int
	groupName     string
	artifactName  string
	quit          bool
	w, h          int // dimensiones del viewport (recibidas por WindowSizeMsg)
	// fondo es el hiperespacio de fondo (nil hasta el primer resize).
	fondo *starfield
}

// fondoTickMsg avanza un frame del hiperespacio. Intervalo 60ms ≈ 16fps.
type fondoTickMsg struct{}

const fondoIntervalo = 60 * time.Millisecond

func fondoCmd() tea.Cmd {
	return tea.Tick(fondoIntervalo, func(time.Time) tea.Msg { return fondoTickMsg{} })
}

// Run ejecuta el asistente y devuelve las opciones elegidas.
//
// tea.WithAltScreen() activa el alternate screen buffer (\x1b[?1049h) al
// iniciar: la consola previa queda OCULTA y la app ocupa toda la terminal.
// Bubbletea restaura el buffer principal (\x1b[?1049l) automáticamente al
// salir, incluso en Ctrl+C/SIGINT — la terminal del usuario no se rompe.
func Run() (model.Options, error) {
	p := tea.NewProgram(initial(), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		return model.Options{}, err
	}
	t := m.(*tui)
	if t.quit {
		os.Exit(0) // el usuario canceló: salida limpia, no se genera nada
	}
	cwd, _ := os.Getwd()
	return model.Options{
		Name:         t.name,
		BuildTool:    t.build,
		JavaVer:      t.java,
		Packaging:    t.pack,
		BootVer:      t.boot,
		Group:        t.groupName,
		Artifact:     t.artifactName,
		Dependencies: t.deps.seleccionadas(),
		Stack:        model.Stacks[0],
		Dir:          filepath.Join(cwd, t.name),
	}, nil
}

func initial() *tui {
	in := textinput.New()
	in.Placeholder = "mi-api"
	in.Prompt = "> "
	in.Focus()
	in.CharLimit = 60
	estiloInputPanel(in)

	t := &tui{fase: faseNombre, input: in, w: 80, h: 24} // defaults hasta el primer resize

	// inputs de metadatos: placeholders dinámicos se setean al entrar
	meta := func() textinput.Model {
		m := textinput.New()
		m.Prompt = "> "
		m.CharLimit = 200
		estiloInputPanel(m)
		return m
	}
	t.groupField = meta()
	t.artifactField = meta()

	t.sel.con(
		"¿Qué build tool?",
		"↑↓: elegir · enter: confirmar · ctrl+c: salir",
		opcion{"Maven", "maven"},
		opcion{"Gradle", "gradle"},
	)
	return t
}

// estiloInputPanel: todos los sub-estilos del input (prompt, cursor, texto,
// placeholder) llevan el fondo del panel — sin esto, el reset interno del
// input deja parches sobre el gris de la caja.
func estiloInputPanel(in textinput.Model) {
	bg := lipgloss.Color(panelBG)
	in.PromptStyle = in.PromptStyle.Background(bg)
	in.CursorStyle = in.CursorStyle.Background(bg)
	in.TextStyle = in.TextStyle.Background(bg)
	in.PlaceholderStyle = in.PlaceholderStyle.Background(bg)
}

func (t *tui) Init() tea.Cmd { return tea.Batch(textinput.Blink, fondoCmd()) }

// enInputTexto: hay un campo de texto activo → las teclas de control del
// fondo (t) y de salida (q) NO deben interceptarse: son caracteres normales.
func (t *tui) enInputTexto() bool {
	return t.fase == faseNombre || t.fase == faseMetadatos
}

func (t *tui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// resize (SIGWINCH): recalcular dimensiones; el centrado se reaplica
		// en cada View() usando w/h actuales
		t.w, t.h = msg.Width, msg.Height
		t.fondo = newStarfield(t.w, t.h)
		return t, nil

	case fondoTickMsg:
		if t.fondo != nil && t.fondo.activa() {
			t.fondo.tick()
		}
		return t, fondoCmd()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			t.quit = true
			return t, tea.Quit
		case "t":
			// FIX: "t" solo togglea el fondo en pantallas SIN input de texto.
			// Si hay input activo (nombre/metadatos), NO retornamos: la tecla
			// cae al manejador del campo y se escribe como carácter normal.
			if !t.enInputTexto() {
				if t.fondo != nil {
					t.fondo.toggle()
				}
				return t, nil
			}
		case "q":
			// mismo criterio: "q" sale SOLO desde el Resumen (no hay input ahí);
			// en faseNombre/Metadatos es un carácter normal del campo.
			if t.fase == faseResumen {
				t.quit = true
				return t, tea.Quit
			}
		}

		switch t.fase {
		case faseNombre:
			if msg.String() == "enter" {
				v := strings.TrimSpace(t.input.Value())
				if slugRe.MatchString(v) {
					t.name = v
					t.fase = faseBuild
					t.sel.con(
						"¿Qué build tool?",
						"↑↓: elegir · enter: confirmar · ctrl+c: salir",
						opcion{"Maven", "maven"},
						opcion{"Gradle", "gradle"},
					)
				}
				return t, nil
			}

		case faseBuild:
			if esNav(msg.String()) {
				nav(&t.sel, msg.String())
				return t, nil
			}
			if msg.String() == "enter" {
				t.build = t.sel.actual()
				t.fase = faseJava
				t.sel.con(
					"¿Versión de Java (SDK)?",
					"↑↓: elegir · enter: confirmar · ctrl+c: salir",
					opcion{"17 (LTS)", "17"},
					opcion{"21 (LTS)", "21"},
				)
				return t, nil
			}

		case faseJava:
			if esNav(msg.String()) {
				nav(&t.sel, msg.String())
				return t, nil
			}
			if msg.String() == "enter" {
				t.java = t.sel.actual()
				t.fase = fasePackaging
				t.sel.con(
					"¿Empaquetador?",
					"↑↓: elegir · enter: confirmar · ctrl+c: salir",
					opcion{"Jar  (ejecutable embebido)", "jar"},
					opcion{"War  (para servidor externo)", "war"},
				)
				return t, nil
			}

		case fasePackaging:
			if esNav(msg.String()) {
				nav(&t.sel, msg.String())
				return t, nil
			}
			if msg.String() == "enter" {
				t.pack = t.sel.actual()
				t.sel.con("¿Versión de Spring Boot?",
					"↑↓: elegir · enter: confirmar · ctrl+c: salir",
					bootOpciones()...)
				t.fase = faseBootVer
				return t, nil
			}

		case faseBootVer:
			if esNav(msg.String()) {
				nav(&t.sel, msg.String())
				return t, nil
			}
			if msg.String() == "enter" {
				t.boot = t.sel.actual()
				t.iniciarMetadatos()
				t.fase = faseMetadatos
				return t, nil
			}

		case faseMetadatos:
			if msg.String() == "enter" {
				if t.metaIdx < 1 {
					t.metaIdx++
					t.focusMeta()
				} else {
					t.aplicarMetadatos()
					t.fase = faseDeps
				}
				return t, nil
			}
			var cmd tea.Cmd
			switch t.metaIdx {
			case 0:
				t.groupField, cmd = t.groupField.Update(msg)
			case 1:
				t.artifactField, cmd = t.artifactField.Update(msg)
			}
			return t, cmd

		case faseDeps:
			if esNav(msg.String()) {
				navDeps(&t.deps, msg.String())
				return t, nil
			}
			switch msg.String() {
			case " ":
				t.deps.toggle()
				return t, nil
			case "enter":
				t.fase = faseResumen
				return t, nil
			}

		case faseResumen:
			switch msg.String() {
			case "enter", "y", "s":
				return t, tea.Quit
			case "n":
				t.fase = faseDeps // volver a elegir dependencias
				return t, nil
			}
		}
	}

	var cmd tea.Cmd
	if t.fase == faseNombre {
		t.input, cmd = t.input.Update(msg)
	}
	return t, cmd
}

func esNav(k string) bool { return k == "up" || k == "down" || k == "k" || k == "j" }

func nav(s *selector, k string) {
	switch k {
	case "up", "k":
		s.subir()
	case "down", "j":
		s.bajar()
	}
}

func navDeps(m *multiselect, k string) {
	switch k {
	case "up", "k":
		m.subir()
	case "down", "j":
		m.bajar()
	}
}

// bootOpciones convierte el catálogo de versiones en opciones del selector.
func bootOpciones() []opcion {
	items := make([]opcion, len(model.BootVersions))
	for i, bv := range model.BootVersions {
		items[i] = opcion{label: bv.Label, valor: bv.Version}
	}
	return items
}

// iniciarMetadatos prepara los 2 inputs del paso Metadatos: value vacío
// (placeholder = default), foco en Group.
func (t *tui) iniciarMetadatos() {
	t.groupField.SetValue("")
	t.artifactField.SetValue("")
	t.metaIdx = 0
	t.refreshMetaPlaceholders()
	t.focusMeta()
}

// refreshMetaPlaceholders: defaults dinámicos tipo Initializr — group hereda
// com.example, artifact hereda el nombre del proyecto.
func (t *tui) refreshMetaPlaceholders() {
	t.groupField.Placeholder = "com.example"
	t.artifactField.Placeholder = t.name
}

func (t *tui) focusMeta() {
	t.groupField.Blur()
	t.artifactField.Blur()
	if t.metaIdx == 0 {
		t.groupField.Focus()
	} else {
		t.artifactField.Focus()
	}
}

// pkgPreview calcula el package EN VIVO (solo lectura): <group>.<artifact>
// sanitizado — sin guiones (convenciones de paquetes Java). Usa los valores
// tipeados, cayendo a los defaults si el campo está vacío.
func (t *tui) pkgPreview() string {
	group := t.groupField.Value()
	if group == "" {
		group = "com.example"
	}
	artifact := t.artifactField.Value()
	if artifact == "" {
		artifact = t.name
	}
	return group + "." + strings.ReplaceAll(artifact, "-", "")
}

// aplicarMetadatos deja los valores como los escribió el usuario; los vacíos
// se resuelven con defaults en model.Resolve* (mismo criterio del generador).
// Package se deriva SIEMPRE (solo lectura) — nunca viaja del wizard.
func (t *tui) aplicarMetadatos() {
	t.groupName = t.groupField.Value()
	t.artifactName = t.artifactField.Value()
	t.deps.con() // arma el multiselect desde el catálogo para el próximo paso
}

// ---- jerarquía visual ----
//
// Caja de ancho automático: el wordmark define el ancho del contenido (con un
// mínimo por la tabla del Resumen); la caja se dimensiona sola (padding + borde).
// El footer de teclas va FUERA del contenedor, debajo del borde inferior,
// centrado respecto a la caja. lipgloss.Place centra todo el bloque en el
// viewport.

// ---- estilos ----
//
// FIX "bloques negros": los textos internos NO cierran con \x1b[0m (que
// resetea el background del panel). Todo cierra con \x1b[39;22;23m (default
// fg + bold/faint off) y el fondo gris lo pone el CONTENEDOR, que rellena
// cada línea completa con su background — superficie uniforme hasta el borde.
const panelBG = "234" // gris oscuro del panel

var estiloBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("245")).
	Padding(1).Background(lipgloss.Color(panelBG))

// fgg pinta texto con foreground de 256 colores y cierra SOLO el foreground
// (39) + bold/faint (22/23): el background del panel sobrevive en toda la
// línea. Esta es la raíz del fix de los parches negros.
func fgg(c int, bold, faint bool, s string) string {
	attrs := ""
	if bold {
		attrs += "1;"
	}
	if faint {
		attrs += "2;"
	}
	return "\x1b[" + attrs + "38;5;" + strconv.Itoa(c) + "m" + s + "\x1b[39;22;23m"
}

func (t *tui) View() string {
	// Framebuffer 2D de tamaño EXACTO termWidth x termHeight: el render
	// final es fijo en ambos modos (fondo on/off) — cero saltos al toggle.
	c := newCanvas(t.w, t.h)

	caja := t.caja()
	ac := anchoBloque(caja)
	altoCaja := len(strings.Split(caja, "\n"))
	// coordenadas FIJAS del modal (idénticas con y sin fondo)
	boxX := max(0, (t.w-ac)/2)
	boxY := max(0, (t.h-altoCaja)/2)

	// Capa 1: fondo — hiperespacio o espacios limpios
	if t.fondo != nil && t.fondo.activa() {
		t.fondo.zona(boxX, boxY, min(ac, t.w), min(altoCaja, t.h))
		t.fondo.dibujar(c)
	}

	// Capa 2: estampado del modal y del footer — SOBRESCRITURA total por
	// celda, con fondo de panel garantizado. Es matemáticamente imposible
	// que el fondo cruce el borde o que un respiro quede negro.
	estampar(c, parsearANSI(caja, panelBG), boxX, boxY)

	// el pie se posiciona con coordenadas (el respiro lo da el canvas, no el
	// string): 1 línea de aire debajo del modal; si no cabe, al último renglón.
	pie := strings.TrimPrefix(t.footer(), "\n\n")
	footerAncho := t.anchoModal() + 4
	yFooter := boxY + altoCaja + 1
	if yFooter >= t.h {
		yFooter = t.h - 1
	}
	estampar(c, parsearANSI(pie, panelBG), max(0, (t.w-footerAncho)/2), yFooter)

	// Capa 3: render fila por fila (alto y ancho exactos)
	return c.render()
}

// caja construye el contenedor con borde redondeado y contenido interno
// ordenado como una única columna: banner → subtítulo → pregunta → control.
func (t *tui) caja() string {
	interno := t.header() + "\n\n" + t.cuerpo()
	// FIX franja negra: ninguna línea vacía del contenido puede ser un "\n"
	// plano (deja ver el negro de la terminal entre los respiros). Todas se
	// rellenan con espacios y el fondo del panel.
	ancho := t.anchoModal()
	interno = rellenarVacias(interno, ancho)
	contenido := lipgloss.NewStyle().Width(ancho).Align(lipgloss.Center).Background(lipgloss.Color(panelBG))
	return estiloBox.Render(contenido.Render(interno))
}

// rellenarVacias convierte cada línea sin caracteres visibles en una cadena
// de espacios con el fondo del panel — nunca un salto plano.
func rellenarVacias(s string, ancho int) string {
	lineas := strings.Split(s, "\n")
	for i, l := range lineas {
		if lipgloss.Width(l) == 0 {
			lineas[i] = "\x1b[48;5;" + panelBG + "m" + strings.Repeat(" ", ancho) + "\x1b[39m"
		}
	}
	return strings.Join(lineas, "\n")
}

// header: banner ASCII + subtítulo tenue (1 línea de separación).
func (t *tui) header() string {
	return t.banner() + "\n" + fgg(246, false, true, "arrancá tu proyecto Spring en local")
}


// degradado interpola linealmente entre verde (140,255,120) y BLANCO
// (255,255,255) según el índice de la línea — los colores de Spring.
// Truecolor: Windows Terminal y las terminales modernas lo soportan.
func degradado(i, n int) (int, int, int) {
	if n <= 1 {
		return 140, 255, 120
	}
	t := float64(i) / float64(n-1)
	lerp := func(a, b float64) int { return int(a + (b-a)*t) }
	return lerp(140, 255), lerp(255, 255), lerp(120, 255)
}

func (t *tui) cuerpo() string {
	switch t.fase {
	case faseNombre:
		return t.cuerpoNombre()
	case faseMetadatos:
		return t.cuerpoMetadatos()
	case faseDeps:
		return t.cuerpoDeps()
	case faseResumen:
		return t.cuerpoResumen()
	default:
		return t.cuerpoSelector()
	}
}

// cuerpoNombre: pregunta vibrante + input con prompt "> " y cursor activo.
func (t *tui) cuerpoNombre() string {
	s := fgg(214, true, false, "¿Cómo se llama el proyecto?") + "\n\n"
	s += inputLinea(t.input, 66) + "\n"
	v := strings.TrimSpace(t.input.Value())
	if v != "" && !slugRe.MatchString(v) {
		s += "\n" + fgg(196, false, false, "Solo minúsculas, números y guiones (ej: mi-api, yui-backend).")
	} else {
		s += "\n"
	}
	return s
}

// cuerpoSelector: pregunta + opciones como bloque alineado y centrado,
// con respiro arriba (2 líneas) y abajo.
func (t *tui) cuerpoSelector() string {
	s := fgg(214, true, false, t.sel.titulo) + "\n\n"
	var lineas []string
	for i, it := range t.sel.items {
		if i == t.sel.idx {
			lineas = append(lineas, fgg(214, true, false, "→ ")+fgg(252, false, false, it.label))
		} else {
			lineas = append(lineas, "  "+fgg(252, false, false, it.label))
		}
	}
	return s + "\n" + bloqueCentrado(strings.Join(lineas, "\n")) + "\n"
}

// cuerpoMetadatos: Group y Artifact editables (vacío = default) + preview de
// Package en vivo (solo lectura). Todo el bloque del formulario forma UNA
// unidad centrada: las etiquetas (ancho fijo) definen la columna de los
// inputs; el preview respira 1 línea después de Artifact.
func (t *tui) cuerpoMetadatos() string {
	s := fgg(214, true, false, "Metadatos del proyecto (enter: siguiente — vacío = default)") + "\n\n"
	filas := []string{
		metaFila("Group:", t.groupField),
		metaFila("Artifact:", t.artifactField),
	}
	bloque := strings.Join(filas, "\n")
	bloque += "\n\n" + fgg(81, false, true, "Package generado: "+t.pkgPreview())
	return s + bloqueCentrado(bloque) + "\n"
}

// metaFila: etiqueta con ancho fijo (%-9s) → Group: y Artifact: arrancan los
// textos exactamente en la misma columna.
func metaFila(label string, in textinput.Model) string {
	eti := fgg(81, false, false, fmt.Sprintf("%-9s", label))
	return eti + " " + inputLinea(in, 62)
}

// inputLinea renderiza el input y RELLENA el resto de la línea con el fondo
// del panel de forma explícita — elimina los parches negros que dejan los
// resets internos del textinput después del texto.
func inputLinea(in textinput.Model, objetivo int) string {
	v := in.View()
	anchoActual := lipgloss.Width(v)
	pad := objetivo - anchoActual
	if pad < 0 {
		pad = 0
	}
	fondo := "\x1b[48;5;" + panelBG + "m"
	return fondo + v + fondo + strings.Repeat(" ", pad)
}

// cuerpoDeps: checkbox por dependencia del catálogo (↑↓ + espacio).
func (t *tui) cuerpoDeps() string {
	s := fgg(214, true, false, t.deps.titulo) + "\n\n"
	var lineas []string
	for i, it := range t.deps.items {
		check := "☐"
		if it.marcada {
			check = "☑"
		}
		linea := fmt.Sprintf("%s %-20s %s", check, it.dep.Label, fgg(246, false, true, it.dep.Desc))
		if i == t.deps.idx {
			lineas = append(lineas, fgg(214, true, false, "→ ")+fgg(252, false, false, linea))
		} else {
			lineas = append(lineas, "  "+fgg(252, false, false, linea))
		}
	}
	return s + "\n" + bloqueCentrado(strings.Join(lineas, "\n")) + "\n"
}

// cuerpoResumen: tabla key-value de 2 columnas centrada como bloque
// dentro de la caja. Etiquetas a la derecha contra los ":", valores a la
// izquierda. Textos largos: dependencias con wrap + indent en la columna
// de valores; destino truncado con elipsis en el medio.
func (t *tui) cuerpoResumen() string {
	// 1 línea de respiro arriba del título
	s := "\n" + fgg(214, true, false, "Resumen") + "\n\n"

	cwd, _ := os.Getwd()
	optsMeta := model.Options{Name: t.name, Group: t.groupName, Artifact: t.artifactName}
	group := model.ResolveGroup(optsMeta)
	artifact := model.ResolveArtifact(optsMeta)
	pkg := model.ResolvePackage(optsMeta)
	boot := t.boot
	if boot == "" {
		boot = model.SpringBootVersion
	}
	bootLabel := boot
	if boot == model.SpringBootVersion {
		bootLabel += " (default)"
	}

	depsBase := "Web + Test (base)"
	if extras := strings.Join(t.deps.textos(), ", "); extras != "" {
		depsBase += " + " + extras
	}
	if model.AutoH2(t.deps.seleccionadas()) {
		depsBase += " + H2 (dev, automático)"
	}
	destino := truncarMedio(filepath.Join(cwd, t.name), anchoValor)

	filas := []string{
		lineaKV("Proyecto:", t.name),
		lineaKV("Build tool:", t.build),
		lineaKV("Spring Boot:", bootLabel),
		lineaKV("Java (SDK):", t.java),
		lineaKV("Empaquetado:", t.pack),
		lineaKV("Group:", group),
		lineaKV("Artifact:", artifact),
		lineaKV("Package:", pkg),
	}
	filas = append(filas, lineaKVWrap("Dependencias:", depsBase)...)
	filas = append(filas, lineaKV("Destino:", destino))

	return s + bloqueCentrado(strings.Join(filas, "\n")) + "\n"
}

// Anchos de la tabla: 14 cols de etiqueta + 2 de separador. El interior del
// box (80 - padding 2 - borde 2) son 76 cols → el valor cabe en 60.
const (
	anchoEtiqueta = 14
	anchoValor    = 60
)

// lineaKV: una fila etiqueta→valor de la tabla.
func lineaKV(etiqueta, valor string) string {
	// etiqueta alineada a la derecha contra los dos puntos
	eti := fgg(81, false, false, fmt.Sprintf("%*s", anchoEtiqueta, etiqueta))
	return eti + "  " + fgg(255, false, false, valor)
}

// lineaKVWrap: como lineaKV pero si el valor no entra, hace word-wrap y las
// líneas siguientes indentan exactamente en la columna donde arranca el valor.
func lineaKVWrap(etiqueta, valor string) []string {
	lineas := wrapPalabras(valor, anchoValor)
	indentValor := strings.Repeat(" ", anchoEtiqueta+2)
	out := []string{lineaKV(etiqueta, lineas[0])}
	for _, l := range lineas[1:] {
		out = append(out, indentValor+fgg(255, false, false, l))
	}
	return out
}

// wrapPalabras: divide un texto en líneas de <= maxCol columnas, sin partir
// palabras (word-wrap).
func wrapPalabras(texto string, maxCol int) []string {
	var lineas []string
	var actual strings.Builder
	cols := 0
	primera := true
	for _, palabra := range strings.Fields(texto) {
		extra := 0
		if !primera {
			extra = 1 // el espacio entre palabras
		}
		if cols+extra+len(palabra) > maxCol && cols > 0 {
			lineas = append(lineas, actual.String())
			actual.Reset()
			cols, primera = 0, true
			extra = 0
		}
		if !primera {
			actual.WriteString(" ")
		}
		actual.WriteString(palabra)
		cols += extra + len(palabra)
		primera = false
	}
	if cols > 0 {
		lineas = append(lineas, actual.String())
	}
	if len(lineas) == 0 {
		lineas = []string{""}
	}
	return lineas
}

// truncarMedio: si el texto supera maxCol columnas, conserva el inicio y el
// fin de la ruta con elipsis en el medio (nunca rompe el layout).
func truncarMedio(s string, maxCol int) string {
	if len(s) <= maxCol {
		return s
	}
	izq := maxCol*3/5 - 1
	der := maxCol * 2 / 5
	r := []rune(s)
	if len(r) > izq+der {
		return string(r[:izq]) + "…" + string(r[len(r)-der:])
	}
	return s
}

// footer: fuera del contenedor, debajo del borde, centrado respecto a la caja.
func (t *tui) footer() string {
	texto := "enter: continuar · ctrl+c: salir"
	switch t.fase {
	case faseBuild, faseJava, fasePackaging, faseBootVer:
		texto = t.sel.ayuda
	case faseMetadatos:
		texto = "enter: siguiente (vacío = default) · ctrl+c: salir"
	case faseDeps:
		texto = t.deps.ayuda
	case faseResumen:
		texto = "enter / y: generar · n: volver · ctrl+c: salir"
	}
	// el toggle del fondo solo se anuncia donde la tecla "t" está disponible
	// (fuera de los campos de texto: ahí es un carácter que se escribe)
	if !t.enInputTexto() {
		texto += " · t: estrellas"
	}
	anchoCaja := t.anchoModal() + 4 // padding 1 c/u + borde 1 c/u
	pie := fgg(240, false, false, texto)
	// footer con FONDO SÓLIDO (máscara propia): el fondo nunca se mezcla
	// con las instrucciones — panel gris consistente con la caja.
	return "\n\n" + lipgloss.NewStyle().
		Width(anchoCaja).Align(lipgloss.Center).
		Background(lipgloss.Color(panelBG)).
		Render(pie)
}

// ---- utilidades de medición ----

// anchoBanner: ancho en columnas de la línea más larga del banner ASCII.
func anchoBanner() int {
	return anchoBloque(limpiarBanner(bannerASCII))
}

func limpiarBanner(s string) string {
	lineas := strings.Split(s, "\n")
	for i := range lineas {
		lineas[i] = strings.TrimRight(lineas[i], " ")
	}
	return strings.Join(lineas, "\n")
}

// anchoBloque: ancho máximo entre las líneas de un bloque multi-línea
// (ANSI-aware: los estilos internos no distorsionan la medición).
func anchoBloque(s string) int {
	m := 0
	for _, l := range strings.Split(s, "\n") {
		if a := lipgloss.Width(l); a > m {
			m = a
		}
	}
	return m
}

// bloqueCentrado: alinea las líneas del bloque a la izquierda entre sí y
// deja que el contenedor lo centre como una unidad (columnas parejas).
func bloqueCentrado(s string) string {
	return lipgloss.NewStyle().Width(anchoBloque(s)).Render(s)
}