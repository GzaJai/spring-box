package tui

import (
	"math/rand"
	"strings"
)

// starfield es el fondo de hiperespacio: estrellas que salen disparadas desde
// el centro hacia afuera. Cada estrella vive en un espacio 3D normalizado
// (x,y en [-1,1], z en (0,1]) y se proyecta con perspectiva (1/z): al
// acercarse (z→0) se acelera sola, sin necesidad de animar la velocidad.
//
// Se dibujan SOLO puntos (nada de estelas ni líneas).
type starfield struct {
	w, h   int
	stars  []star
	active bool
	// rectángulo del modal para la vignette de transición.
	vx, vy, vw, vh int
}

type star struct {
	x, y, z float64
}

// starSpeed: cuánto decrece z por frame (más alto = más rápido).
const starSpeed = 0.05

// starNear: al cruzar este plano la estrella reaparece lejos.
const starNear = 0.04

// newStarfield arma el campo de estrellas. La densidad escala con el área,
// con un piso alto para que el cielo se vea poblado.
func newStarfield(w, h int) *starfield {
	s := &starfield{w: w, h: h, active: true}
	n := max(120, w*h/30)
	s.stars = make([]star, n)
	for i := range s.stars {
		s.stars[i] = nuevaEstrella()
	}
	return s
}

func nuevaEstrella() star {
	return star{
		x: rand.Float64()*2 - 1,
		y: rand.Float64()*2 - 1,
		z: starNear + rand.Float64()*(1-starNear),
	}
}

func (s *starfield) toggle()      { s.active = !s.active }
func (s *starfield) activa() bool { return s.active }

// zona define el rectángulo del modal (para la vignette de transición).
func (s *starfield) zona(x, y, w, h int) {
	s.vx, s.vy, s.vw, s.vh = x, y, w, h
}

// enTransicion: celda en el margen fuera del modal (no en su interior).
func (s *starfield) enTransicion(j, i int) bool {
	if s.vw == 0 {
		return false
	}
	dentroExt := j >= s.vx-vignetteX && j < s.vx+s.vw+vignetteX &&
		i >= s.vy-vignetteY && i < s.vy+s.vh+vignetteY
	dentroModal := j >= s.vx && j < s.vx+s.vw && i >= s.vy && i < s.vy+s.vh
	return dentroExt && !dentroModal
}

// tick avanza un frame: cada estrella se acerca (z baja). Al cruzar el plano
// cercano, reaparece lejos en una dirección nueva.
func (s *starfield) tick() {
	for i := range s.stars {
		st := &s.stars[i]
		st.z -= starSpeed
		if st.z <= starNear {
			*st = nuevaEstrella()
		}
	}
}

// dibujar pinta cada estrella como un punto proyectado. Sin estelas: solo
// estrellas. La vignette descarta la mitad y atenúa cerca del modal.
func (s *starfield) dibujar(c *canvas) {
	cxf := float64(s.w) / 2
	cyf := float64(s.h) / 2
	for i := range s.stars {
		st := &s.stars[i]
		sx := int(cxf + (st.x/st.z)*(float64(s.w)/2))
		sy := int(cyf + (st.y/st.z)*(float64(s.h)/2))
		if sx < 0 || sx >= s.w || sy < 0 || sy >= s.h {
			continue
		}
		if s.enTransicion(sx, sy) {
			if rand.Intn(100) < 50 {
				continue
			}
			c.set(sy, sx, '.', colorVignette, "", false, true)
			continue
		}
		ch := '.'
		if st.z < 0.35 {
			ch = '*'
		}
		c.set(sy, sx, ch, colorEstrella(st.z), "", false, false)
	}
}

// colorEstrella: las más cercanas destellan en blanco, las lejanas en azul.
func colorEstrella(z float64) string {
	switch {
	case z < 0.12:
		return "38;2;255;255;255"
	case z < 0.3:
		return "38;2;190;225;255"
	case z < 0.55:
		return "38;2;120;175;245"
	default:
		return "38;2;70;115;190"
	}
}

// colorVignette: azul muy tenue para el margen alrededor del modal.
const colorVignette = "38;2;25;55;105"

// frame es un wrapper de diagnóstico (tests): dibuja en un canvas nuevo.
func (s *starfield) frame() []string {
	c := newCanvas(s.w, s.h)
	s.dibujar(c)
	return strings.Split(c.render(), "\n")
}
