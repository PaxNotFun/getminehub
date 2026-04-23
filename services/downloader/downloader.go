package downloader

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	httpclient "getminehub/services/http"
)

// ProgressCallback es llamado con el porcentaje de progreso (0-100).
type ProgressCallback func(percent float64)

// DownloadFileWithProgress descarga un archivo y reporta el progreso.
// Usa un Transport con ResponseHeaderTimeout para no cortar descargas grandes.
func DownloadFileWithProgress(url, destPath string, progress ProgressCallback) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("creando directorio de destino: %w", err)
	}

	req, err := httpclient.NewDownloadRequest(url)
	if err != nil {
		return err
	}

	resp, err := httpclient.DownloadTransport().Do(req)
	if err != nil {
		return fmt.Errorf("error de red descargando %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d (%s) al descargar %s", resp.StatusCode, resp.Status, url)
	}

	totalSize := resp.ContentLength

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creando archivo de destino %s: %w", destPath, err)
	}
	defer out.Close()

	buf := make([]byte, 32*1024)
	var downloaded int64

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("escribiendo en %s: %w", destPath, writeErr)
			}
			downloaded += int64(n)
			if totalSize > 0 && progress != nil {
				progress(float64(downloaded) / float64(totalSize) * 100)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("leyendo respuesta de descarga: %w", readErr)
		}
	}

	if progress != nil {
		progress(100)
	}
	return nil
}

// ExtractZip extrae un archivo ZIP de forma segura (previene path traversal).
func ExtractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("abriendo ZIP %s: %w", zipPath, err)
	}
	defer r.Close()

	cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)

	for _, f := range r.File {
		fpath := filepath.Join(destDir, filepath.Clean(f.Name))
		if !strings.HasPrefix(fpath, cleanDest) {
			continue // path traversal: ignorar
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return fmt.Errorf("creando directorio para %s: %w", fpath, err)
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("creando archivo %s: %w", fpath, err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("abriendo entrada ZIP %s: %w", f.Name, err)
		}

		_, copyErr := io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if copyErr != nil {
			return fmt.Errorf("extrayendo %s: %w", f.Name, copyErr)
		}
	}
	return nil
}

// ExtractTarGz extrae un archivo .tar.gz de forma segura (Linux/macOS).
func ExtractTarGz(tarPath, destDir string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("abriendo tar.gz %s: %w", tarPath, err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("iniciando descompresión gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	cleanDest := filepath.Clean(destDir)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("leyendo entrada tar: %w", err)
		}

		fpath := filepath.Join(destDir, filepath.Clean(header.Name))
		if !strings.HasPrefix(filepath.Clean(fpath), cleanDest) {
			continue // path traversal: ignorar
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(fpath, 0755); err != nil {
				return fmt.Errorf("creando directorio %s: %w", fpath, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
				return fmt.Errorf("creando directorio para %s: %w", fpath, err)
			}
			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("creando archivo %s: %w", fpath, err)
			}
			_, copyErr := io.Copy(outFile, tr)
			outFile.Close()
			if copyErr != nil {
				return fmt.Errorf("extrayendo %s: %w", header.Name, copyErr)
			}
		case tar.TypeSymlink:
			// Ignorar errores de symlinks (pueden fallar en algunos SO)
			_ = os.Symlink(header.Linkname, fpath)
		}
	}
	return nil
}

// ExtractArchive extrae un archivo detectando su formato automáticamente
// por la extensión (.tar.gz / .tgz → tar, resto → zip).
func ExtractArchive(archivePath, destDir string) error {
	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		return ExtractTarGz(archivePath, destDir)
	}
	return ExtractZip(archivePath, destDir)
}
