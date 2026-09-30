package fsutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"springy/internal/generator"
)

const backupDir = ".springy-backup"

// WriteAll escribe los archivos generados en el destino.
//
// Patrón tipo gentle-ai: antes de sobrescribir CUALQUIER archivo existente,
// se guarda una copia en <destino>/.springy-backup (snapshot previo a cada
// escritura). Generar dos veces o generar sobre una carpeta con trabajo no
// destruye nada.
func WriteAll(dest string, files []generator.File) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("no se pudo crear %s: %w", dest, err)
	}

	escritos := 0
	backups := 0
	for _, f := range files {
		destino := filepath.Join(dest, f.Path)
		if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
			return err
		}

		// snapshot antes de tocar nada
		if info, err := os.Stat(destino); err == nil && !info.IsDir() {
			if err := guardarBackup(dest, f.Path, info.Mode()); err != nil {
				return err
			}
			backups++
		}

		if err := os.WriteFile(destino, f.Data, f.Mode); err != nil {
			return err
		}
		escritos++
	}

	if backups > 0 {
		fmt.Printf("  • %d archivo(s) existente(s) respaldado(s) en %s\n", backups, filepath.Join(dest, backupDir))
	}
	return nil
}

func guardarBackup(dest, ruta string, mode os.FileMode) error {
	back := filepath.Join(dest, backupDir)
	if err := os.MkdirAll(back, 0o755); err != nil {
		return err
	}
	origen := filepath.Join(dest, ruta)
	destino := filepath.Join(back, fmt.Sprintf("%s.%d.bak", filepath.Base(ruta), time.Now().UnixNano()))

	src, err := os.Open(origen)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(destino, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}