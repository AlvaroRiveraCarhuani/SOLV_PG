//go:build integration

package docker

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

// TestLinkWorkspaceDependenciesIntegration ejercita el enlace contra Docker de
// verdad: crea un volumen con el diario de inicialización, un árbol con
// node_modules y comprueba que los archivos llegan al volumen y que repetir el
// enlace con la misma clave no falla.
func TestLinkWorkspaceDependenciesIntegration(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("no se pudo crear el cliente de Docker: %v", err)
	}
	ctx := context.Background()
	volume := "solv-deps-link-" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))

	if err := client.EnsureVolumeExists(ctx, volume); err != nil {
		t.Fatalf("no se pudo crear el volumen de prueba: %v", err)
	}
	t.Cleanup(func() {
		_ = exec.Command("docker", "volume", "rm", "-f", volume).Run()
	})

	if err := client.WritePracticeEnvironmentVolumeFiles(ctx, volume, map[string]string{
		".solv_workspace/init.json": `{"version_id":"v1","content_sha256":"h","state":"verified"}`,
	}); err != nil {
		t.Fatalf("no se pudo sembrar el diario del volumen: %v", err)
	}

	root := t.TempDir()
	packageDir := filepath.Join(root, "node_modules", "demo")
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatalf("no se pudo crear el árbol: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packageDir, "index.js"), []byte("module.exports = 1\n"), 0o644); err != nil {
		t.Fatalf("no se pudo escribir el paquete: %v", err)
	}

	tree := domain.WorkspaceDependencyTree{
		RootPath: root,
		Key:      strings.Repeat("b", 64),
		Paths:    []string{"node_modules"},
	}
	if err := client.LinkWorkspaceDependencies(ctx, volume, tree); err != nil {
		t.Fatalf("el enlace debía completarse: %v", err)
	}

	files, err := client.readWorkspaceVolumeTree(ctx, volume, "", 1<<20, 64)
	if err != nil {
		t.Fatalf("no se pudo leer el volumen: %v", err)
	}
	found := false
	for _, file := range files {
		if file.Path == "node_modules/demo/index.js" {
			found = true
			if string(file.Content) != "module.exports = 1\n" {
				t.Fatalf("el contenido enlazado no coincide: %q", file.Content)
			}
		}
	}
	if !found {
		t.Fatalf("el paquete no llegó al volumen; archivos: %v", files)
	}

	// Repetir con la misma clave es idempotente y no vuelve a copiar.
	if err := client.LinkWorkspaceDependencies(ctx, volume, tree); err != nil {
		t.Fatalf("repetir el enlace debía ser inocuo: %v", err)
	}
}

// TestLinkWorkspaceDependenciesAcrossVolumesIntegration comprueba el enlace de
// la misma clave en dos volúmenes: el contenido llega a los dos, reemplazar un
// archivo en uno no toca al otro ni al árbol preparado, y borrar un volumen no
// deja incompleta la clave.
//
// Medido: con Docker cada volumen es un montaje distinto y el kernel rechaza el
// enlace duro entre montajes con EXDEV, aunque compartan superbloque. La prueba
// mide el resultado real —copias independientes— y solo exige inodo compartido
// cuando el árbol y el volumen caen en el mismo montaje.
func TestLinkWorkspaceDependenciesAcrossVolumesIntegration(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("no se pudo crear el cliente de Docker: %v", err)
	}
	ctx := context.Background()
	suffix := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	firstVolume := "solv-deps-copy-a-" + suffix
	secondVolume := "solv-deps-copy-b-" + suffix

	for _, volume := range []string{firstVolume, secondVolume} {
		volume := volume
		if err := client.EnsureVolumeExists(ctx, volume); err != nil {
			t.Fatalf("no se pudo crear %s: %v", volume, err)
		}
		t.Cleanup(func() { _ = exec.Command("docker", "volume", "rm", "-f", volume).Run() })
		if err := client.WritePracticeEnvironmentVolumeFiles(ctx, volume, map[string]string{
			".solv_workspace/init.json": `{"version_id":"v1","content_sha256":"h","state":"verified"}`,
		}); err != nil {
			t.Fatalf("no se pudo sembrar el diario de %s: %v", volume, err)
		}
	}

	// El árbol tiene que vivir en el mismo sistema de archivos que los volúmenes:
	// con el temporal en /tmp —tmpfs en muchos nodos— `cp -al` no puede enlazar y
	// el enlace cae a copia. Por eso la prueba lo crea junto al paquete, en el
	// mismo disco que la raíz de Docker, y declara el requisito si no coincide.
	root, err := os.MkdirTemp(".", "solv-deps-tree-")
	if err != nil {
		t.Fatalf("no se pudo crear el árbol de prueba: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	if root, err = filepath.Abs(root); err != nil {
		t.Fatalf("no se pudo resolver la ruta del árbol: %v", err)
	}

	demoDir := filepath.Join(root, "node_modules", "demo")
	if err := os.MkdirAll(demoDir, 0o755); err != nil {
		t.Fatalf("no se pudo crear el árbol: %v", err)
	}
	const original = "module.exports = 1\n"
	const relative = "node_modules/demo/index.js"
	if err := os.WriteFile(filepath.Join(demoDir, "index.js"), []byte(original), 0o644); err != nil {
		t.Fatalf("no se pudo escribir el paquete: %v", err)
	}
	if err := os.WriteFile(filepath.Join(demoDir, "package.json"), []byte("{\"name\":\"demo\"}\n"), 0o644); err != nil {
		t.Fatalf("no se pudo escribir el manifiesto: %v", err)
	}

	tree := domain.WorkspaceDependencyTree{
		RootPath: root,
		Key:      strings.Repeat("c", 64),
		Paths:    []string{"node_modules"},
	}
	for _, volume := range []string{firstVolume, secondVolume} {
		if err := client.LinkWorkspaceDependencies(ctx, volume, tree); err != nil {
			t.Fatalf("el enlace de %s debía completarse: %v", volume, err)
		}
	}

	// Los dos volúmenes reciben el contenido declarado antes de medir nada.
	dependencyTestAssertVolumeFile(t, client, firstVolume, relative, original)
	dependencyTestAssertVolumeFile(t, client, secondVolume, relative, original)

	// Mismo dispositivo e inodo en los dos volúmenes y en el árbol: una sola
	// copia física de los bytes, con un enlace por destino.
	firstIdentity, firstLinks := dependencyTestVolumeStat(t, firstVolume, relative)
	secondIdentity, _ := dependencyTestVolumeStat(t, secondVolume, relative)
	treeIdentity := dependencyTestTreeStat(t, root, relative)
	shared := dependencyTestSameDevice(firstIdentity, treeIdentity)
	if shared {
		if firstIdentity != secondIdentity || firstIdentity != treeIdentity {
			t.Fatalf("los dos volúmenes y el árbol debían compartir el archivo: %s, %s, %s",
				firstIdentity, secondIdentity, treeIdentity)
		}
		if firstLinks != 3 {
			t.Fatalf("el archivo debía tener tres enlaces (árbol y dos volúmenes), tiene %d", firstLinks)
		}
	} else {
		// Caso medido en Docker: el árbol y el volumen son montajes distintos y
		// el enlace cae a copia. Cada volumen tiene su propio archivo y el
		// contenido sigue siendo el mismo.
		if firstIdentity == treeIdentity || firstIdentity == secondIdentity {
			t.Fatalf("sin enlaces duros los archivos no debían compartirse: %s, %s, %s",
				firstIdentity, secondIdentity, treeIdentity)
		}
		t.Logf("árbol (%s) y volúmenes (%s) en filesistemas distintos: se verificó la caída a copia", treeIdentity, firstIdentity)
	}

	// Reemplazar el archivo rompe el enlace de ese volumen: el otro y el árbol
	// conservan el original. Se escribe un archivo nuevo y se mueve encima, que
	// es lo que hace un editor; escribir en el lugar tocaría los tres enlaces.
	dependencyTestDockerRun(t, "run", "--rm",
		"-v", firstVolume+":"+environmentVolumeMountPoint+":rw",
		environmentVolumeHelperImage,
		"sh", "-c", "printf '%s' 'module.exports = 2' > "+shellQuote(environmentVolumeMountPoint+"/"+relative+".tmp")+
			" && mv "+shellQuote(environmentVolumeMountPoint+"/"+relative+".tmp")+" "+
			shellQuote(environmentVolumeMountPoint+"/"+relative))
	dependencyTestAssertVolumeFile(t, client, firstVolume, relative, "module.exports = 2")
	dependencyTestAssertVolumeFile(t, client, secondVolume, relative, original)
	dependencyTestAssertTreeFile(t, root, relative, original)

	// Borrar el primer volumen no deja incompleta la clave: el segundo volumen y
	// el árbol siguen enteros, y el archivo vuelve a tener dos enlaces.
	if out, err := exec.Command("docker", "volume", "rm", "-f", firstVolume).CombinedOutput(); err != nil {
		t.Fatalf("no se pudo borrar %s: %v\n%s", firstVolume, err, out)
	}
	dependencyTestAssertVolumeFile(t, client, secondVolume, relative, original)
	dependencyTestAssertTreeFile(t, root, relative, original)
	_, linksAfterRemoval := dependencyTestVolumeStat(t, secondVolume, relative)
	wantLinks := 1
	if shared {
		wantLinks = 2
	}
	if linksAfterRemoval != wantLinks {
		t.Fatalf("tras borrar un volumen el archivo debía conservar %d enlaces, tiene %d", wantLinks, linksAfterRemoval)
	}
}

// dependencyTestSameDevice compara el dispositivo de dos identidades
// `dispositivo:inodo` obtenidas dentro de contenedores.
func dependencyTestSameDevice(first, second string) bool {
	return strings.SplitN(first, ":", 2)[0] == strings.SplitN(second, ":", 2)[0]
}

func dependencyTestDockerRun(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("docker %s falló: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// dependencyTestVolumeStat devuelve dispositivo:inodo y número de enlaces de un
// archivo del volumen, que es la prueba de que los bytes no se duplican.
func dependencyTestVolumeStat(t *testing.T, volume, relativePath string) (string, int) {
	t.Helper()
	out := dependencyTestDockerRun(t, "run", "--rm",
		"-v", volume+":"+environmentVolumeMountPoint+":ro",
		environmentVolumeHelperImage,
		"stat", "-c", "%d:%i %h", environmentVolumeMountPoint+"/"+relativePath)
	fields := strings.Fields(out)
	if len(fields) != 2 {
		t.Fatalf("salida inesperada de stat: %q", out)
	}
	links, err := strconv.Atoi(fields[1])
	if err != nil {
		t.Fatalf("no se pudo leer el número de enlaces: %q", out)
	}
	return fields[0], links
}

func dependencyTestTreeStat(t *testing.T, treeRoot, relativePath string) string {
	t.Helper()
	return dependencyTestDockerRun(t, "run", "--rm",
		"-v", treeRoot+":"+preparedTreeMountPoint+":ro",
		environmentVolumeHelperImage,
		"stat", "-c", "%d:%i", preparedTreeMountPoint+"/"+relativePath)
}

func dependencyTestAssertVolumeFile(t *testing.T, client *Client, volume, relativePath, want string) {
	t.Helper()
	files, err := client.readWorkspaceVolumeTree(context.Background(), volume, "", 1<<20, 64)
	if err != nil {
		t.Fatalf("no se pudo leer %s: %v", volume, err)
	}
	for _, file := range files {
		if file.Path == relativePath {
			if string(file.Content) != want {
				t.Fatalf("%s en %s cambió sin motivo: %q", relativePath, volume, file.Content)
			}
			return
		}
	}
	t.Fatalf("%s no está en %s; archivos: %v", relativePath, volume, files)
}

func dependencyTestAssertTreeFile(t *testing.T, treeRoot, relativePath, want string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(treeRoot, relativePath))
	if err != nil {
		t.Fatalf("el árbol preparado perdió %s: %v", relativePath, err)
	}
	if string(content) != want {
		t.Fatalf("%s en el árbol preparado cambió sin motivo: %q", relativePath, content)
	}
}
