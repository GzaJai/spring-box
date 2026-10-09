package generator

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/GzaJai/spring-box/internal/model"
)

//go:embed all:templates
var templatesFS embed.FS

// File es un archivo listo para escribir en disco.
type File struct {
	Path string      // ruta relativa al destino
	Mode os.FileMode // permisos (Windows los ignora; Linux los respeta)
	Data []byte
}

// Generate materializa el template del stack elegido con las opciones del usuario.
//
// Convenciones del molde (por stack, en templates/<stack>/):
//   - los archivos con sufijo .tmpl pasan por text/template antes de escribirse;
//     los que no lo tienen se copian tal cual.
//   - la ruta final se obtiene de la ruta relativa dentro del embed, quitando
//     .tmpl y reemplazando tokens:
//       __pkg__   → carpeta del paquete Java (com/example/miapi)
//       __class__ → nombre de clase (MiapiApplication)
//       __name__  → nombre del proyecto (slug: miapi)
//
// Un stack puede tener subárboles por variante (java/ es común a todas las
// variantes; maven/ o gradle/ aportan solo su archivo de build). El generador
// combina el común + la variante elegida.
func Generate(opts model.Options) ([]File, error) {
	data := dataFor(opts)
	deps, err := depsSeleccionadas(opts.Dependencies)
	if err != nil {
		return nil, err
	}

	// JPA elegida sin driver de BD → H2 automático en dev, para que la app
	// SIEMPRE arranque (el "Failed to determine a suitable driver class" de
	// start.spring.io no pasa acá). Si el usuario eligió H2 explícitamente,
	// AutoH2 devuelve false y no se duplica.
	if model.AutoH2(opts.Dependencies) {
		for _, d := range model.Dependencies {
			if d.ID == "h2" {
				deps = append(deps, d)
				break
			}
		}
	}

	data["ExtraDeps"] = bloqueDeps(opts.BuildTool, deps)
	data["Datasource"] = datasourceProps(opts.Name, deps)

	base := path.Join("templates", opts.Stack.Name)

	var files []File
	ramas := []string{"java", opts.BuildTool}
	for _, rama := range ramas {
		if err := collect(path.Join(base, rama), path.Join(base, rama), data, &files); err != nil {
			return nil, err
		}
	}
	return files, nil
}

// datasourceProps devuelve el bloque de spring.datasource.* para
// application.properties según el motor elegido (o vacío si no aplica).
// Deuda técnica pagada: sin esto, JPA/MySQL no arranca.
func datasourceProps(name string, deps []model.Dependency) string {
	for _, d := range deps {
		switch d.DB {
		case "mysql":
			return "spring.datasource.url=jdbc:mysql://localhost:3306/" + name + "?createDatabaseIfNotExist=true\n" +
				"spring.datasource.username=root\n" +
				"spring.datasource.driver-class-name=com.mysql.cj.jdbc.Driver"
		case "postgresql":
			return "spring.datasource.url=jdbc:postgresql://localhost:5432/" + name + "\n" +
				"spring.datasource.username=postgres\n" +
				"spring.datasource.password=postgres\n" +
				"spring.datasource.driver-class-name=org.postgresql.Driver"
		case "h2":
			return "spring.datasource.url=jdbc:h2:mem:testdb\n" +
				"spring.datasource.driver-class-name=org.h2.Driver"
		}
	}
	return ""
}

// depsSeleccionadas resuelve los ids elegidos contra el catálogo curado.
func depsSeleccionadas(ids []string) ([]model.Dependency, error) {
	var deps []model.Dependency
	for _, id := range ids {
		encontrada := false
		for _, d := range model.Dependencies {
			if d.ID == id {
				deps = append(deps, d)
				encontrada = true
				break
			}
		}
		if !encontrada {
			return nil, fmt.Errorf("dependencia desconocida en el catálogo: %q", id)
		}
	}
	return deps, nil
}

// bloqueDeps construye el bloque de dependencias extra en la sintaxis del
// build tool. Sin versiones: las gestiona el BOM de la Boot version elegida.
func bloqueDeps(buildTool string, deps []model.Dependency) string {
	var sb strings.Builder
	for _, d := range deps {
		coord := d.Group + ":" + d.Artifact
		switch buildTool {
		case "gradle":
			switch {
			case d.Compile:
				sb.WriteString("\n    compileOnly '" + coord + "'")
			case d.Runtime:
				sb.WriteString("\n    runtimeOnly '" + coord + "'")
			default:
				sb.WriteString("\n    implementation '" + coord + "'")
			}
		default: // maven
			sb.WriteString("\n        <dependency>")
			sb.WriteString("\n            <groupId>" + d.Group + "</groupId>")
			sb.WriteString("\n            <artifactId>" + d.Artifact + "</artifactId>")
			if d.Runtime {
				sb.WriteString("\n            <scope>runtime</scope>")
			}
			if d.Compile {
				sb.WriteString("\n            <optional>true</optional>")
			}
			sb.WriteString("\n        </dependency>")
		}
	}
	return sb.String()
}

// dataFor precomputa todos los valores que los templates pueden interpolar.
// Los metadatos (group/artifact/package) se resuelven con defaults tipo
// Initializr: vacío → hereda (com.example, el nombre del proyecto, el
// derivado <group>.<artifact>).
func dataFor(opts model.Options) map[string]string {
	group := model.ResolveGroup(opts)
	artifact := model.ResolveArtifact(opts)
	pkg := model.ResolvePackage(opts)
	boot := opts.BootVer
	if boot == "" {
		boot = model.SpringBootVersion
	}
	class := toCamel(artifact) + "Application"
	return map[string]string{
		"Name":              opts.Name,
		"Group":             group,
		"Artifact":          artifact,
		"Package":           pkg,
		"pkgDir":            strings.ReplaceAll(pkg, ".", "/"),
		"Class":             class,
		"BuildTool":         opts.BuildTool,
		"JavaVersion":       opts.JavaVer,
		"IntelliJName":      opts.JavaVer, // nombre del SDK para .idea/misc.xml (matchea los SDK nombrados por versión)
		"Packaging":         opts.Packaging,
		"SpringBootVersion": boot,
	}
}

// collect recorre el árbol de templates recursivamente.
// stackDir es el prefijo interno que se elimina de la ruta destino final
// (la variante interna de spring-box no viaja al proyecto).
func collect(tplDir, stackDir string, data map[string]string, out *[]File) error {
	entries, err := fs.ReadDir(templatesFS, tplDir)
	if err != nil {
		return fmt.Errorf("no se encontró el árbol de templates %q: %w", tplDir, err)
	}

	for _, e := range entries {
		rel := path.Join(tplDir, e.Name())
		if e.IsDir() {
			if err := collect(rel, stackDir, data, out); err != nil {
				return err
			}
			continue
		}

		raw, err := fs.ReadFile(templatesFS, rel)
		if err != nil {
			return err
		}

		// solo los .tmpl se renderizan; el resto se copia tal cual
		if strings.HasSuffix(e.Name(), ".tmpl") {
			tpl, err := template.New(e.Name()).
				Funcs(template.FuncMap{"eq": func(a, b string) bool { return a == b }}).
				Parse(string(raw))
			if err != nil {
				return fmt.Errorf("template %s: %w", rel, err)
			}
			var sb strings.Builder
			if err := tpl.Execute(&sb, data); err != nil {
				return fmt.Errorf("render %s: %w", rel, err)
			}
			raw = []byte(sb.String())
		}

		destino := strings.TrimSuffix(rel, ".tmpl")
		destino = strings.TrimPrefix(destino, stackDir+"/")
		destino = strings.ReplaceAll(destino, "__pkg__", data["pkgDir"])
		destino = strings.ReplaceAll(destino, "__class__", data["Class"])
		destino = strings.ReplaceAll(destino, "__name__", data["Name"])

		*out = append(*out, File{Path: destino, Mode: 0o644, Data: raw})
	}
	return nil
}

// toCamel convierte un slug kebab-case a CamelCase: mi-api → MiApi.
func toCamel(slug string) string {
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}