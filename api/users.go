package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/utils"

	"github.com/gin-gonic/gin"
)

// TODO [config]     add TOKEN_SYMMETRIC_KEY (32+ chars), ACCESS_TOKEN_DURATION, REFRESH_TOKEN_DURATION
// TODO [token]      token maker (JWT): CreateToken(userID, duration) + VerifyToken(token)
// TODO [middleware] authMiddleware: read "Authorization: Bearer <token>", verify, load the user,
//                   reject if status != "active", put {UserID, Role} in the gin context
// TODO [middleware] adminMiddleware: reject if role != "admin"
// TODO [middleware] apply auth to /users/me*, /users/phone|email|password|close, /users/me/kyc
//                   apply auth + admin to /admin/*
// TODO [handlers]   every "userID := int64(1)" / "id := int64(1)" -> read from the token
// TODO [login]      after the password check: reject non-active users, create access token,
//                   create refresh token (store only its HASH in refresh_tokens), return both
// TODO [endpoints]  POST /auth/refresh, POST /auth/logout, POST /auth/logout-all
// TODO [sqlc]       CreateRefreshToken, GetRefreshTokenByHash, RevokeRefreshToken, RevokeAllUserRefreshTokens
// TODO [admin]      admins must not be able to change status / role / delete THEMSELVES
// ======================================================================

type createUserRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Img string `json:"img" binding:"omitempty,url"`
	Phone     string `json:"phone"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type updateUserRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Img       *string `json:"img" binding:"omitempty,url"`
}
type updateEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	CurrentPassword string `json:"current_password" binding:"required"`
}

type updatePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=72"`
}

type updatePhoneRequest struct {
	Phone string `json:"phone"`
}

type updateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active suspended closed"`
}


type userResponse struct {
	ID        int64   `json:"id"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
	Img       *string `json:"img"`
	Phone     *string `json:"phone"`
	Status       string `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
		Status:       user.Status,
		CreatedAt:   user.CreatedAt.Time,
		UpdatedAt:    user.UpdatedAt.Time,
	}
}

func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
	ctx.JSON(http.StatusInternalServerError, errorResponse(err))
	return
	}

	user, err := server.store.CreateUser(ctx, db.CreateUserParams{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		Img:          textOrNull(req.Img),
		Phone:        textOrNull(req.Phone),
		PasswordHash: hashedPassword,
		Status:       "active",
	})

	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newUserResponse(user))
}

func (server *Server) getUser(ctx *gin.Context) {
	// todo add real id that u get from token 
		userId :=1
	user, err := server.store.GetUser(ctx, int64(userId))
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, newUserResponse(user))
}


func (server *Server) updateUser(ctx *gin.Context) {
	// TODO: id from the token
	userId := 1

	var req updateUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	
	current, err := server.store.GetUser(ctx, int64(userId))
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	firstName, lastName, img := current.FirstName, current.LastName, current.Img

	if req.FirstName != nil {
		firstName = *req.FirstName
	}
	if req.LastName != nil {
		lastName = *req.LastName
	}
	if req.Img != nil {
		img = textOrNull(*req.Img) 
	}

	user, err := server.store.UpdateUser(ctx, db.UpdateUserParams{
		ID:int64(userId),
		FirstName :  firstName,
		LastName:  lastName,
		Img:       img,
	},)

	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newUserResponse(user))
}




func (server *Server) updateUserEmail(ctx *gin.Context) {

	// TODO: id from the token
	userId := 1

	var req updateEmailRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user , err := server.store.GetUser(ctx , int64(userId))
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	if !utils.CheckPasswordHash(req.CurrentPassword, user.PasswordHash) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}
	UpdatedUser, err := server.store.UpdateUserEmail(ctx, db.UpdateUserEmailParams{
		ID:    user.ID,
		Email: req.Email,
	})

	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newUserResponse(UpdatedUser))
}

func (server *Server) updateUserPassword(ctx *gin.Context) {
	// TODO: userID from the token
	userID := int64(1)

	var req updatePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.store.GetUser(ctx, userID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if !utils.CheckPasswordHash(req.CurrentPassword, user.PasswordHash) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}

	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	if err := server.store.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: string(hash),
	}); err != nil {
		handleDBError(ctx, err)
		return
	}


	// TODO REVOKE REFRESHTOKENS ALL OF THEM
	ctx.Status(http.StatusNoContent)
}

func (server *Server) updateUserPhone(ctx *gin.Context) {
	// TODO: id from the token
	userId := 1
	var req updatePhoneRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := server.store.UpdateUserPhone(ctx, db.UpdateUserPhoneParams{
		ID:    int64(userId),
		Phone: textOrNull(req.Phone),
	})

	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}



type closeAccountRequest struct {
	Password string `json:"password" binding:"required"`
}

func (server *Server) closeUserAccount(ctx *gin.Context) {
	userID := int64(1)

	var req closeAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.store.GetUser(ctx, userID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "password is incorrect"})
		return
	}

	if !server.requireZeroBalances(ctx, userID) {
	return
	}


	closedAcc, err := server.store.CloseUserAccount(ctx, userID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if closedAcc.Status != "closed" {
		ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("failed to close user account")))
		return
	}

	// TODO: revoke ALL refresh tokens

	ctx.JSON(http.StatusOK, gin.H{"message": "Account closed successfully"})
}


// ADMIN


func (server *Server) listUsers(ctx *gin.Context) {
	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "page must be a number"})
		return
	}

	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a number"})
		return
	}

	const maxLimit = 100
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset := (page - 1) * limit

	users, err := server.store.GetAllUsers(ctx, db.GetAllUsersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	allUsers := make([]userResponse, 0, len(users))
	for _, u := range users {
		allUsers = append(allUsers, newUserResponse(u))
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data":  allUsers,
		"page":  page,
		"limit": limit,
	})
}


func (server *Server) getUserByAdmin(ctx *gin.Context) {
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

type setUserStatusReq struct {
Status string `json:"status" binding:"required,oneof=active suspended closed"`
}

func (server *Server) setUserStatus(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}
	// todo add real current id 
	currentId := 1
	if id ==  int64(currentId) { // admin's own id
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "you cannot change your own status"})
		return
	}

	var req setUserStatusReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if req.Status == "closed" && !server.requireZeroBalances(ctx, id) {
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

	// TODO: if req.Status != "active", revoke that user's refresh tokens

	ctx.JSON(http.StatusOK, newUserResponse(user))
}

func (server *Server) closeUserAccountByAdmin(ctx *gin.Context) {

	// TODO: refuse if this id is the admin's own id; revoke that user's refresh tokens
	id, ok := parseID(ctx)
	if !ok {
		return
	}

	// todo add real current id 
	currentId := 1
	if id ==  int64(currentId) { // admin's own id
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "you cannot close  your own account "})
		return
	}

	if !server.requireZeroBalances(ctx, id) {
	return
}

	_, err := server.store.CloseUserAccount(ctx, id)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Account closed successfully",
	})
}

func (server *Server) requireZeroBalances(ctx *gin.Context, userID int64) bool {
	accounts, err := server.store.ListAccountsByUserID(ctx, userID)
	if err != nil {
		handleDBError(ctx, err)
		return false
	}
	for _, a := range accounts {
		if a.Balance != 0 {
			ctx.JSON(http.StatusConflict, gin.H{"error": "user still has funds; move or withdraw them first"})
			return false
		}
	}
	return true
}