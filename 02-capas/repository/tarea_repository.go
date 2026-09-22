package repository

import (
	"database/sql"

	"capas/models"
)

// TareaRepository encapsula TODO el SQL de tareas.
type TareaRepository struct {
	db *sql.DB
}

func NuevoTareaRepository(db *sql.DB) *TareaRepository {
	return &TareaRepository{db: db}
}

// Crear inserta una tarea nueva en estado pendiente y completa su ID.
func (r *TareaRepository) Crear(tarea *models.Tarea) error {
	resultado, err := r.db.Exec(
		"INSERT INTO tareas (titulo, descripcion, estado, usuario_id) VALUES (?, ?, ?, ?)",
		tarea.Titulo, tarea.Descripcion, tarea.Estado, tarea.UsuarioID,
	)
	if err != nil {
		return err
	}
	id, err := resultado.LastInsertId()
	if err != nil {
		return err
	}
	tarea.ID = id
	return nil
}

// ObtenerPorID devuelve una tarea o sql.ErrNoRows si no existe.
func (r *TareaRepository) ObtenerPorID(id int64) (models.Tarea, error) {
	var t models.Tarea
	err := r.db.QueryRow(
		"SELECT id, titulo, descripcion, estado, usuario_id, created_at FROM tareas WHERE id = ?", id,
	).Scan(&t.ID, &t.Titulo, &t.Descripcion, &t.Estado, &t.UsuarioID, &t.CreatedAt)
	return t, err
}

// FiltrosTarea agrupa los filtros opcionales del listado.
// Un struct evita que Listar acumule parámetros booleanos.
type FiltrosTarea struct {
	UsuarioID *int64
	Estado    *string
}

// Listar devuelve las tareas que cumplen los filtros (ninguno = todas).
func (r *TareaRepository) Listar(filtros FiltrosTarea) ([]models.Tarea, error) {
	consulta := "SELECT id, titulo, descripcion, estado, usuario_id, created_at FROM tareas WHERE 1=1"
	var args []any

	if filtros.UsuarioID != nil {
		consulta += " AND usuario_id = ?"
		args = append(args, *filtros.UsuarioID)
	}
	if filtros.Estado != nil {
		consulta += " AND estado = ?"
		args = append(args, *filtros.Estado)
	}
	consulta += " ORDER BY id"

	filas, err := r.db.Query(consulta, args...)
	if err != nil {
		return nil, err
	}
	defer filas.Close()

	tareas := []models.Tarea{}
	for filas.Next() {
		var t models.Tarea
		if err := filas.Scan(&t.ID, &t.Titulo, &t.Descripcion, &t.Estado, &t.UsuarioID, &t.CreatedAt); err != nil {
			return nil, err
		}
		tareas = append(tareas, t)
	}
	return tareas, filas.Err()
}

// ListarPorUsuario devuelve todas las tareas de un usuario.
func (r *TareaRepository) ListarPorUsuario(usuarioID int64) ([]models.Tarea, error) {
	return r.Listar(FiltrosTarea{UsuarioID: &usuarioID})
}

// CambiarEstado actualiza el estado de una tarea y devuelve si existía.
func (r *TareaRepository) CambiarEstado(id int64, estado string) (bool, error) {
	resultado, err := r.db.Exec("UPDATE tareas SET estado = ? WHERE id = ?", estado, id)
	if err != nil {
		return false, err
	}
	afectadas, err := resultado.RowsAffected()
	if err != nil {
		return false, err
	}
	return afectadas > 0, nil
}
