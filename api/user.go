package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/utils"
)

type createUserRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Img string `json:"img binding:"url"`
	Email     string `json:"email" binding:"required,email"`
	Phone     string `json:"phone"`
	Password  string `json:"password" binding:"required,min=8,max=72"`
}

type userResponse struct {
	ID        int64     `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	Img string `json:"img"`
	CreatedAt time.Time `json:"created_at"`
}

func (server *Server) CreateUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	user, err := server.store.CreateUser(ctx, db.CreateUserParams{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		Phone:        pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
		PasswordHash: string(hash),
		Img: pgtype.Text{
			String: req.Img,
			Valid: req.Img != "",
		},
		Status:       "active",
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, userResponse{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Time, 
	})
}



type LoginUserRequest struct  {
	Email  string  `json:"email"`
	Password string `json:"password"`
}




func (server *Server) LoginUser(ctx *gin.Context) {
	var req LoginUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("invalid credentials")))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	passwordIsValid := utils.CheckPasswordHash(
		req.Password,
		user.PasswordHash,
	)

	if !passwordIsValid {
		ctx.JSON(
			http.StatusUnauthorized,
			errorResponse(errors.New("invalid credentials")),
		)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "login successful",
	})
}






//  GET USER KYC AND PROFILE 


type kycSummary struct {
	Status          string     `json:"status"` // not_started | pending | verified | rejected
	DocumentType    *string    `json:"document_type"`
	RejectionReason *string    `json:"rejection_reason"`
	VerifiedAt      *time.Time `json:"verified_at"`
}

type accountSummary struct {
	ID       int64  `json:"id"`
	Currency string `json:"currency"`
	Balance  int64  `json:"balance"`
	Status   string `json:"status"`
}

type meResponse struct {
	ID        int64            `json:"id"`
	FirstName string           `json:"first_name"`
	LastName  string           `json:"last_name"`
	Email     string           `json:"email"`
	Img       *string           `json:"img"`
	Phone     *string          `json:"phone"`
	Status    string           `json:"status"`
	CreatedAt time.Time        `json:"created_at"`
	KYC       kycSummary       `json:"kyc"`
	Accounts  []accountSummary `json:"accounts"`
}



func (server *Server) GetCurrentUserProfile(ctx  *gin.Context){
// todo : get user id from token 
userID := int64(1) 

currentUser , err :=	server.store.GetUser(ctx , userID)

if err != nil {
if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
}
kyc := kycSummary{Status: "not_started"}

kycVerification , err  := server.store.GetLatestKYCByUser(ctx , userID)

switch {
	case err == nil:
		kyc = kycSummary{
			Status:          kycVerification.Status,
			DocumentType:    &kycVerification.DocumentType,
			RejectionReason: ptrText(kycVerification.RejectionReason),
			VerifiedAt:      ptrTime(kycVerification.VerifiedAt),
		}
	case !errors.Is(err, pgx.ErrNoRows):
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
}


	accountsFromDb , err := server.store.ListAccountsByUserID(ctx , userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return

	}

	accounts := make([]accountSummary, 0, len(accountsFromDb))
	for _, a := range accountsFromDb {
		accounts = append(accounts, accountSummary{a.ID, a.Currency, a.Balance, a.Status })
	}
	ctx.JSON(http.StatusOK, meResponse{
		ID:        currentUser.ID,
		FirstName: currentUser.FirstName,
		LastName:  currentUser.LastName,
		Img :	   ptrText(currentUser.Img),
		Email:     currentUser.Email,
		Phone:     ptrText(currentUser.Phone),
		Status:    currentUser.Status,
		CreatedAt: currentUser.CreatedAt.Time,
		KYC:       kyc,
		Accounts:  accounts,
	})
}




// UPDATE PROFILE WITH PATCH 
type updateProfileRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Img       *string `json:"img"`
}

func (server *Server) UpdateCurrentUserProfile(ctx *gin.Context) {

	userID := int64(1) 

	var req updateProfileRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}


	// Get current user
user , err :=	server.store.GetUser(ctx , userID)

if err != nil {
if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
}
	firstName := user.FirstName
	lastName := user.LastName
	img := user.Img

	if req.FirstName != nil {
		firstName = *req.FirstName
	}

	if req.LastName != nil {
		lastName = *req.LastName
	}

	if req.Img != nil {
		img = pgtype.Text{
			String: *req.Img,
			Valid:  true,
		}
	}

	// Update user
	updatedUser, err := server.store.UpdateUser(ctx, db.UpdateUserParams{
		ID:        userID,
		FirstName: firstName,
		LastName:  lastName,
		Img:       img,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, updatedUser)
}



type UpdateUserPhoneRequest struct {
	Phone string `json:"phone"`
}

func (server *Server) UpdateUserPhone(ctx *gin.Context){
		userID := int64(1) 

		var req  UpdateUserPhoneRequest
	 	err := ctx.ShouldBindJSON(&req)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(err)) 
		}
		_ , err =	server.store.GetUser(ctx , userID)

		if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	    }

		 err  = server.store.UpdateUserPhone(ctx , db.UpdateUserPhoneParams{
				ID: userID,
				Phone :pgtype.Text{
					String: req.Phone,
					Valid: true,
				},
		})
		if err != nil {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			ctx.JSON(http.StatusConflict, errorResponse(errors.New("phone number is already in use")))
			return
		}
	}

	ctx.JSON(http.StatusInternalServerError, errorResponse(err))
	return
	}	
	
}



// update user's email 

type UpdateUserEmailRequest struct {
	Email string `json:"email"`
}

func (server *Server) UpdateUserEmail(ctx *gin.Context) {
	userID := int64(1)

	var req UpdateUserEmailRequest
	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	_, err = server.store.GetUser(ctx, userID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

  _, err = server.store.UpdateUserEmail(ctx, db.UpdateUserEmailParams{
		ID: userID,
		Email: req.Email,
	})

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				ctx.JSON(http.StatusConflict, errorResponse(errors.New("email is already in use")))
				return
			}
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
}



// 
func ptrText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func ptrTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}


