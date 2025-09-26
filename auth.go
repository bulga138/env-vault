package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var oauthConfig *oauth2.Config

// init se ejecuta antes que main() y configura nuestro cliente OAuth
func init() {
	oauthConfig = &oauth2.Config{
		ClientID:     githubClientID,
		ClientSecret: githubClientSecret,
		RedirectURL:  "http://localhost:8080/github/callback",
		Scopes:       []string{"read:user"}, // Solo necesitamos permiso para leer el perfil del usuario
		Endpoint:     github.Endpoint,
	}
}

// handleLogin inicia el proceso de autenticación
func handleLogin() {
	// Canal para esperar a que la autenticación termine
	authComplete := make(chan string)

	// Iniciamos un servidor web local para recibir la respuesta de GitHub
	http.HandleFunc("/github/callback", func(w http.ResponseWriter, r *http.Request) {
		// GitHub nos devuelve un código de autorización
		code := r.URL.Query().Get("code")
		if code == "" {
			fmt.Fprintln(w, "Error: No se recibió el código de autorización.")
			authComplete <- ""
			return
		}

		// Intercambiamos el código por un token de acceso
		token, err := oauthConfig.Exchange(context.Background(), code)
		if err != nil {
			fmt.Fprintf(w, "Error al intercambiar el código por token: %v\n", err)
			authComplete <- ""
			return
		}

		if err := saveToken(token.AccessToken); err != nil {
        fmt.Fprintf(w, "Error crítico: No se pudo guardar el token de acceso de forma segura: %v\n", err)
		authComplete <- ""
        return
    }

		// (En una app real, guardarías este token de forma segura)
		// Por ahora, solo confirmamos que funcionó.
		fmt.Fprintln(w, "¡Autenticación exitosa! Puedes cerrar esta ventana.")
		authComplete <- token.AccessToken
	})

	// Ejecutamos el servidor en un hilo separado (goroutine)
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Printf("Error al iniciar el servidor local: %v\n", err)
		}
	}()

	// Generamos la URL a la que el usuario debe ir
	authURL := oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Println(" Abre esta URL en tu navegador para autorizar la aplicación:")
	fmt.Printf("\n%s\n\n", authURL)

	// Esperamos a que el callback nos envíe el token
	<-authComplete
	fmt.Println("✅ ¡Login completado con éxito!")
	// Damos un pequeño margen para que el usuario lea el mensaje
	time.Sleep(1 * time.Second)
}

// TODO: En un sistema real, aquí irían funciones para guardar y cargar
// el token de acceso de forma segura (p. ej., en el keychain del sistema)
// y comprobar si el usuario ya está logueado antes de ejecutar `save` o `get`.