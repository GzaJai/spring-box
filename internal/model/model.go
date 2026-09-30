package model

import "strings"

// SpringBootVersion es la versión por defecto del proyecto generado.
// Única fuente de verdad (la usan generator y la TUI).
// OJO: debe existir SIEMPRE en Maven Central — el sufijo ".RELEASE" es un
// remanente de Boot 1.x que YA NO existe (4.1.1.RELEASE → fail a 1.5).
const SpringBootVersion = "4.1.1"

// BootVersion es una opción de versión del selector del wizard.
type BootVersion struct {
	Label   string // lo que se ve en la TUI (con "(default)" en la marcada)
	Version string // la versión pura que va al pom/build.gradle
}

// BootVersions es la lista configurable de versiones. Verificadas contra
// Maven Central (repo1.maven.org) — la Lección .RELEASE: no confiar en el
// metadata de start.spring.io, validar siempre. La GA estable es la default.
var BootVersions = []BootVersion{
	{Label: "4.1.1 (GA, estable — default)", Version: "4.1.1"},
	{Label: "4.2.0-M2 (milestone)", Version: "4.2.0-M2"},
	{Label: "4.2.0-M1 (milestone)", Version: "4.2.0-M1"},
}

// Stack describe un tipo de proyecto que springy puede generar.
type Stack struct {
	Name        string // identificador interno; también el nombre de la carpeta en templates/
	Label       string // texto corto que se muestra en la TUI
	Description string // ayuda de una línea en la TUI
}

// Stacks es el catálogo disponible. Cada entrada tiene su carpeta templates/<Name>.
// Agregar un stack nuevo = agregar una entrada acá + su carpeta de templates.
var Stacks = []Stack{
	{
		Name:        "spring-boot",
		Label:       "Spring Boot",
		Description: "pom.xml + app Java + tests. Limpio, como start.spring.io, sin el zip.",
	},
}

// Options es lo que el usuario elige en la TUI y lo que el generador materializa.
// Es el "calco" de las opciones de start.spring.io (Project, Packaging, Java, ...).
type Options struct {
	Name     string   // nombre del proyecto (display)
	Stack    Stack    // tipo de proyecto elegido
	BuildTool    string // maven | gradle  (Project en Initializr)
	BootVer      string // versión de Spring Boot elegida (vacío = SpringBootVersion)
	Group        string // groupId Maven (vacío = com.example)
	Artifact     string // artifactId Maven (vacío = Name)
	Package      string // packageName Java (vacío = Group.Artifact sin guiones)
	JavaVer      string // 17 | 21          (Java version / SDK)
	Packaging    string // jar | war         (Packaging / empaquetador)
	Dependencies []string // ids de dependencias elegidas (el catálogo de abajo)
	Dir          string   // ruta absoluta donde se genera
}

// ResolveGroup aplica el default del group cuando está vacío.
func ResolveGroup(opts Options) string {
	if opts.Group != "" {
		return opts.Group
	}
	return "com.example"
}

// ResolveArtifact aplica el default del artifact (el nombre del proyecto).
func ResolveArtifact(opts Options) string {
	if opts.Artifact != "" {
		return opts.Artifact
	}
	return opts.Name
}

// ResolvePackage aplica el default del package: <group>.<artifact sin guiones>.
func ResolvePackage(opts Options) string {
	if opts.Package != "" {
		return opts.Package
	}
	return ResolveGroup(opts) + "." + strings.ReplaceAll(ResolveArtifact(opts), "-", "")
}

// Dependency describe una dependencia del catálogo curado.
// NO lleva versión: todas las coordenadas del catálogo están manageadas por
// el BOM de Spring Boot (spring-boot-dependencies via parent/dependency-
// management), así que la versión la decide la Boot version elegida — cero
// drift de versiones (el mismo criterio del catálogo oficial de start.spring.io).
type Dependency struct {
	ID       string // identificador interno (se guarda en Options.Dependencies)
	Group    string // groupId Maven
	Artifact string // artifactId Maven
	Label    string // texto corto para la TUI
	Desc     string // ayuda de una línea para la TUI
	Runtime  bool   // scope runtime (drivers de DB, devtools → runtimeOnly/scope runtime)
	Compile  bool   // solo compilación (lombok → compileOnly / optional)
	DB       string // si trae base de datos propia: "mysql" | "postgresql" ("" = no)
}

// Dependencies es el catálogo curado v1. Web y Test quedan SIEMPRE incluidas
// (son la base funcional del proyecto); esto es lo que el usuario puede sumar.
var Dependencies = []Dependency{
	{
		ID: "security", Group: "org.springframework.boot", Artifact: "spring-boot-starter-security",
		Label: "Spring Security", Desc: "Autenticación y autorización",
	},
	{
		ID: "actuator", Group: "org.springframework.boot", Artifact: "spring-boot-starter-actuator",
		Label: "Actuator", Desc: "Métricas y health checks de producción",
	},
	{
		ID: "validation", Group: "org.springframework.boot", Artifact: "spring-boot-starter-validation",
		Label: "Validation", Desc: "Validación de datos con Bean Validation",
	},
	{
		ID: "data-jpa", Group: "org.springframework.boot", Artifact: "spring-boot-starter-data-jpa",
		Label: "JPA", Desc: "Persistencia con Hibernate (ORM)",
	},
	{
		ID: "postgresql", Group: "org.postgresql", Artifact: "postgresql",
		Label: "PostgreSQL", Desc: "Driver JDBC de PostgreSQL", Runtime: true, DB: "postgresql",
	},
	{
		ID: "h2", Group: "com.h2database", Artifact: "h2",
		Label: "H2 (en memoria)", Desc: "Base en memoria, ideal para desarrollo", Runtime: true, DB: "h2",
	},
	{
		ID: "mysql", Group: "com.mysql", Artifact: "mysql-connector-j",
		Label: "MySQL", Desc: "Driver JDBC de MySQL", Runtime: true, DB: "mysql",
	},
	{
		ID: "lombok", Group: "org.projectlombok", Artifact: "lombok",
		Label: "Lombok", Desc: "Menos boilerplate (getters, builders...)", Compile: true,
	},
	{
		ID: "devtools", Group: "org.springframework.boot", Artifact: "spring-boot-devtools",
		Label: "DevTools", Desc: "Reinicio automático en desarrollo", Runtime: true,
	},
	{
		ID: "thymeleaf", Group: "org.springframework.boot", Artifact: "spring-boot-starter-thymeleaf",
		Label: "Thymeleaf", Desc: "Template engine para vistas HTML",
	},
	{
		ID: "data-redis", Group: "org.springframework.boot", Artifact: "spring-boot-starter-data-redis",
		Label: "Redis", Desc: "Cache y datos en memoria con Redis",
	},
}

// AutoH2 devuelve true cuando corresponde activar H2 automáticamente:
// JPA elegida SIN driver de base de datos (mysql/postgresql/h2 explícito).
// Sin esto, Spring Boot intenta autoconfigurar un DataSource y revienta con
// "Failed to determine a suitable driver class" (el proyecto no arranca).
func AutoH2(ids []string) bool {
	hasJPA, hasDB := false, false
	for _, id := range ids {
		switch id {
		case "data-jpa":
			hasJPA = true
		case "mysql", "postgresql", "h2":
			hasDB = true
		}
	}
	return hasJPA && !hasDB
}

// TieneDB devuelve el motor de base de datos elegido ("", "mysql",
// "postgresql", "h2").
func TieneDB(ids []string) string {
	for _, id := range ids {
		switch id {
		case "mysql":
			return "mysql"
		case "postgresql":
			return "postgresql"
		case "h2":
			return "h2"
		}
	}
	return ""
}