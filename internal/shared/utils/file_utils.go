package utils

import (
	"bytes"
	"fitbase/internal/models"

	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	MaxImageSize  = 5 << 20  // 5MB por imagen
	MaxBatchFiles = 10       // Máximo 10 archivos simultáneos por lote
	MaxPDFSize    = 10 << 20 // 10MB por PDF
	MaxExcelSize  = 10 << 20 // 10MB por archivo Excel
)

// readHeaderBytes lee hasta 512 bytes del encabezado de un archivo para detección de tipo MIME real.
func readHeaderBytes(file *multipart.FileHeader) ([]byte, error) {
	if file == nil {
		return nil, errors.New("archivo no proporcionado")
	}

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("error al abrir archivo: %w", err)
	}
	defer src.Close()

	buf := make([]byte, 512)
	n, err := src.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error al leer encabezado del archivo: %w", err)
	}
	if n == 0 {
		return nil, errors.New("el archivo está vacío")
	}
	return buf[:n], nil
}

func ExtractFileIdFromURL(filename string) string {
	if strings.TrimSpace(filename) == "" {
		return ""
	}

	parts := strings.Split(filename, "/")
	if len(parts) == 0 {
		return ""
	}

	// Retorna el último elemento del slice, que es el ID del archivo
	return parts[len(parts)-1]
}

func ExtractFileId(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// ValidateImageHeader valida que la extensión y el contenido real (magic bytes) correspondan a una imagen permitida.
func ValidateImageHeader(file *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return "", errors.New("solo se permiten imágenes JPG, JPEG, PNG o WEBP")
	}

	header, err := readHeaderBytes(file)
	if err != nil {
		return "", err
	}

	realMime := http.DetectContentType(header)

	switch {
	case strings.HasPrefix(realMime, "image/jpeg") && (ext == ".jpg" || ext == ".jpeg"):
		return "image/jpeg", nil
	case strings.HasPrefix(realMime, "image/png") && ext == ".png":
		return "image/png", nil
	case (strings.HasPrefix(realMime, "image/webp") || (len(header) >= 12 && string(header[0:4]) == "RIFF" && string(header[8:12]) == "WEBP")) && ext == ".webp":
		return "image/webp", nil
	default:
		return "", fmt.Errorf("el contenido real del archivo no corresponde a una imagen válida (%s)", realMime)
	}
}

// ExtractImage extrae y valida una sola imagen (JPG/JPEG/PNG/WEBP, máx 5MB).
func ExtractImage(c *gin.Context, fieldName string) (*multipart.FileHeader, int64, string, error) {
	if err := c.Request.ParseMultipartForm(MaxImageSize); err != nil {
		return nil, 0, "", fmt.Errorf("parse form: %w", err)
	}

	file, err := c.FormFile(fieldName)
	if err != nil {
		return nil, 0, "", fmt.Errorf("archivo '%s' requerido", fieldName)
	}

	if file.Size <= 0 {
		return nil, 0, "", errors.New("el archivo está vacío")
	}

	if file.Size > MaxImageSize {
		return nil, 0, "", fmt.Errorf("archivo muy grande (máx %dMB)", MaxImageSize/(1<<20))
	}

	mimeType, err := ValidateImageHeader(file)
	if err != nil {
		return nil, 0, "", err
	}

	return file, file.Size, mimeType, nil
}

// ExtractImages extrae y valida un lote de imágenes con límite de tamaño y cantidad.
func ExtractImages(c *gin.Context, fieldName string) ([]models.UploadedFile, error) {
	if err := c.Request.ParseMultipartForm(50 << 20); err != nil {
		return nil, fmt.Errorf("parse form: %w", err)
	}

	form, err := c.MultipartForm()
	if err != nil {
		return nil, fmt.Errorf("multipart form: %w", err)
	}

	files := form.File[fieldName]
	if len(files) == 0 {
		files = form.File[fieldName+"[]"]
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("se requiere al menos un archivo en '%s'", fieldName)
	}

	if len(files) > MaxBatchFiles {
		return nil, fmt.Errorf("número de archivos excede el límite permitido (máximo %d archivos)", MaxBatchFiles)
	}

	var result []models.UploadedFile
	for _, file := range files {
		if file == nil || file.Size <= 0 {
			continue
		}

		if file.Size > MaxImageSize {
			return nil, fmt.Errorf("archivo '%s' supera el tamaño máximo permitido (5MB)", filepath.Base(file.Filename))
		}

		mimeType, err := ValidateImageHeader(file)
		if err != nil {
			return nil, fmt.Errorf("archivo '%s': %w", filepath.Base(file.Filename), err)
		}

		result = append(result, models.UploadedFile{
			File:     file,
			Size:     file.Size,
			MimeType: mimeType,
		})
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no se encontraron imágenes válidas en '%s'", fieldName)
	}

	return result, nil
}

// ExtractExcel extrae y valida un archivo Excel (.xlsx/.xls, máx 10MB) comprobando sus magic bytes reales.
func ExtractExcel(c *gin.Context, fieldName string) (*multipart.FileHeader, int64, string, error) {
	if err := c.Request.ParseMultipartForm(MaxExcelSize); err != nil {
		return nil, 0, "", fmt.Errorf("parse form: %w", err)
	}

	file, err := c.FormFile(fieldName)
	if err != nil {
		return nil, 0, "", fmt.Errorf("archivo '%s' requerido", fieldName)
	}

	if file.Size <= 0 {
		return nil, 0, "", errors.New("el archivo está vacío")
	}

	if file.Size > MaxExcelSize {
		return nil, 0, "", fmt.Errorf("archivo muy grande (máx %dMB)", MaxExcelSize/(1<<20))
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".xlsx" && ext != ".xls" {
		return nil, 0, "", errors.New("solo archivos Excel (.xlsx/.xls) permitidos")
	}

	header, err := readHeaderBytes(file)
	if err != nil {
		return nil, 0, "", err
	}

	// .xlsx es un archivo ZIP empaquetado (PK\x03\x04)
	isZip := bytes.HasPrefix(header, []byte("PK\x03\x04")) || http.DetectContentType(header) == "application/zip"
	// .xls es formato OLE2 Compound Document (\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1)
	isOle2 := bytes.HasPrefix(header, []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1})

	if ext == ".xlsx" && !isZip {
		return nil, 0, "", errors.New("el archivo no corresponde a un libro de Excel (.xlsx) válido")
	}
	if ext == ".xls" && !isOle2 {
		return nil, 0, "", errors.New("el archivo no corresponde a un libro de Excel (.xls) binario válido")
	}

	return file, file.Size, file.Filename, nil
}

// ExtractPDF extrae y valida un archivo PDF (máx 10MB) comprobando sus magic bytes reales (%PDF-).
func ExtractPDF(c *gin.Context, fieldName string) (*multipart.FileHeader, int64, string, error) {
	if err := c.Request.ParseMultipartForm(MaxPDFSize); err != nil {
		return nil, 0, "", fmt.Errorf("parse form: %w", err)
	}

	file, err := c.FormFile(fieldName)
	if err != nil {
		return nil, 0, "", fmt.Errorf("archivo '%s' requerido", fieldName)
	}

	if file.Size <= 0 {
		return nil, 0, "", errors.New("el archivo está vacío")
	}

	if file.Size > MaxPDFSize {
		return nil, 0, "", fmt.Errorf("archivo muy grande (máx %dMB)", MaxPDFSize/(1<<20))
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" {
		return nil, 0, "", errors.New("solo se permiten archivos PDF")
	}

	header, err := readHeaderBytes(file)
	if err != nil {
		return nil, 0, "", err
	}

	// Comprobar magic bytes de PDF (%PDF-) y ContentType
	realMime := http.DetectContentType(header)
	if !bytes.HasPrefix(header, []byte("%PDF-")) && realMime != "application/pdf" {
		return nil, 0, "", errors.New("el archivo no corresponde a un documento PDF válido")
	}

	return file, file.Size, "application/pdf", nil
}
