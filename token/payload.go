package token

import (
	"errors"
	"time"

	"github.com/google/uuid"
)
type Payload struct {
	ID uuid.UUID `json:"id"`
	UserID int64 `json:"user_id"`
	Role string `json:"role"`
	Email string `json:"email"`
	IssuedAt time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}




func NewPayload(email string , userID int64  , role string  ,duration time.Duration) (*Payload , error){
	tokenID , err :=uuid.NewRandom()
	if err !=nil  {
		return  nil, err
	}
payload := &Payload{
	ID:        tokenID,
	UserID:    userID,
	Role:      role,
	Email:     email,
	IssuedAt:  time.Now(),
	ExpiredAt: time.Now().Add(duration),
}
	return payload , nil 
}




func (payload *Payload) Valid() error {
	if ( time.Now().After(payload.ExpiredAt)){
		return errors.New("Token has expired")
	}
	return  nil 
}





