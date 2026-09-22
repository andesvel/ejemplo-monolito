package models

// Usuario es la representación de un usuario en la API (JSON) y en memoria.
// Aquí solo viven datos: ninguna regla de negocio ni SQL.
type Usuario struct {
	ID     int64  `json:"id"`
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
}
