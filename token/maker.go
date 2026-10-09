package token

import "time"

type Maker interface {
	// creates a new token for a specific username 
	
	CreateToken(email string , userID int64  ,role string  ,duration time.Duration) (string , error)
	// check if input token is valid or not 
	VerifyToken(token string ) (*Payload , error)
}