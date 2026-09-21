package db

import (
	"context"
	"testing"
	"time"

	"github.com/bank-app/utils"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createKYC(t *testing.T) (User, KycVerification) {
	user := createNewUser(t)

	arg := CreateKYCVerificationParams{
		UserID:         user.ID,
		DocumentType:   "passport",
		DocumentNumber: pgtype.Text{
			String: utils.RandomString(10),
			Valid: true,	
		},
		Status:         "pending",
		RejectionReason: pgtype.Text{
			Valid: false,
		},
		VerifiedAt: pgtype.Timestamptz{
			Valid: false,
		},
	}

	verification, err := testQueries.CreateKYCVerification(
		context.Background(),
		arg,
	)

	require.NoError(t, err)
	require.NotEmpty(t, verification)

	require.Equal(t, verification.UserID, user.ID)
	require.Equal(t, verification.DocumentType, arg.DocumentType)
	require.Equal(t, verification.DocumentNumber, arg.DocumentNumber)
	require.Equal(t, verification.Status, arg.Status)

	require.False(t, verification.RejectionReason.Valid)
	require.False(t, verification.VerifiedAt.Valid)

	return user, verification
}

// CREATE KYC VERIFICATION

func TestCreateKYCVerification(t *testing.T) {
	createKYC(t)
}


// GET KYC VERIFICATION BY ID

func TestGetKYCVerification(t *testing.T) {
	_, verification := createKYC(t)

	verificationFetched, err := testQueries.GetKYCVerification(
		context.Background(),
		verification.ID,
	)

	require.NoError(t, err)
	require.NotEmpty(t, verificationFetched)

	require.Equal(t, verification.ID, verificationFetched.ID)
	require.Equal(t, verification.UserID, verificationFetched.UserID)
	require.Equal(t, verification.DocumentType, verificationFetched.DocumentType)
	require.Equal(t, verification.DocumentNumber, verificationFetched.DocumentNumber)
	require.Equal(t, verification.Status, verificationFetched.Status)
	require.Equal(t, verification.CreatedAt, verificationFetched.CreatedAt)
}


// GET KYC VERIFICATIONS BY USER ID

func TestGetKYCVerificationByUserID(t *testing.T) {
	user, verification := createKYC(t)

	verifications, err := testQueries.GetKYCVerificationByUserId(
		context.Background(),
		user.ID,
	)

	require.NoError(t, err)
	require.Len(t, verifications, 1)

	require.Equal(t, verification.ID, verifications[0].ID)
	require.Equal(t, user.ID, verifications[0].UserID)
	require.Equal(t, verification.Status, verifications[0].Status)
}

// LIST OF ALL THE KYC VERIFICATIONS 

func TestListKYCVerifications(t *testing.T) {
	
	for i := 0; i < 5; i++ {
		user := createNewUser(t)
		arg := CreateKYCVerificationParams{
			UserID:         user.ID,
			DocumentType:   "passport",
			DocumentNumber: pgtype.Text{  
			 String:	utils.RandomString(10),
			 Valid: true,
			},
			Status:         "pending",
			RejectionReason: pgtype.Text{
				Valid: false,
			},
			VerifiedAt: pgtype.Timestamptz{
				Valid: false,
			},
		}

		_, err := testQueries.CreateKYCVerification(
			context.Background(),
			arg,
		)

		require.NoError(t, err)
	}

	arg := ListKYCVerificationsParams{
	Limit: 5,
	Offset: 0,
	}

	verifications, err := testQueries.ListKYCVerifications(context.Background(),arg)

	require.NoError(t, err)
	require.Len(t, verifications, 5)

	for _, verification := range verifications {
		require.NotEmpty(t, verification.DocumentType)
		require.NotEmpty(t, verification.DocumentNumber)
		require.Equal(t, "pending", verification.Status)
	}
}



// CHANGE PENDING → VERIFIED

func TestUpdateKYCStatusToVerified(t *testing.T) {
	_, verification := createKYC(t)

	verifiedAt := pgtype.Timestamptz{
		Time:  time.Now(),
		Valid: true,
	}

	updated, err := testQueries.VerifyKYC(
		context.Background(),
		VerifyKYCParams{
			ID:         verification.ID,
			Status:     "verified",
		},
	)

	require.NoError(t, err)

	require.Equal(t, "verified", updated.Status)
	require.True(t, updated.VerifiedAt.Valid)

	require.WithinDuration(
		t,
		verifiedAt.Time,
		updated.VerifiedAt.Time,
		time.Second,
	)

	require.False(t, updated.RejectionReason.Valid)
}




// CHANGE PENDING → REJECTED

func TestUpdateKYCStatusToRejected(t *testing.T) {
	_, verification := createKYC(t)

	reason := pgtype.Text{
		String: "Document is expired",
		Valid:  true,
	}

	updated, err := testQueries.RejectKYC(
		context.Background(),
		RejectKYCParams{
			ID:              verification.ID,
			RejectionReason: reason,
		},
	)

	require.NoError(t, err)
	require.Equal(t, "rejected", updated.Status)
	require.True(t, updated.RejectionReason.Valid)
	require.Equal(t, "Document is expired", updated.RejectionReason.String)
	require.False(t, updated.VerifiedAt.Valid)
}

