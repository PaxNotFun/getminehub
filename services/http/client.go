// Package httpclient provee un cliente HTTP centralizado y reutilizable
// para todas las peticiones de red de GetMineHub.
package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"getminehub/config"
)

const userAgent = "GetMineHub"

// client construye un *http.Client con el timeout configurado por el usuario.
// Se crea uno nuevo por llamada para respetar cambios en settings en caliente.
func client() *http.Client {
	timeout := config.LoadAllSettings().Timeout
	if timeout <= 0 {
		timeout = config.DefaultTimeout
	}
	return &http.Client{Timeout: time.Duration(timeout) * time.Second}
}

// Get realiza un GET con User-Agent y devuelve el cuerpo completo.
// Usa context.Background(); para cancelación explícita usa GetWithContext.
func Get(url string) ([]byte, error) {
	return GetWithContext(context.Background(), url)
}

// GetWithContext realiza un GET cancelable. El caller puede pasar el ctx de
// Wails (app.ctx) para que las peticiones se anulen cuando la ventana se cierra.
func GetWithContext(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("construyendo request para %s: %w", url, err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("error de red al obtener %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d (%s) al obtener %s", resp.StatusCode, resp.Status, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("leyendo respuesta de %s: %w", url, err)
	}
	return body, nil
}

// DownloadTransport construye un *http.Client optimizado para descargas largas,
// usando ResponseHeaderTimeout en lugar de un Timeout global para no cortar
// transferencias legítimas de archivos grandes.
func DownloadTransport() *http.Client {
	timeout := config.LoadAllSettings().Timeout
	if timeout <= 0 {
		timeout = config.DefaultTimeout
	}
	return &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: time.Duration(timeout) * time.Second,
		},
	}
}

// NewDownloadRequest crea un GET cancelable con User-Agent listo para
// pasarle a DownloadTransport(). Usar context.Background() si no se necesita
// cancelación explícita.
func NewDownloadRequest(url string) (*http.Request, error) {
	return NewDownloadRequestWithContext(context.Background(), url)
}

// NewDownloadRequestWithContext crea un GET cancelable con User-Agent.
// Pasar el ctx de Wails permite abortar descargas cuando la app se cierra.
func NewDownloadRequestWithContext(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("construyendo request de descarga para %s: %w", url, err)
	}
	req.Header.Set("User-Agent", userAgent)
	return req, nil
}
