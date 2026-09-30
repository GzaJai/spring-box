package main

import (
	"fmt"
	"os"
	"path/filepath"

	"springy/internal/doctor"
	"springy/internal/fsutil"
	"springy/internal/generator"
	"springy/internal/tui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "doctor" {
		runDoctor()
		return
	}
	runInit()
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