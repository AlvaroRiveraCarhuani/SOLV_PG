package services

import (
	"database/sql"
	"errors"

	"solv-backend/internal/core/domain"
)

// isMissingRow indica que una lectura terminó sin fila. Es un resultado normal
// del dominio —la práctica no existe, el laboratorio ya se borró— y no un fallo
// del servidor: los llamadores lo traducen a domain.ErrNotFound o lo cuentan como
// omitido, y solo se registra en el log cuando la lectura falló de verdad.
//
// Se incluye err == nil porque un repositorio puede devolver (nil, nil) cuando no
// encuentra el registro, y ese caso tampoco es un error.
func isMissingRow(err error) bool {
	return err == nil || errors.Is(err, sql.ErrNoRows) || errors.Is(err, domain.ErrNotFound)
}
