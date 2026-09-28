package db

import (
	"context"
	"testing"
	"time"

	"github.com/bank-app/utils"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createInvoiceUtil(t *testing.T) Invoice {
	business := createNewBusiness(t)

	args := CreateInvoiceParams{
		BusinessID: business.ID,
		CustomerName: "Ayoub",
		CustomerEmail: pgtype.Text{
			String: "ayoub@gmail.com",
			Valid:  true,
		},
		Amount:   20000,
		Currency: "USD",
		Status:   "draft",
		DueDate: pgtype.Date{
			Time:  time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC),
			Valid: true,
		},
		PaidAt: pgtype.Timestamptz{
			Valid: false,
		},
	}

	invoice, err := testQueries.CreateInvoice(
		context.Background(),
		args,
	)

	require.NoError(t, err)
	require.NotZero(t, invoice.ID)
	require.NotZero(t, invoice.CreatedAt)

	require.Equal(t, args.BusinessID, invoice.BusinessID)
	require.Equal(t, args.CustomerName, invoice.CustomerName)
	require.Equal(t, args.CustomerEmail, invoice.CustomerEmail)
	require.Equal(t, args.Amount, invoice.Amount)
	require.Equal(t, args.Currency, invoice.Currency)
	require.Equal(t, args.DueDate.Time, invoice.DueDate.Time)
	require.Equal(t, args.PaidAt, invoice.PaidAt)

	invoiceFromDB, err := testQueries.GetInvoiceById(
		context.Background(),
		invoice.ID,
	)

	require.NoError(t, err)

	require.Equal(t, invoice.BusinessID, invoiceFromDB.BusinessID)
	require.Equal(t, invoice.CustomerName, invoiceFromDB.CustomerName)
	require.Equal(t, invoice.CustomerEmail, invoiceFromDB.CustomerEmail)
	require.Equal(t, invoice.Amount, invoiceFromDB.Amount)
	require.Equal(t, invoice.Currency, invoiceFromDB.Currency)
	require.Equal(t, invoice.DueDate.Time, invoiceFromDB.DueDate.Time)
	require.Equal(t, invoice.PaidAt, invoiceFromDB.PaidAt)

	return invoice
}

func TestCreateAndGetInvoice(t *testing.T) {
	createInvoiceUtil(t)
}

func TestListInvoicesByBusinessID(t *testing.T) {
	business := createNewBusiness(t)

	n := 4
	invoices := make([]Invoice, 0, n)

	for i := 0; i < n; i++ {
		args := CreateInvoiceParams{
			BusinessID: business.ID,
			CustomerName: utils.RandomFirstName(),
			CustomerEmail: pgtype.Text{
				String: utils.RandomEmail(),
				Valid:  true,
			},
			Amount:   utils.RandomInt(0, 6),
			Currency: "USD",
			Status:   "draft",
			DueDate: pgtype.Date{
				Time:  time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC),
				Valid: true,
			},
			PaidAt: pgtype.Timestamptz{
				Valid: false,
			},
		}

		invoice, err := testQueries.CreateInvoice(
			context.Background(),
			args,
		)

		require.NoError(t, err)
		require.NotZero(t, invoice.ID)

		invoices = append(invoices, invoice)
	}

	invoicesFromDB, err := testQueries.ListInvoicesByBusinessID(
		context.Background(),
		ListInvoicesByBusinessIDParams{
			BusinessID: business.ID,
			Limit:      4,
			Offset:     0,
		},
	)

	require.NoError(t, err)
	require.Len(t, invoicesFromDB, n)

	for i := 0; i < n; i++ {
		require.Equal(t, invoices[i].ID, invoicesFromDB[i].ID)
		require.Equal(t, invoices[i].BusinessID, invoicesFromDB[i].BusinessID)
		require.Equal(t, invoices[i].CustomerName, invoicesFromDB[i].CustomerName)
		require.Equal(t, invoices[i].CustomerEmail, invoicesFromDB[i].CustomerEmail)
		require.Equal(t, invoices[i].Amount, invoicesFromDB[i].Amount)
		require.Equal(t, invoices[i].Currency, invoicesFromDB[i].Currency)
		require.Equal(t, invoices[i].Status, invoicesFromDB[i].Status)
		require.Equal(t, invoices[i].DueDate, invoicesFromDB[i].DueDate)
		require.Equal(t, invoices[i].PaidAt, invoicesFromDB[i].PaidAt)
	}
}

func TestUpdateInvoiceDetails(t *testing.T) {
	invoice := createInvoiceUtil(t)

	newCustomerName := "Updated Customer"

	newCustomerEmail := pgtype.Text{
		String: "updated@gmail.com",
		Valid:  true,
	}

	newDueDate := pgtype.Date{
		Time:  time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC),
		Valid: true,
	}

	updatedInvoice, err := testQueries.UpdateInvoiceDetails(
		context.Background(),
		UpdateInvoiceDetailsParams{
			ID:            invoice.ID,
			CustomerName:  newCustomerName,
			CustomerEmail: newCustomerEmail,
			DueDate:       newDueDate,
		},
	)

	require.NoError(t, err)

	require.Equal(t, invoice.ID, updatedInvoice.ID)
	require.Equal(t, newCustomerName, updatedInvoice.CustomerName)
	require.Equal(t, newCustomerEmail, updatedInvoice.CustomerEmail)
	require.Equal(t, newDueDate, updatedInvoice.DueDate)

	// Make sure the invoice is still a draft.
	require.Equal(t, "draft", updatedInvoice.Status)
}

func TestIssueInvoice(t *testing.T) {
	invoice := createInvoiceUtil(t)

	require.Equal(t, "draft", invoice.Status)

	issuedInvoice, err := testQueries.IssueInvoice(
		context.Background(),
		invoice.ID,
	)

	require.NoError(t, err)

	require.Equal(t, invoice.ID, issuedInvoice.ID)
	require.Equal(t, "pending", issuedInvoice.Status)
}

func TestMarkInvoiceAsOverdue(t *testing.T) {
	business := createNewBusiness(t)

	args := CreateInvoiceParams{
		BusinessID: business.ID,
		CustomerName: "Overdue Customer",
		CustomerEmail: pgtype.Text{
			String: "overdue@gmail.com",
			Valid:  true,
		},
		Amount:   20000,
		Currency: "USD",
		Status:   "draft",
		DueDate: pgtype.Date{
			// Yesterday so CURRENT_DATE will consider it overdue.
			Time:  time.Now().UTC().AddDate(0, 0, -1),
			Valid: true,
		},
		PaidAt: pgtype.Timestamptz{
			Valid: false,
		},
	}

	invoice, err := testQueries.CreateInvoice(
		context.Background(),
		args,
	)

	require.NoError(t, err)
	require.Equal(t, "draft", invoice.Status)

	pendingInvoice, err := testQueries.IssueInvoice(
		context.Background(),
		invoice.ID,
	)

	require.NoError(t, err)
	require.Equal(t, "pending", pendingInvoice.Status)

	overdueInvoices, err := testQueries.MarkInvoiceAsOverdue(
		context.Background(),
	)

	require.NoError(t, err)
	require.NotEmpty(t, overdueInvoices)

	var foundInvoice Invoice

	for _, overdueInvoice := range overdueInvoices {
		if overdueInvoice.ID == invoice.ID {
			foundInvoice = overdueInvoice
			break
		}
	}

	require.NotZero(t, foundInvoice.ID)
	require.Equal(t, invoice.ID, foundInvoice.ID)
	require.Equal(t, "overdue", foundInvoice.Status)
	require.False(t, foundInvoice.PaidAt.Valid)
}

func TestCancelInvoice(t *testing.T) {
	invoice := createInvoiceUtil(t)

	require.Equal(t, "draft", invoice.Status)

	cancelledInvoice, err := testQueries.CancelInvoice(
		context.Background(),
		invoice.ID,
	)

	require.NoError(t, err)

	require.Equal(t, invoice.ID, cancelledInvoice.ID)
	require.Equal(t, "cancelled", cancelledInvoice.Status)
	require.False(t, cancelledInvoice.PaidAt.Valid)
}

func TestDeleteInvoice(t *testing.T) {
	invoice := createInvoiceUtil(t)

	require.Equal(t, "draft", invoice.Status)

	deletedInvoice, err := testQueries.DeleteInvoice(
		context.Background(),
		invoice.ID,
	)

	require.NoError(t, err)

	require.Equal(t, invoice.ID, deletedInvoice.ID)
	require.Equal(t, invoice.BusinessID, deletedInvoice.BusinessID)
	require.Equal(t, invoice.CustomerName, deletedInvoice.CustomerName)
	require.Equal(t, invoice.CustomerEmail, deletedInvoice.CustomerEmail)
	require.Equal(t, invoice.Amount, deletedInvoice.Amount)
	require.Equal(t, invoice.Currency, deletedInvoice.Currency)
	require.Equal(t, "draft", deletedInvoice.Status)

	// Make sure it was actually deleted.
	_, err = testQueries.GetInvoiceById(
		context.Background(),
		invoice.ID,
	)

	require.Error(t, err)
}