# lazygit-ai

Fork de [Lazygit](https://github.com/jesseduffield/lazygit) con generación de mensajes de commit mediante modelos locales de **Ollama**. Conserva la interfaz de terminal para gestionar archivos, ramas, commits, stashes y rebases.

La IA analiza los cambios preparados para commit (*staged*) y propone un título y una descripción al abrir el formulario de commit. Puedes revisarlos y editarlos antes de confirmar; la generación no crea el commit por sí sola.

## Instalación rápida (sin Go)

Esta es la opción recomendada: **no necesitas Go ni clonar el repositorio**. Necesitas Git para usar la aplicación. El instalador descarga el binario de la última release, detecta tu sistema y arquitectura y verifica su SHA256 antes de instalarlo. Ollama y los modelos se configuran por separado, como se explica más abajo.

### macOS y Linux con curl

```sh
curl -fsSL https://raw.githubusercontent.com/javiermoya2504/lazygit-ai/lazygitai/scripts/install.sh | sh
```

Instala en `~/.local/bin`, sin `sudo`. Si esa carpeta no está en tu `PATH`, agrega esta línea a `~/.zshrc` o `~/.bashrc` y abre otra terminal:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

### Windows con PowerShell

```powershell
irm https://raw.githubusercontent.com/javiermoya2504/lazygit-ai/lazygitai/scripts/install.ps1 | iex
```

Instala en `%LOCALAPPDATA%\Programs\lazygit-ai` y agrega esa carpeta al `Path` de tu usuario. Abre otra terminal si hace falta.

### Comprobar y abrir

```sh
lazygit-ai --version
```

Después, abre una terminal dentro de cualquier repositorio Git y ejecuta:

```sh
lazygit-ai
```

Para generar mensajes con IA, continúa con [Instalar y preparar Ollama](#instalar-y-preparar-ollama) y [Configurar lazygit-ai](#configurar-lazygit-ai).

### Actualizar

Cierra `lazygit-ai` y vuelve a ejecutar el comando de instalación de tu sistema. Descarga la última release y reemplaza el ejecutable; conserva tu configuración y los modelos de Ollama.

### Opciones del instalador

Puedes descargar e inspeccionar los scripts antes de ejecutarlos: [Unix](scripts/install.sh) y [Windows](scripts/install.ps1).

Variables opcionales: `LAZYGIT_AI_VERSION` fija una versión (por ejemplo `v1.0.1`) y `LAZYGIT_AI_INSTALL_DIR` cambia el destino. En Unix:

```sh
curl -fsSL https://raw.githubusercontent.com/javiermoya2504/lazygit-ai/lazygitai/scripts/install.sh | LAZYGIT_AI_VERSION=v1.0.1 sh
```

También puedes descargar los archivos de [Releases](https://github.com/javiermoya2504/lazygit-ai/releases). Si falla la descarga, comprueba tu conexión y que exista un archivo para tu sistema y arquitectura en la versión elegida.

## Instalar y preparar Ollama

- **macOS:** descarga [Ollama para macOS](https://ollama.com/download/mac), instálalo y abre la aplicación.
- **Windows:** instala [Ollama para Windows](https://ollama.com/download/windows). El instalador permite usar `ollama` desde PowerShell y la aplicación ejecuta el servicio en segundo plano.
- **Linux:** sigue la [guía oficial de Linux](https://docs.ollama.com/linux), cuyo instalador es:

  ```sh
  curl -fsSL https://ollama.com/install.sh | sh
  ```

Comprueba que el servicio responde y descarga el modelo configurado por defecto:

```sh
ollama list
ollama pull qwen2.5-coder:7b
```

Si no hay un servidor en ejecución, inicia `ollama serve` en otra terminal y déjalo abierto. Si Ollama ya corre como aplicación o servicio, no necesitas iniciar otra instancia. La [guía de inicio de Ollama](https://docs.ollama.com/quickstart) explica el uso básico.

### Elegir un modelo

Puedes usar otros modelos de generación de texto que funcionen con `/api/generate`. Estos tamaños de [Qwen2.5 Coder](https://ollama.com/library/qwen2.5-coder) son ejemplos:

| Modelo | Uso orientativo |
| --- | --- |
| `qwen2.5-coder:1.5b` | Probar con un modelo más pequeño. |
| `qwen2.5-coder:3b` | Alternativa de tamaño intermedio. |
| `qwen2.5-coder:7b` | Modelo predeterminado del fork. |

El rendimiento depende del hardware, la memoria disponible y el tamaño del diff. Para cambiar de modelo, descárgalo y copia su nombre exacto al campo `ai.model`:

```sh
ollama pull qwen2.5-coder:3b
ollama list
```

## Configurar lazygit-ai

Encuentra la carpeta de configuración efectiva:

```sh
lazygit-ai --print-config-dir
```

Abre o crea `config.yml` dentro de esa carpeta. El fork conserva las rutas de configuración de Lazygit por compatibilidad, por lo que puede compartir configuración con una instalación existente. Para separar la configuración, usa `lazygit-ai --use-config-dir /ruta/a/lazygit-ai` cada vez que lo inicies; en Windows puedes usar una ruta como `"$env:LOCALAPPDATA\lazygit-ai-config"`.

Agrega este bloque a `config.yml` (si ya existe `ai:`, edítalo en lugar de duplicarlo):

```yaml
ai:
  enabled: true
  provider: ollama
  model: qwen2.5-coder:7b
  endpoint: http://localhost:11434
  maxDiffLines: 300
  autoGenerateCommitMessage: true
  timeoutSeconds: 60
```

Reinicia `lazygit-ai` después de guardar.

| Opción | Descripción |
| --- | --- |
| `enabled` | Activa la IA; viene desactivada por defecto. |
| `provider` | Actualmente solo admite `ollama`. |
| `model` | Nombre exacto del modelo descargado, incluida su etiqueta. |
| `endpoint` | URL raíz del servidor local. Admite `localhost` e IP de loopback como `127.0.0.1` o `::1`; no admite servidores en otra máquina. |
| `maxDiffLines` | Máximo de líneas del diff enviadas al modelo. `0` desactiva el recorte. |
| `autoGenerateCommitMessage` | Genera la propuesta al abrir el formulario de commit. Debe estar activado junto con `enabled`. |
| `timeoutSeconds` | Tiempo máximo de espera. El valor predeterminado es 10 segundos; el ejemplo lo aumenta a 60 para permitir la carga inicial del modelo. |

El diff preparado se envía al servidor Ollama local configurado. Para mantener la inferencia en tu equipo, usa modelos descargados locales como los ejemplos anteriores.

## Generar un mensaje de commit

1. Abre una terminal en un repositorio Git y ejecuta `lazygit-ai`.
2. En el panel de archivos, prepara los cambios con `Espacio`, o ejecuta antes `git add` desde tu terminal.
3. Presiona `c` con los atajos predeterminados para abrir el formulario de commit.
4. Espera la propuesta de título y descripción. El prompt solicita un mensaje con formato Conventional Commits; no hay una opción específica para elegir el idioma.
5. Revisa el resultado, edítalo si hace falta y confirma desde el formulario.

La generación usa únicamente el diff staged. Si comienzas a editar mientras llega la respuesta, no sustituye tu texto. También respeta un borrador conservado y omite la generación en el flujo de commit que salta hooks. Si Ollama falla, puedes escribir el mensaje manualmente.

## Solución de problemas

| Problema | Qué revisar |
| --- | --- |
| `lazygit-ai` no se encuentra | Comprueba el `PATH` y abre otra terminal. En macOS/Linux puedes probar `~/.local/bin/lazygit-ai`; en PowerShell, `& "$env:LOCALAPPDATA\Programs\lazygit-ai\lazygit-ai.exe"` si usaste el destino predeterminado. |
| No aparece una propuesta | Verifica `enabled`, `autoGenerateCommitMessage`, que haya cambios staged y que el formulario no tenga un borrador conservado. |
| `AI commit message generation failed` | Comprueba `ollama list`, el modelo descargado y el endpoint; revisa los logs para conocer el error concreto. |
| Modelo no encontrado | Ejecuta `ollama pull` con el mismo nombre y etiqueta de `ai.model`. |
| Tiempo de espera agotado | Prueba `ollama run qwen2.5-coder:7b` para cargar el modelo, aumenta `timeoutSeconds` o elige uno más pequeño. |
| Endpoint inválido | Usa `http://localhost:11434` o una IP de loopback. Una IP de red local no está admitida. |
| Resumen incompleto | Revisa si el diff supera `maxDiffLines`; aumenta el límite o prepara commits más pequeños. |

Para diagnosticar, inicia `lazygit-ai --debug` y, desde otra terminal, ejecuta `lazygit-ai --logs`. Si usas `--use-config-dir`, indica la misma carpeta en ambos comandos.

## Instalación desde código fuente

Necesitas **Git** y **Go 1.25 o posterior** en el `PATH`. Puedes obtenerlos desde [Git](https://git-scm.com/downloads) y [Go](https://go.dev/dl/). Comprueba `git --version` y `go version` después de instalarlos; si alguna dependencia solicita una versión más reciente de Go, permite su descarga automática o actualiza Go.

Las instrucciones compilan este fork desde código fuente. Los paquetes habituales llamados `lazygit` instalan el proyecto original y no incluyen necesariamente estas funciones de IA.

El repositorio y el ejecutable se llaman `lazygit-ai`. La rama predeterminada de este fork, `lazygitai`, incluye la integración con Ollama.

### macOS

Instala Git y Go con los instaladores anteriores. Después, en Terminal:

```sh
git clone https://github.com/javiermoya2504/lazygit-ai.git lazygit-ai
cd lazygit-ai
go build -o lazygit-ai .
mkdir -p "$HOME/.local/bin"
install -m 755 lazygit-ai "$HOME/.local/bin/lazygit-ai"
export PATH="$HOME/.local/bin:$PATH"
lazygit-ai --version
```

Para conservar el `PATH` al abrir otra terminal, agrega `export PATH="$HOME/.local/bin:$PATH"` a `~/.zshrc` (o al archivo de inicio de tu shell).

### Linux

Instala Git con el gestor de paquetes de tu distribución y Go desde su instalador oficial si la versión disponible es anterior a la requerida. Después:

```sh
git clone https://github.com/javiermoya2504/lazygit-ai.git lazygit-ai
cd lazygit-ai
go build -o lazygit-ai .
mkdir -p "$HOME/.local/bin"
install -m 755 lazygit-ai "$HOME/.local/bin/lazygit-ai"
export PATH="$HOME/.local/bin:$PATH"
lazygit-ai --version
```

Agrega `export PATH="$HOME/.local/bin:$PATH"` a `~/.bashrc` o `~/.zshrc`, según tu shell, para conservarlo.

### Windows (PowerShell)

Instala Git y Go con sus instaladores oficiales y abre una nueva ventana de PowerShell:

```powershell
git clone https://github.com/javiermoya2504/lazygit-ai.git lazygit-ai
Set-Location lazygit-ai
go build -o lazygit-ai.exe .
if ($LASTEXITCODE -ne 0) { throw "Falló la compilación" }
$binDir = Join-Path $env:LOCALAPPDATA 'Programs\lazygit-ai'
New-Item -ItemType Directory -Force -Path $binDir | Out-Null
Copy-Item .\lazygit-ai.exe $binDir -Force
$env:Path = "$binDir;$env:Path"
lazygit-ai --version
```

Para usarlo en futuras sesiones, agrega `%LOCALAPPDATA%\Programs\lazygit-ai` al `Path` de tu usuario desde **Editar las variables de entorno de esta cuenta**, y abre otra terminal.

### Si ya tienes el fork clonado

Desde su carpeta, compila con `go build -o lazygit-ai .` (Windows: `go build -o lazygit-ai.exe .`) y copia el resultado a la ubicación de instalación indicada arriba.

Con GNU Make también puedes ejecutar `make build`, `make run` o `make install`. Este último instala `lazygit-ai` en el directorio `bin` de `go env GOPATH`; puedes elegir otro con `make install INSTALL_DIR="$HOME/.local/bin"`.

El módulo Go conserva la ruta interna de upstream para facilitar su mantenimiento. Usa los comandos de compilación anteriores para obtener el ejecutable con el nombre del fork.

## Desarrollo y publicación

Para actualizar una instalación compilada desde código fuente, ejecuta `git pull --ff-only`, vuelve a compilar y copia el nuevo binario a tu directorio de instalación.

```sh
go test ./pkg/ai ./pkg/config ./pkg/gui/controllers/helpers -short
```

La configuración de GoReleaser produce binarios llamados `lazygit-ai` (`lazygit-ai.exe` en Windows). Al subir un tag `vX.Y.Z`, GitHub Actions ejecuta las comprobaciones y publica los binarios y `checksums.txt` en este fork. La primera distribución del fork es `v1.0.1`; su numeración es independiente de upstream.

Si el evento del tag no inicia el workflow, también puedes publicar un tag existente desde **Actions → Release → Run workflow**, o con:

```sh
gh workflow run release.yml -R javiermoya2504/lazygit-ai --ref lazygitai -f tag=v1.0.1
```

Consulta la [documentación de configuración](docs/Config.md), los [atajos](docs/keybindings/Keybindings_en.md) y la [guía de desarrollo](docs/dev/README.md).

## Créditos y licencia

Basado en [Lazygit de Jesse Duffield y sus colaboradores](https://github.com/jesseduffield/lazygit). Este fork añade la integración de IA local y su propia identidad de distribución. Se conserva la [licencia MIT](LICENSE).
