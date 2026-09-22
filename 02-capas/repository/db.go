// Package repository: ÚNICA capa que conoce SQL y el driver de base de datos.
package repository

import (
	"database/sql"

	_ "modernc.org/sqlite" // driver SQLite sin cgo, registrado una sola vez
)

// BENEFICIO: el driver y la conexión viven en este paquete. Si mañana
// cambiáramos de motor de base de datos, solo este archivo (y los archivos
// SQL de los repositorios) necesitarían cambios: ni handlers ni services
// se enteran.
func AbrirBD(ruta string) (*sql.DB, error) {
	return sql.Open("sqlite", ruta)
}
