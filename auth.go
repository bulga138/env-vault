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

func initOAuth(clientID, clientSecret string) {
    oauthConfig = &oauth2.Config{
        ClientID:     clientID,
        ClientSecret: clientSecret,
        RedirectURL:  "http://localhost:8080/github/callback",
        Scopes:       []string{"read:user"},
        Endpoint:     github.Endpoint,
    }
}

func handleLogin() {
	authComplete := make(chan string)

	http.HandleFunc("/github/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			fmt.Fprintln(w, "Error: No se recibió el código de autorización.")
			authComplete <- ""
			return
		}

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

	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Printf("Error al iniciar el servidor local: %v\n", err)
		}
	}()

	authURL := oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Println(" Abre esta URL en tu navegador para autorizar la aplicación:")
	fmt.Printf("\n%s\n\n", authURL)

	<-authComplete
	fmt.Println("✅ ¡Login completado con éxito!")
	time.Sleep(1 * time.Second)
}

// TODO: En un sistema real, aquí irían funciones para guardar y cargar
// el token de acceso de forma segura (p. ej., en el keychain del sistema)
// y comprobar si el usuario ya está logueado antes de ejecutar `save` o `get`.
