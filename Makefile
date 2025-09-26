# --- Variables de Configuración ---

# Nombre base del binario
BINARY_NAME=env-vault

# Directorio de salida para la compilación
BUILD_DIR=build

# Flags del compilador de Go para optimizar el binario de producción.
LDFLAGS=-ldflags="-s -w"

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
    BINARY_FULL_NAME=$(BUILD_DIR)/$(BINARY_NAME).exe
endif

# --- Reglas (Targets) ---

# El target por defecto que se ejecuta con solo 'make'
all: build

# Target para compilar la aplicación localmente
build:
	@echo "🔨 Compilando para $(OS)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BINARY_FULL_NAME) .
	@echo "✅ ¡Compilación exitosa! Binario disponible en: $(BINARY_FULL_NAME)"

# Target para limpiar los artefactos de compilación local
clean:
	@echo "🧹 Limpiando artefactos de compilación..."
	@rm -rf $(BUILD_DIR)
	@echo "✨ Directorio de trabajo limpio."

# Target para compilar y ejecutar la aplicación localmente
run: build
	@echo "🚀 Ejecutando la aplicación localmente..."
	@$(BINARY_FULL_NAME) $(ARGS)

# Target para gestionar las dependencias del proyecto
tidy:
	@echo "📦 Ordenando módulos de Go..."
	@go mod tidy

# --- Reglas de Docker ---

# Target para construir la imagen de Docker
docker-build:
	@echo "🐳 Construyendo la imagen de Docker..."
	@docker build -t $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) .
	@echo "✅ Imagen '$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)' construida con éxito."

# Target para ejecutar la aplicación dentro de un contenedor Docker
# Monta el directorio actual para que la app pueda acceder a .env y vault.json
docker-run:
	@echo "🚀 Ejecutando la aplicación en Docker..."
	@docker run -it --rm -v "$(shell pwd):/app" -w /app $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) $(ARGS)

# Target para limpiar la imagen de Docker
docker-clean:
	@echo "🧹 Eliminando la imagen de Docker..."
	@docker rmi $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) || true
	@echo "✨ Imagen eliminada."


# Declara los targets que no corresponden a archivos para evitar conflictos
.PHONY: all build clean run tidy docker-build docker-run docker-clean