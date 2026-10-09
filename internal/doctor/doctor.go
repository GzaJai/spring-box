package doctor

import "os"

// Check verifica que la carpeta tenga la estructura esperada de un proyecto
// generado por spring-box. Es read-only: no modifica absolutamente nada.
//
// Devuelve la lista de problemas encontrados (vacía = todo en orden).
func Check(dir string) []string {
	var problems []string

	switch {
	case !exists(dir, "pom.xml") && !exists(dir, "build.gradle"):
		problems = append(problems, "falta el archivo de build (pom.xml o build.gradle)")
	case !exists(dir, "pom.xml") && exists(dir, "build.gradle"):
		if !exists(dir, "settings.gradle") {
			problems = append(problems, "falta settings.gradle")
		}
	}
	if !exists(dir, "src/main/java") {
		problems = append(problems, "falta src/main/java")
	}
	if !exists(dir, "src/main/resources/application.properties") {
		problems = append(problems, "falta src/main/resources/application.properties")
	}
	return problems
}

func exists(dir, name string) bool {
	_, err := os.Stat(dir + string(os.PathSeparator) + name)
	return err == nil
}