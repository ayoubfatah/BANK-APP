package api

import (
	"errors"
	"net/http"
	"time"

	db "github.com/bank-app/db/sqlc"

	"github.com/gin-gonic/gin"
)

type createUserRequest struct {
	FirstName    string `json:"first_name" binding:"required"`
	LastName     string `json:"last_name" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Img          string `json:"img"`
	Phone        string `json:"phone"`
	PasswordHash string `json:"password_hash" binding:"required"`
	Status       string `json:"status" binding:"required"`
}

type updateUserRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Img       string `json:"img"`
}

type updateEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type updatePasswordRequest struct {
	PasswordHash string `json:"password_hash" binding:"required"`
}

type updatePhoneRequest struct {
	Phone string `json:"phone"`
}

type updateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type userResponse struct {
	ID           int64      `json:"id"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Email        string     `json:"email"`
	Role         string     `json:"role"`
	Img          *string    `json:"img"`
	Phone        *string    `json:"phone"`
	PasswordHash string     `json:"password_hash"`
	Status       string     `json:"status"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

func newUserResponse(user db.User) userResponse {
	return userResponse{
		ID:           user.ID,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Email:        user.Email,
		Role:         user.Role,
		Img:          ptrText(user.Img),
		Phone:        ptrText(user.Phone),
		PasswordHash: user.PasswordHash,
		Status:       user.Status,
		CreatedAt:    ptrTime(user.CreatedAt),
		UpdatedAt:    ptrTime(user.UpdatedAt),
	}
}

func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.store.CreateUser(ctx, db.CreateUserParams{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		Img:          textOrNull(req.Img),
		Phone:        textOrNull(req.Phone),
		PasswordHash: req.PasswordHash,
		Status:       req.Status,
	})

	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newUserResponse(user))
}

func (server *Server) getUser(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}

	user, err := server.store.GetUser(ctx, id)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newUserResponse(user))
}

func (server *Server) getUserByEmail(ctx *gin.Context) {
	email := ctx.Param("email")

	if email == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "email is required",
		})
		return
	}

	user, err := server.store.GetUserByEmail(ctx, email)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newUserResponse(user))
}

func (server *Server) getAllUsers(ctx *gin.Context) {
	limit := int32(10)
	offset := int32(0)

	users, err := server.store.GetAllUsers(ctx, db.GetAllUsersParams{
		Limit:  limit,
		Offset: offset,
	})

	if err != nil {
		handleDBError(ctx, err)
		return
	}

	response := make([]userResponse, 0, len(users))

	for _, user := range users {
		response = append(response, newUserResponse(user))
	}

	ctx.JSON(http.StatusOK, response)
}

func (server *Server) updateUser(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}

	var req updateUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.store.UpdateUser(ctx, db.UpdateUserParams{
		ID:        id,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Img:       textOrNull(req.Img),
	})

	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newUserResponse(user))
}

func (server *Server) updateUserEmail(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}

	var req updateEmailRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.store.UpdateUserEmail(ctx, db.UpdateUserEmailParams{
		ID:    id,
		Email: req.Email,
	})

	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newUserResponse(user))
}

func (server *Server) updateUserPassword(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}

	var req updatePasswordRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := server.store.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           id,
		PasswordHash: req.PasswordHash,
	})

	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (server *Server) updateUserPhone(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}

	var req updatePhoneRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := server.store.UpdateUserPhone(ctx, db.UpdateUserPhoneParams{
		ID:    id,
		Phone: textOrNull(req.Phone),
	})

	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (server *Server) updateUserStatus(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}

	var req updateStatusRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.store.UpdateUserStatus(ctx, db.UpdateUserStatusParams{
		ID:     id,
		Status: req.Status,
	})

	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newUserResponse(user))
}

func (server *Server) closeUserAccount(ctx *gin.Context) {
	// id, ok := parseID(ctx)
	// if !ok {
	// 	return
	// }
// TODO change to real id later 
	id := int64(1)

  	closedAcc,	err := server.store.CloseUserAccount(ctx, id)
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if closedAcc.Status != "closed" {
		ctx.JSON(http.StatusInternalServerError,errorResponse(errors.New("failed to close user account")))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":"Account closed successfully",
	})
}





