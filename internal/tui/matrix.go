package tui

import (
	"math/rand"
	"strings"
)

// matrix es el fondo full-screen estilo cmatrix: la lluvia corre en TODO el
// ancho y alto del viewport (sin franjas vacías); el modal y el footer (con
// fondos sólidos) se imprimen por encima; alrededor del modal hay una
// vignette de transición que atenúa las gotas (dim + drop) para un corte
// suave en vez de un borde tosco.
//
// Modelo: cada celda guarda un glow = "distancia desde la cabeza" (cuanto
// mayor, más cerca de la cabeza). Cada tick la cabeza escribe una celda con
// glow = maxGlow de la columna y TODO decae 1 — la estela tiene naturalmente
// maxGlow caracteres de largo, con gradiente incluido.
type matrix struct {
	w, h   int
	cols   []mCol
	celdas [][]mCelda
	ticks  int
	active bool
	// rectángulo del modal (caja) para la vignette: las gotas en la franja
	// de transición (margen alrededor) se atenúan; el modal las tapa.
	vx, vy, vw, vh int
}

// Vignette: margen de transición fuera de los bordes de la caja.
const (
	vignetteX = 3 // columnas a cada lado
	vignetteY = 2 // filas arriba y abajo
)

type mCol struct {
	x       int
	head    int
	delay   int // ticks de espera antes de caer de nuevo (velocidades distintas)
	step    int // 1 = lenta, 2 = rápida
	maxGlow int // largo de la estela: 12..25
}

type mCelda struct {
	ch   rune
	glow int
}

// lluviaChars: halfwidth katakana + alfanuméricos. Los katakana full-width
// (ア) miden 2 columnas y descuadran el grid — los halfwidth (ｱ) miden 1.
var lluviaChars = []rune("ｱｲｳｴｵｶｷｸｹｺｻｼｽｾｿﾀﾁﾂﾃﾄﾅﾆﾇﾈﾉ0123456789ABCDEF$#%&")

// Densidad: entre 60% y 80% de las columnas del viewport con gotas activas.
const densidadLluvia = 0.7

func newMatrix(w, h int) *matrix {
	m := &matrix{w: w, h: h, active: true}
	m.celdas = make([][]mCelda, h)
	for i := range m.celdas {
		m.celdas[i] = make([]mCelda, w)
	}
	nCol := max(1, int(float64(w)*densidadLluvia))
	for i := 0; i < nCol; i++ {
		m.cols = append(m.cols, nuevaCol(w))
	}
	return m
}

// nuevaCol crea una gota con largo de estela, velocidad y delay aleatorios.
func nuevaCol(w int) mCol {
	return mCol{
		x:       rand.Intn(w),
		head:    0,
		delay:   rand.Intn(4),     // arranque rápido para densidad inmediata
		step:    1 + rand.Intn(2), // 1 lenta / 2 rápida
		maxGlow: 12 + rand.Intn(14), // estela de 12 a 25 caracteres
	}
}

func (m *matrix) toggle()      { m.active = !m.active }
func (m *matrix) activa() bool { return m.active }

// zona define el rectángulo del modal (NO excluye: la lluvia corre en todo
// el viewport; el modal la tapa con su fondo sólido). Sirve para calcular
// la franja de transición de la vignette.
func (m *matrix) zona(x, y, w, h int) {
	m.vx, m.vy, m.vw, m.vh = x, y, w, h
}

// enTransicion: celda en la franja de TRANSICIÓN propiamente dicha — el
// margen fuera del modal (NO el interior, que el modal tapa). Solo ese
// anillo se atenúa; el resto del canvas conserva la lluvia a plena potencia.
func (m *matrix) enTransicion(j, i int) bool {
	if m.vw == 0 {
		return false
	}
	dentroExt := j >= m.vx-vignetteX && j < m.vx+m.vw+vignetteX &&
		i >= m.vy-vignetteY && i < m.vy+m.vh+vignetteY
	dentroModal := j >= m.vx && j < m.vx+m.vw && i >= m.vy && i < m.vy+m.vh
	return dentroExt && !dentroModal
}

// tick avanza un frame: cada columna con delay 0 suelta una gota nueva con
// maxGlow en su cabeza; el cuerpo de la estela ya formado decae (glow--).
func (m *matrix) tick() {
	m.ticks++
	for i := range m.cols {
		c := &m.cols[i]
		if c.delay > 0 {
			c.delay--
			continue
		}
		// cabeza nueva con glow máximo de esta columna (estela de maxGlow)
		if c.head < m.h {
			m.celdas[c.head][c.x] = mCelda{ch: lluviaChars[rand.Intn(len(lluviaChars))], glow: c.maxGlow}
			c.head += c.step
			if c.head >= m.h {
				c.head = 0
				c.delay = 2 + rand.Intn(6)
				c.step = 1 + rand.Intn(2)
				c.maxGlow = 12 + rand.Intn(14)
			}
		}
	}
	// decaimiento global del cuerpo (estela que se desvanece en el tiempo)
	for _, fila := range m.celdas {
		for j := range fila {
			if fila[j].glow > 0 {
				fila[j].glow--
			}
			if fila[j].glow == 0 {
				fila[j].ch = 0
			}
		}
	}
}

func indexCols(m *matrix) map[int]int {
	idx := make(map[int]int, len(m.cols))
	for i, c := range m.cols {
		idx[c.x] = i // si dos columnas comparten x, la última gana (raro)
	}
	return idx
}

// dibujar pinta la lluvia en el framebuffer (capa 1). Cada gota escribe su
// celda en el canvas con su color; la vignette atenúa (dim + drop 50%).
func (m *matrix) dibujar(c *canvas) {
	posCol := indexCols(m)
	for i := 0; i < m.h; i++ {
		for j := 0; j < m.w; j++ {
			cl := m.celdas[i][j]
			if cl.glow == 0 {
				continue // celda vacía: el canvas ya tiene ' '
			}
			// vignette (SOLO el margen fuera del modal): ultra tenues + drop 50%
			if m.enTransicion(j, i) {
				if rand.Intn(100) < 50 {
					continue
				}
				c.set(i, j, cl.ch, "38;2;0;120;35", "", false, true)
				continue
			}
			// rel = distancia desde la cabeza de esta columna
			rel := cl.glow
			if ci, ok := posCol[j]; ok {
				rel = m.cols[ci].maxGlow - cl.glow
				if rel < 0 {
					rel = 0
				}
			} else {
				rel = 20
			}
			switch {
			case rel == 0: // cabeza: verde lima vivo (sin blanco ni fosforescente)
				c.set(i, j, cl.ch, "38;2;85;255;120", "", false, false)
			case rel <= 4: // estela cercana: verde digital vivo
				c.set(i, j, cl.ch, "38;2;0;222;80", "", false, false)
			case rel <= 10: // cuerpo: verde medio
				c.set(i, j, cl.ch, "38;2;0;170;55", "", false, false)
			default: // cola lejana: verde visible (SIN dim — el dim lo apagaba)
				c.set(i, j, cl.ch, "38;2;0;120;35", "", false, false)
			}
		}
	}
}

// frame es un wrapper de diagnóstico: dibuja en un canvas nuevo y lo
// renderiza a []string (usado por los tests del matrix).
func (m *matrix) frame() []string {
	c := newCanvas(m.w, m.h)
	m.dibujar(c)
	return strings.Split(c.render(), "\n")
}