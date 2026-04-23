# Contribuir a GetMineHub

¡Gracias por tu interés en contribuir! Este documento explica cómo puedes ayudar al proyecto.

## Antes de empezar

- Revisa los [Issues abiertos](https://github.com/PaxNotFun/getminehub/issues) para ver si alguien ya está trabajando en lo mismo.
- Para cambios grandes, abre primero un Issue para discutir la idea antes de escribir código.
- Para correcciones pequeñas (typos, bugs obvios) puedes ir directo con un Pull Request.

## Requisitos para compilar

- [Go 1.22+](https://go.dev/dl/)
- [Wails v2](https://wails.io/docs/gettingstarted/installation)

```bash
# Instalar Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Clonar el repositorio
git clone https://github.com/PaxNotFun/getminehub.git
cd getminehub

# Compilar en modo desarrollo
wails dev

# Compilar para producción
wails build
```

## Cómo contribuir

### Reportar un bug
Usa la plantilla de [reporte de bug](https://github.com/PaxNotFun/getminehub/issues/new?template=bug_report.md). Incluye siempre el sistema operativo, versión de GetMineHub y los logs de la consola.

### Sugerir una función
Usa la plantilla de [solicitud de función](https://github.com/PaxNotFun/getminehub/issues/new?template=feature_request.md).

### Enviar código

1. Haz un **fork** del repositorio
2. Crea una rama con un nombre descriptivo:
   ```bash
   git checkout -b fix/error-instalacion-forge
   git checkout -b feature/soporte-neoforge
   ```
3. Haz tus cambios y commits con mensajes claros en español o inglés
4. Abre un **Pull Request** hacia la rama `main`
5. Describe qué cambiaste y por qué en la descripción del PR

## Estilo de código

- El código backend está en **Go** — sigue las convenciones estándar de Go (`gofmt`)
- El frontend usa **JavaScript/CSS** vanilla — mantén el estilo existente
- Los comentarios en el código están en **español**, mantenlo así

## ¿Dudas?

Abre una discusión en [GitHub Discussions](https://github.com/PaxNotFun/getminehub/discussions).
