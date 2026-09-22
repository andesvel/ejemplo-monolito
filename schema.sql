-- Schema compartido por las 3 versiones de la demo.
-- Cada main.go lo ejecuta al arrancar sobre su propio archivo demo.db.

CREATE TABLE IF NOT EXISTS usuarios (
    id     INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre TEXT NOT NULL,
    email  TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS tareas (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    titulo      TEXT    NOT NULL,
    descripcion TEXT    NOT NULL DEFAULT '',
    estado      TEXT    NOT NULL DEFAULT 'pendiente'
                CHECK (estado IN ('pendiente', 'en_progreso', 'completada')),
    usuario_id  INTEGER NOT NULL REFERENCES usuarios(id),
    created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- Facilita el filtro GET /tareas?usuario_id=
CREATE INDEX IF NOT EXISTS idx_tareas_usuario_id ON tareas(usuario_id);
-- Facilita el filtro GET /tareas?estado=
CREATE INDEX IF NOT EXISTS idx_tareas_estado ON tareas(estado);
