// 02-capas: arquitectura en capas horizontales.
//
// main.go es el único archivo que conoce a todas las capas y las conecta:
// base de datos -> repositories -> services -> handlers -> rutas.
// En la demo en vivo, seguir estas 5 líneas es el "mapa" del programa.
package main

import (
	"log"
	"net/http"
	"os"

	"capas/handlers"
	"capas/repository"
	"capas/services"
)

func main() {
	// Capa de datos: una conexión, compartida por todos los repositorios.
	db, err := repository.AbrirBD("demo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	schema, err := os.ReadFile("../schema.sql")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		log.Fatal(err)
	}

	// Capa de datos: repositorios (todo el SQL vive en repository/).
	repoUsuarios := repository.NuevoUsuarioRepository(db)
	repoTareas := repository.NuevoTareaRepository(db)

	// Capa de negocio: services (reglas de negocio en services/).
	servUsuarios := services.NuevoUsuarioService(repoUsuarios)
	servTareas := services.NuevoTareaService(repoTareas, repoUsuarios)

	// Capa HTTP: handlers (parsear request, llamar service, responder).
	hUsuarios := handlers.NuevoUsuarioHandler(servUsuarios)
	hTareas := handlers.NuevoTareaHandler(servTareas)

	mux := http.NewServeMux()
	hUsuarios.Registrar(mux)
	hTareas.Registrar(mux)

	log.Println("02-capas escuchando en http://localhost:8082")
	log.Fatal(http.ListenAndServe(":8082", mux))
}
