// 03-modular: organización vertical por dominios (features).
//
// main.go es deliberadamente mínimo: prepara la base de datos y delega en
// cada feature el registro de sus propias rutas. Agregar un dominio nuevo
// = crear una carpeta nueva en features/ y una línea aquí.
package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"monolito/features/tareas"
	"monolito/features/usuarios"

	_ "modernc.org/sqlite" // driver SQLite sin cgo
)

func main() {
	db, err := sql.Open("sqlite", "demo.db")
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

	mux := http.NewServeMux()
	usuarios.RegistrarRutas(mux, db)
	tareas.RegistrarRutas(mux, db)

	log.Println("03-modular escuchando en http://localhost:8083")
	log.Fatal(http.ListenAndServe(":8083", mux))
}
