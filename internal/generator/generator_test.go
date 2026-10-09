package generator

import (
	"strings"
	"testing"

	"github.com/GzaJai/spring-box/internal/model"
)

func TestToCamel(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"mi-api", "MiApi"},
		{"a", "A"},
		{"x-y-z", "XYZ"},
		{"yui-backend", "YuiBackend"},
		{"prueba1", "Prueba1"},
		{"so-lo-una", "SoLoUna"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := toCamel(tt.in); got != tt.want {
				t.Errorf("toCamel(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	tests := []struct {
		name      string
		opts      model.Options
		wantFiles []string // rutas que DEBEN existir
		notFiles  []string // rutas que NO deben existir
		check     map[string][]string // path → substrings obligatorios
		notCheck  map[string][]string // path → substrings que NO deben estar
		countCheck map[string]map[string]int // path → sub → veces exactas esperadas
		wantErr   bool
	}{
		{
			name: "maven jar java17",
			opts: model.Options{Name: "mi-api", Stack: model.Stacks[0], BuildTool: "maven", JavaVer: "17", Packaging: "jar"},
			wantFiles: []string{
				"pom.xml",
				".idea/misc.xml",
				"src/main/java/com/example/miapi/MiApiApplication.java",
				"src/test/java/com/example/miapi/MiApiApplicationTests.java",
				"src/main/resources/application.properties",
				".gitignore",
			},
			notFiles: []string{"build.gradle", "settings.gradle"},
check: map[string][]string{
				"pom.xml": {
					"<packaging>jar</packaging>",
					"<java.version>17</java.version>",
					"<maven.compiler.release>17</maven.compiler.release>",
					"<version>4.1.1</version>", // parent sin ".RELEASE" — debe existir en Maven Central
					"<artifactId>mi-api</artifactId>",
					"<groupId>com.example</groupId>",
				},
				".idea/misc.xml": {
					`languageLevel="JDK_17"`,
					`project-jdk-name="17"`,
				},
				"src/main/java/com/example/miapi/MiApiApplication.java": {
					"package com.example.miapi;",
					"public class MiApiApplication",
					"@SpringBootApplication",
				},
			},
		},
		{
			name: "maven war java21",
			opts: model.Options{Name: "yui-backend", Stack: model.Stacks[0], BuildTool: "maven", JavaVer: "21", Packaging: "war"},
			wantFiles: []string{
				"pom.xml",
				"src/main/java/com/example/yuibackend/YuiBackendApplication.java",
			},
			notFiles: []string{"build.gradle"},
			check: map[string][]string{
				"pom.xml": {"<packaging>war</packaging>", "<java.version>21</java.version>"},
			},
		},
		{
			name: "gradle jar",
			opts: model.Options{Name: "api-uno", Stack: model.Stacks[0], BuildTool: "gradle", JavaVer: "21", Packaging: "jar"},
			wantFiles: []string{
				"build.gradle",
				"settings.gradle",
				"src/main/java/com/example/apiuno/ApiUnoApplication.java",
			},
			notFiles: []string{"pom.xml"},
			check: map[string][]string{
				"build.gradle": {
					"io.spring.dependency-management",
					"JavaLanguageVersion.of(21)",
					"spring-boot-starter-web",
					"version '4.1.1'",
				},
				"settings.gradle": {"rootProject.name = 'api-uno'"},
			},
		},
		{
			name: "gradle war",
			opts: model.Options{Name: "web-app", Stack: model.Stacks[0], BuildTool: "gradle", JavaVer: "17", Packaging: "war"},
			wantFiles: []string{"build.gradle"},
			notFiles:  []string{"pom.xml"},
			check: map[string][]string{
				"build.gradle": {"id 'war'"},
			},
		},
		{
			name: "maven con dependencias",
			opts: model.Options{Name: "app-db", Stack: model.Stacks[0], BuildTool: "maven", JavaVer: "17", Packaging: "jar",
				Dependencies: []string{"security", "postgresql", "lombok"}},
			wantFiles: []string{"pom.xml"},
			check: map[string][]string{
				"pom.xml": {
					"<artifactId>spring-boot-starter-security</artifactId>",
					"<groupId>org.postgresql</groupId>",
					"<artifactId>postgresql</artifactId>",
					"<scope>runtime</scope>",
					"<artifactId>lombok</artifactId>",
					"<optional>true</optional>",
				},
			},
			notCheck: map[string][]string{
				"pom.xml": {"<artifactId>spring-boot-starter-data-jpa</artifactId>"},
			},
		},
		{
			name: "gradle con compileOnly y runtimeOnly",
			opts: model.Options{Name: "api-sec", Stack: model.Stacks[0], BuildTool: "gradle", JavaVer: "21", Packaging: "jar",
				Dependencies: []string{"lombok", "devtools"}},
			wantFiles: []string{"build.gradle"},
			check: map[string][]string{
				"build.gradle": {
					"compileOnly 'org.projectlombok:lombok'",
					"runtimeOnly 'org.springframework.boot:spring-boot-devtools'",
				},
			},
		},
		{
			name: "sin dependencias no agrega bloques extra",
			opts: model.Options{Name: "limpio", Stack: model.Stacks[0], BuildTool: "maven", JavaVer: "17", Packaging: "jar"},
			wantFiles: []string{"pom.xml"},
			check: map[string][]string{
				"pom.xml": {
					"<artifactId>spring-boot-starter-web</artifactId>",
					"<artifactId>spring-boot-starter-test</artifactId>",
				},
			},
			notCheck: map[string][]string{
				"pom.xml": {"<optional>true</optional>", "<scope>runtime</scope>"},
			},
		},
		{
			name: "dependencia desconocida falla",
			opts: model.Options{Name: "raro", Stack: model.Stacks[0], BuildTool: "maven", JavaVer: "17", Packaging: "jar",
				Dependencies: []string{"stripe"}},
			wantErr: true,
		},
		{
			name: "jpa sin driver agrega H2 y datasource de memoria",
			opts: model.Options{Name: "app-mem", Stack: model.Stacks[0], BuildTool: "maven", JavaVer: "17", Packaging: "jar",
				Dependencies: []string{"data-jpa"}},
			wantFiles: []string{"pom.xml", "src/main/resources/application.properties"},
			check: map[string][]string{
				"pom.xml": {"<groupId>com.h2database</groupId>", "<artifactId>h2</artifactId>", "<scope>runtime</scope>"},
				"src/main/resources/application.properties": {
					"spring.datasource.url=jdbc:h2:mem:testdb",
					"spring.datasource.driver-class-name=org.h2.Driver",
				},
			},
		},
		{
			name: "jpa con postgresql configura el datasource real",
			opts: model.Options{Name: "app-db", Stack: model.Stacks[0], BuildTool: "maven", JavaVer: "17", Packaging: "jar",
				Dependencies: []string{"data-jpa", "postgresql"}},
			wantFiles: []string{"pom.xml", "src/main/resources/application.properties"},
			check: map[string][]string{
				"src/main/resources/application.properties": {
					"spring.datasource.url=jdbc:postgresql://localhost:5432/app-db",
					"spring.datasource.username=postgres",
					"spring.datasource.driver-class-name=org.postgresql.Driver",
				},
			},
			notCheck: map[string][]string{
				"pom.xml": {"com.h2database"},
			},
		},
		{
			name: "metadatos custom (group/artifact/package/boot)",
			opts: model.Options{Name: "demo", Stack: model.Stacks[0], BuildTool: "maven", JavaVer: "21", Packaging: "jar",
				BootVer: "4.2.0-M2", Group: "ar.com.yui", Artifact: "transporte-api", Package: "ar.com.yui.transporte"},
			wantFiles: []string{
				"pom.xml",
				"src/main/java/ar/com/yui/transporte/TransporteApiApplication.java",
			},
			check: map[string][]string{
				"pom.xml": {
					"<groupId>ar.com.yui</groupId>",
					"<artifactId>transporte-api</artifactId>",
					"<version>4.2.0-M2</version>",
				},
				"src/main/java/ar/com/yui/transporte/TransporteApiApplication.java": {
					"package ar.com.yui.transporte;",
					"public class TransporteApiApplication",
				},
			},
			notCheck: map[string][]string{
				"pom.xml": {"<groupId>com.example</groupId>"},
			},
		},
		{
			name: "h2 elegido como opción no se duplica",
			opts: model.Options{Name: "app-h2", Stack: model.Stacks[0], BuildTool: "maven", JavaVer: "17", Packaging: "jar",
				Dependencies: []string{"data-jpa", "h2"}},
			wantFiles: []string{"pom.xml", "src/main/resources/application.properties"},
			check: map[string][]string{
				"pom.xml": {"<groupId>com.h2database</groupId>", "<artifactId>h2</artifactId>", "<scope>runtime</scope>"},
				"src/main/resources/application.properties": {
					"spring.datasource.url=jdbc:h2:mem:testdb",
					"spring.datasource.driver-class-name=org.h2.Driver",
				},
			},
			countCheck: map[string]map[string]int{
				"pom.xml": {"com.h2database": 1}, // exactamente una vez, sin duplicados
			},
		},
		{
			name: "jpa con mysql configura el datasource mysql",
			opts: model.Options{Name: "app-mysql", Stack: model.Stacks[0], BuildTool: "gradle", JavaVer: "21", Packaging: "jar",
				Dependencies: []string{"data-jpa", "mysql"}},
			wantFiles: []string{"build.gradle", "src/main/resources/application.properties"},
			check: map[string][]string{
				"src/main/resources/application.properties": {
					"spring.datasource.url=jdbc:mysql://localhost:3306/app-mysql?createDatabaseIfNotExist=true",
					"spring.datasource.driver-class-name=com.mysql.cj.jdbc.Driver",
				},
				"build.gradle": {"runtimeOnly 'com.mysql:mysql-connector-j'"},
			},
			notCheck: map[string][]string{
				"build.gradle": {"com.h2database"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := Generate(tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("se esperaba un error y no llegó")
				}
				return
			}
			if err != nil {
				t.Fatalf("Generate() error: %v", err)
			}

			got := make(map[string]string, len(files))
			for _, f := range files {
				got[f.Path] = string(f.Data)
			}

			for _, p := range tt.wantFiles {
				if _, ok := got[p]; !ok {
					t.Errorf("falta el archivo generado %q", p)
				}
			}
			for _, p := range tt.notFiles {
				if _, ok := got[p]; ok {
					t.Errorf("no debería generarse %q", p)
				}
			}
			for path, subs := range tt.check {
				content, ok := got[path]
				if !ok {
					t.Errorf("no se pudo verificar %q: no existe", path)
					continue
				}
				for _, sub := range subs {
					if !strings.Contains(content, sub) {
						t.Errorf("%s no contiene %q", path, sub)
					}
				}
			}
			for path, subs := range tt.notCheck {
				content, ok := got[path]
				if !ok {
					continue
				}
				for _, sub := range subs {
					if strings.Contains(content, sub) {
						t.Errorf("%s NO debería contener %q", path, sub)
					}
				}
			}
			for path, counts := range tt.countCheck {
				content, ok := got[path]
				if !ok {
					continue
				}
				for sub, want := range counts {
					if gotCount := strings.Count(content, sub); gotCount != want {
						t.Errorf("%s contiene %q %d veces, esperado %d", path, sub, gotCount, want)
					}
				}
			}
		})
	}
}