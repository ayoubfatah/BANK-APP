package api

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

func handleDBError(ctx *gin.Context, err error) {
	if err == nil {
		return
	}

	if errors.Is(err, pgx.ErrNoRows) {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "not found",
		})
		return
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
			
		case "23502":
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "required field is missing",
			})
			return

		case "23503":
			ctx.JSON(http.StatusConflict, gin.H{
				"error": "record is referenced by other data",
			})
			return

		case "23505":
			msg := "already exists"

			switch {
			case strings.Contains(pgErr.ConstraintName, "email"):
				msg = "email is already in use"
			case strings.Contains(pgErr.ConstraintName, "phone"):
				msg = "phone number is already in use"
			case strings.Contains(pgErr.ConstraintName, "username"):
				msg = "username is already in use"
			case strings.Contains(pgErr.ConstraintName, "idempotency"):
				msg = "request has already been processed"
			}

			ctx.JSON(http.StatusConflict, gin.H{
				"error": msg,
			})
			return

		case "23514":
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid value",
			})
			return

		case "22001":
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "value is too long",
			})
			return

		case "22003":
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "value is out of range",
			})
			return

		case "22P02":
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid input value",
			})
			return

		case "40001", "40P01":
			ctx.JSON(http.StatusConflict, gin.H{
				"error": "transaction could not be completed, please try again",
			})
			return
		}
	}

	log.Printf("internal database error: %v", err)

	ctx.JSON(http.StatusInternalServerError, gin.H{
		"error": "internal server error",
	})
}