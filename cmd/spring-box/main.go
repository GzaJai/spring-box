package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/GzaJai/spring-box/internal/doctor"
	"github.com/GzaJai/spring-box/internal/fsutil"
	"github.com/GzaJai/spring-box/internal/generator"
	"github.com/GzaJai/spring-box/internal/tui"
)

// Inyectadas en build time con ldflags (-X main.version, -X main.commit,
// -X main.date). GoReleaser las completa en cada release. Si el binario viene
// de `go install ...@vX.Y.Z` sin ldflags, version queda "dev" y se cae al
// fallback de resolveVersion().
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "doctor":
			runDoctor()
			return
		case "version", "--version", "-v":
			printVersion()
			return
		}
	}
	runInit()
}

// printVersion imprime la versión del binario. Prioriza el valor inyectado por
// ldflags; si no, intenta leerlo de la metadata del módulo (poblada por
// `go install <modulo>@<version>`).
func printVersion() {
	fmt.Printf("spring-box %s\n", resolveVersion())
	if commit != "none" || date != "unknown" {
		fmt.Printf("  commit: %s\n  build:  %s\n", commit, date)
	}
}

func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return version
}

// runInit es el flujo principal: TUI → generar → escribir.
func runInit() {
	opts, err := tui.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "✗", err)
		os.Exit(1)
	}

	files, err := generator.Generate(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "✗", err)
		os.Exit(1)
	}

	if err := fsutil.WriteAll(opts.Dir, files); err != nil {
		fmt.Fprintln(os.Stderr, "✗", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✔ Proyecto Spring Boot generado en", opts.Dir)
	fmt.Println("   Para arrancar:")
	fmt.Println("     cd", opts.Dir)
	fmt.Println("     mvn spring-boot:run")
}

// runDoctor verifica la estructura de un proyecto generado. Solo lectura.
func runDoctor() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "✗", err)
		os.Exit(1)
	}
	if len(os.Args) > 2 {
		dir = os.Args[2]
	}
	dir, _ = filepath.Abs(dir)

	problems := doctor.Check(dir)
	if len(problems) == 0 {
		fmt.Println("✔ Todo en orden en", dir)
		return
	}
	fmt.Fprintln(os.Stderr, "Se encontraron problemas en", dir)
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, "  •", p)
	}
	os.Exit(1)
}