# --- Variables de Configuración ---

# Nombre base del binario
BINARY_NAME=env-vault

# Directorio de salida para la compilación
BUILD_DIR=build

# Flags del compilador de Go para optimizar el binario de producción.
LDFLAGS=-ldflags="-s -w"

# Encuentra todos los archivos .go en el directorio actual
GO_FILES := $(wildcard *.go)

# --- Variables de Docker ---

# Nombre de la imagen de Docker
DOCKER_IMAGE_NAME=env-vault

# Tag de la imagen de Docker
DOCKER_IMAGE_TAG=latest

# --- Detección del Sistema Operativo ---

# Por defecto, asumimos un sistema tipo Unix (Linux/macOS)
BINARY_FULL_NAME=$(BUILD_DIR)/$(BINARY_NAME)

# Si el sistema operativo es Windows, ajustamos el nombre del binario
ifeq ($(OS), Windows_NT)
    BINARY_FULL_NAME := $(BUILD_DIR)/$(BINARY_NAME).exe
    MKDIR_P = if not exist $(subst /,\,$(BUILD_DIR)) mkdir $(subst /,\,$(BUILD_DIR))
else
    MKDIR_P = mkdir -p $(BUILD_DIR)
endif

# --- Reglas (Targets) ---

# El target por defecto que se ejecuta con solo 'make'
all: build

# El target 'build' ahora es solo un alias para el binario real.
build: $(BINARY_FULL_NAME)

# Esta es la regla principal de compilación.
# Solo se ejecuta si el binario no existe o si algún archivo .go es más nuevo.
$(BINARY_FULL_NAME): $(GO_FILES)
	@echo "Compilando para $(OS)..."
	@$(MKDIR_P)
	@go build $(LDFLAGS) -o $(BINARY_FULL_NAME) .
	@echo "Binario disponible en: $(BINARY_FULL_NAME)"

# Target para limpiar los artefactos de compilación local
clean:
	@echo "Limpiando artefactos de compilación..."
	@rm -rf $(BUILD_DIR)
	@echo "Directorio de trabajo limpio."

# Target para compilar y ejecutar la aplicación localmente
run: build
	@echo "Ejecutando la aplicación localmente..."
	@$(BINARY_FULL_NAME) $(ARGS)

# Target para gestionar las dependencias del proyecto
tidy:
	@echo "Ordenando módulos de Go..."
	@go mod tidy

# --- Atajos de Comandos ---
login:
	@make run ARGS=login

save:
	@make run ARGS=save

get:
	@make run ARGS=get

generate-pass:
	@make run ARGS=generate-pass

sync-save:
	@make run ARGS=sync-save

sync-load:
	@make run ARGS=sync-load

save-bw:
	@make run ARGS="--use-bitwarden save"

get-bw:
	@make run ARGS="--use-bitwarden get"

# Declara los targets que no corresponden a archivos para evitar conflictos
.PHONY: all clean run tidy login save get generate-pass sync-save sync-load save-bw get-bw