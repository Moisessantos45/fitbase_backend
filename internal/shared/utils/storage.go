package utils

import (
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	validIDPattern  = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	validExtPattern = regexp.MustCompile(`^\.[a-zA-Z0-9]+$`)
)

func AppDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		dir, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return dir, nil
	}

	dir := filepath.Dir(exe)
	if strings.Contains(dir, "go-build") {
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	return dir, nil
}

// SaveFile guarda un archivo de forma segura dentro de cmd/uploads/<dir>.
// Genera un UUID único si fileId viene vacío, sanitiza rutas para evitar Path Traversal
// y aplica permisos de lectura sin ejecución (0644).
func SaveFile(dir string, fileId string, file *multipart.FileHeader, c *gin.Context) (string, string, error) {
	if file == nil {
		return "", "", errors.New("archivo es requerido")
	}

	appDir, err := AppDir()
	if err != nil {
		return "", "", fmt.Errorf("app dir: %w", err)
	}

	baseUploadsDir := filepath.Clean(filepath.Join(appDir, "cmd", "uploads"))

	// 1. Sanitizar y validar directorio relativo (evitar path traversal)
	cleanDir := filepath.Clean(filepath.ToSlash(dir))
	if strings.HasPrefix(cleanDir, "/") || strings.HasPrefix(cleanDir, "../") || cleanDir == ".." {
		return "", "", errors.New("ruta de directorio inválida o no permitida")
	}

	uploadsDir := filepath.Clean(filepath.Join(baseUploadsDir, filepath.FromSlash(cleanDir)))

	// Comprobar que uploadsDir esté estrictamente dentro de baseUploadsDir
	relDir, err := filepath.Rel(baseUploadsDir, uploadsDir)
	if err != nil || strings.HasPrefix(relDir, "..") {
		return "", "", errors.New("ruta de directorio insegura fuera de almacenamiento")
	}

	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", "", fmt.Errorf("mkdir: %w", err)
	}

	// 2. Generar UUID aleatorio criptográficamente seguro o sanitizar ID proporcionado
	id := strings.TrimSpace(fileId)
	if id == "" {
		id = uuid.New().String()
	} else {
		id = filepath.Base(filepath.Clean(id))
		if !validIDPattern.MatchString(id) {
			return "", "", errors.New("fileId contiene caracteres inválidos")
		}
	}

	// 3. Sanitizar extensión
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != "" && !validExtPattern.MatchString(ext) {
		return "", "", errors.New("extensión de archivo no permitida")
	}

	// 4. Construir nombre seguro (UUID + extensión, sin nombres originales del cliente)
	filename := fmt.Sprintf("%s%s", id, ext)
	dst := filepath.Clean(filepath.Join(uploadsDir, filename))

	// Comprobar contención en uploadsDir
	relDst, err := filepath.Rel(uploadsDir, dst)
	if err != nil || strings.HasPrefix(relDst, "..") {
		return "", "", errors.New("ruta de destino insegura")
	}

	if err := c.SaveUploadedFile(file, dst); err != nil {
		return "", "", fmt.Errorf("save file: %w", err)
	}

	// 5. Aplicar permisos restrictivos (0644: lectura/escritura dueño, lectura otros, NO ejecutable)
	_ = os.Chmod(dst, 0644)

	return id, dst, nil
}

// DeleteFile elimina un archivo verificando que se encuentre dentro del directorio de almacenamiento.
func DeleteFile(filePath string) error {
	cleanPath := strings.TrimSpace(filePath)
	if cleanPath == "" {
		return errors.New("ruta de archivo requerida")
	}

	appDir, err := AppDir()
	if err != nil {
		return fmt.Errorf("app dir: %w", err)
	}

	baseUploadsDir := filepath.Clean(filepath.Join(appDir, "cmd", "uploads"))
	targetPath := filepath.Clean(cleanPath)

	// Prevenir eliminación arbitraria fuera del directorio uploads
	rel, err := filepath.Rel(baseUploadsDir, targetPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return errors.New("operación denegada: la ruta no pertenece al directorio de almacenamiento")
	}

	if err := os.Remove(targetPath); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	return nil
}

// ServeFileSecurely envía un archivo al cliente con cabeceras de seguridad contra MIME-sniffing y ejecución embebida.
func ServeFileSecurely(c *gin.Context, filePath string, downloadName string) {
	c.Header("X-Content-Type-Options", "nosniff")
	if strings.TrimSpace(downloadName) != "" {
		cleanName := filepath.Base(filepath.Clean(downloadName))
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", cleanName))
	} else {
		c.Header("Content-Disposition", "inline")
	}
	c.File(filePath)
}

