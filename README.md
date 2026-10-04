# UFC TUI

TUI en Go para consultar la cartelera de UFC desde
[ufcstats.com](http://ufcstats.com). Sin cuenta, sin base de datos, sin clave
de API: el scraper escribe un JSON y la app solo lo lee.

## Teclas

| Tecla | Acción |
|---|---|
| `1` … `5` | Ir a la vista. |
| `←` `→` | Cambiar de vista. También `h` / `l` y `tab` / `shift+tab`. |
| `↑` `↓` | Mover el cursor en cartelera y resultados. También `k` / `j`. |
| `q` | Salir. También `ctrl+c` y `esc`. |

### Las cinco vistas

- **próximo** — el evento más cercano: fecha, lugar y la pelea estelar en un
  panel propio.
- **hoy** — si hay pelea esta noche, o cuándo es la próxima.
- **calendario** — los eventos agrupados por mes, con cuántas peleas tienen.
- **cartelera** — las peleas del próximo evento, numeradas, con la estelar
  marcada. Scrollea con `↑`/`↓`.
- **resultados** — hasta 20 eventos ya peleados, del más nuevo al más viejo,
  con un resumen de su estelar (método, ronda, tiempo).

## Tests

```bash
go test ./...
```

Las pruebas no requieren red: el scraper se prueba contra HTML guardado en
`internal/fetch/testdata/`, incluido el marcado real de un evento con la
cartelera sin anunciar. `go test ./... -v` lista cada caso.

```bash
gofmt -l .
go vet ./...
```

## Cómo funciona

```
ufcstats.com
     │  HTTP + challenge proof-of-work
     ▼
internal/fetch ──► data/events.json
     │                 │
     │                 ▼
     └──────────► internal/dataset ──► internal/ui
```

| Paquete | Responsabilidad |
|---|---|
| `cmd/ufc` | Entry point de la TUI. Decide si sincronizar y arranca Bubble Tea. |
| `cmd/ufc-sync` | Entry point del sync. |
| `internal/fetch` | Cliente HTTP, challenge PoW, parsing de HTML, dedupe y escritura del JSON. |
| `internal/dataset` | Consultas sobre el JSON ya leído: próximas, cartelera, resultados. |
| `internal/ui` | Modelo Bubble Tea, vistas y estilos. |
| `internal/model` | `Fight`, `Event`, `Dataset` y el parseo de fechas. |

Dependencias: `bubbletea`, `lipgloss` y `goquery`.

## Instalar con Go

Requiere Go 1.27 o posterior.

```sh
go install github.com/offluisangel/UFC/cmd/ufc@latest
ufc tui
```

Go instala el ejecutable en `GOBIN` o, si no está configurado, en
`GOPATH/bin`. Esa carpeta debe estar en el `PATH`; en Windows, el ejecutable se
llama `ufc.exe`. Para actualizarlo, vuelve a ejecutar `go install`.
La caché se guarda en `data/events.json`, relativa a la carpeta desde donde
lanzas el comando.

## Ejecutar en Windows

Descarga y extrae el ZIP; puedes abrir `ufc.exe` directamente o, desde una terminal en esa carpeta, ejecutar:

```powershell
.\ufc.exe tui
```
