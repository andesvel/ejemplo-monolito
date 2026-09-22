package repository

import (
	"database/sql"

	"capas/models"
)

// UsuarioRepository encapsula TODO el SQL de usuarios.
// Es una estructura con la conexión inyectada, no funciones sueltas:
// eso permite que el service la reciba ya construida (y en un test, una
// versión falsa con la misma forma).
type UsuarioRepository struct {
	db *sql.DB
}

func NuevoUsuarioRepository(db *sql.DB) *UsuarioRepository {
	return &UsuarioRepository{db: db}
}

// Crear inserta un usuario y completa su ID generado.
func (r *UsuarioRepository) Crear(usuario *models.Usuario) error {
	resultado, err := r.db.Exec(
		"INSERT INTO usuarios (nombre, email) VALUES (?, ?)",
		usuario.Nombre, usuario.Email,
	)
	if err != nil {
		return err
	}
	id, err := resultado.LastInsertId()
	if err != nil {
		return err
	}
	usuario.ID = id
	return nil
}

// Listar devuelve todos los usuarios ordenados por id.
func (r *UsuarioRepository) Listar() ([]models.Usuario, error) {
	filas, err := r.db.Query("SELECT id, nombre, email FROM usuarios ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer filas.Close()

	usuarios := []models.Usuario{}
	for filas.Next() {
		var u models.Usuario
		if err := filas.Scan(&u.ID, &u.Nombre, &u.Email); err != nil {
			return nil, err
		}
		usuarios = append(usuarios, u)
	}
	return usuarios, filas.Err()
}

// Existe reporta si hay un usuario con ese id.
func (r *UsuarioRepository) Existe(id int64) (bool, error) {
	var n int
	if err := r.db.QueryRow("SELECT COUNT(1) FROM usuarios WHERE id = ?", id).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}
