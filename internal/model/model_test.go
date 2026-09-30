package model

import "testing"

func TestAutoH2(t *testing.T) {
	tests := []struct {
		name string
		ids  []string
		want bool
	}{
		{"sin deps", nil, false},
		{"jpa sola", []string{"data-jpa"}, true},
		{"jpa + mysql", []string{"data-jpa", "mysql"}, false},
		{"jpa + postgresql", []string{"data-jpa", "postgresql"}, false},
		{"mysql sola", []string{"mysql"}, false},
		{"security sola", []string{"security"}, false},
		{"jpa + redis", []string{"data-jpa", "data-redis"}, true},
		{"jpa + h2 explicito", []string{"data-jpa", "h2"}, false},
		{"h2 sola", []string{"h2"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AutoH2(tt.ids); got != tt.want {
				t.Errorf("AutoH2(%v) = %v, want %v", tt.ids, got, tt.want)
			}
		})
	}
}

func TestTieneDB(t *testing.T) {
	tests := []struct {
		name string
		ids  []string
		want string
	}{
		{"sin db", nil, ""},
		{"mysql", []string{"data-jpa", "mysql"}, "mysql"},
		{"postgresql", []string{"postgresql"}, "postgresql"},
		{"h2", []string{"h2"}, "h2"},
		{"sin driver solo jpa", []string{"data-jpa"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TieneDB(tt.ids); got != tt.want {
				t.Errorf("TieneDB(%v) = %q, want %q", tt.ids, got, tt.want)
			}
		})
	}
}

// TestCatalogoCoherente: el catálogo no puede tener ids duplicados, y un
// driver de BD SIEMPRE declara su motor (no puede haber dos con el mismo).
func TestCatalogoCoherente(t *testing.T) {
	vistos := map[string]bool{}
	for _, d := range Dependencies {
		if vistos[d.ID] {
			t.Errorf("id duplicado en el catálogo: %q", d.ID)
		}
		vistos[d.ID] = true
	}
	if len(TieneDB([]string{"mysql"})) == 0 || len(TieneDB([]string{"postgresql"})) == 0 {
		t.Errorf("mysql y postgresql deben declarar su motor (campo DB)")
	}
}