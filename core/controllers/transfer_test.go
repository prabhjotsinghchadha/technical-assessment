package controllers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/ortizdavid/go-bank-core-api/core/controllers"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testIDSeq atomic.Int64

type testAccount struct {
	ID         int64
	Number     string
	CustomerID int64
}

func TestTransferSuccess(t *testing.T) {
	db, mux := setup(t)
	source := createTestAccount(t, db, 1000, 1)
	dest := createTestAccount(t, db, 100, 1)

	rec := doTransfer(t, mux, source.Number, dest.Number, 250.50, "USD", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertNoDatabaseErrorLeak(t, rec.Body.String())

	if got := getBalance(t, db, source.ID); !almostEqual(got, 749.50) {
		t.Fatalf("source balance: got %.2f, want 749.50", got)
	}
	if got := getBalance(t, db, dest.ID); !almostEqual(got, 350.50) {
		t.Fatalf("destination balance: got %.2f, want 350.50", got)
	}
	if n := countTransfers(t, db, source.ID, dest.ID); n != 1 {
		t.Fatalf("expected 1 transfer row, got %d", n)
	}

	var row struct {
		Amount              float64
		BalanceBefore       float64
		BalanceAfter        float64
		Currency            string
		TransactionStatusId int
	}
	id := decodeTransfer(t, rec).Data.TransactionId
	if err := db.Raw(`SELECT amount, balance_before, balance_after, currency, transaction_status_id FROM transactions WHERE transaction_id = ?`, id).Scan(&row).Error; err != nil {
		t.Fatalf("read transaction: %v", err)
	}
	if !almostEqual(row.Amount, 250.50) || !almostEqual(row.BalanceBefore, 1000) || !almostEqual(row.BalanceAfter, 749.50) {
		t.Fatalf("unexpected transaction row: %+v", row)
	}
	if row.Currency != "USD" || row.TransactionStatusId != 2 {
		t.Fatalf("expected a completed USD transaction, got %+v", row)
	}
}

func TestTransferInsufficientBalance(t *testing.T) {
	db, mux := setup(t)
	source := createTestAccount(t, db, 50, 1)
	dest := createTestAccount(t, db, 0, 1)

	rec := doTransfer(t, mux, source.Number, dest.Number, 100, "USD", "")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertNoDatabaseErrorLeak(t, rec.Body.String())
	assertUnchanged(t, db, source, dest, 50, 0)
}

func TestTransferInvalidAmount(t *testing.T) {
	db, mux := setup(t)
	source := createTestAccount(t, db, 500, 1)
	dest := createTestAccount(t, db, 0, 1)

	t.Run("zero", func(t *testing.T) {
		rec := doTransfer(t, mux, source.Number, dest.Number, 0, "USD", "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for zero amount, got %d body=%s", rec.Code, rec.Body.String())
		}
		assertNoDatabaseErrorLeak(t, rec.Body.String())
		assertUnchanged(t, db, source, dest, 500, 0)
	})

	t.Run("negative", func(t *testing.T) {
		rec := doTransfer(t, mux, source.Number, dest.Number, -25, "USD", "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for negative amount, got %d body=%s", rec.Code, rec.Body.String())
		}
		assertNoDatabaseErrorLeak(t, rec.Body.String())
		assertUnchanged(t, db, source, dest, 500, 0)
	})

	t.Run("fraction_of_a_cent", func(t *testing.T) {
		rec := doTransfer(t, mux, source.Number, dest.Number, 10.005, "USD", "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for a sub-cent amount, got %d body=%s", rec.Code, rec.Body.String())
		}
		assertUnchanged(t, db, source, dest, 500, 0)
	})
}

func TestTransferInvalidAccount(t *testing.T) {
	db, mux := setup(t)
	source := createTestAccount(t, db, 500, 1)
	dest := createTestAccount(t, db, 0, 1)

	t.Run("missing_source", func(t *testing.T) {
		rec := doTransfer(t, mux, "0000000000", dest.Number, 10, "USD", "")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
		}
		assertNoDatabaseErrorLeak(t, rec.Body.String())
		assertUnchanged(t, db, source, dest, 500, 0)
	})

	t.Run("missing_destination", func(t *testing.T) {
		rec := doTransfer(t, mux, source.Number, "0000000000", 10, "USD", "")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
		}
		assertNoDatabaseErrorLeak(t, rec.Body.String())
		assertUnchanged(t, db, source, dest, 500, 0)
	})
}

func TestTransferInactiveAccount(t *testing.T) {
	db, mux := setup(t)

	t.Run("inactive_source", func(t *testing.T) {
		source := createTestAccount(t, db, 500, 2)
		dest := createTestAccount(t, db, 0, 1)
		rec := doTransfer(t, mux, source.Number, dest.Number, 10, "USD", "")
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d body=%s", rec.Code, rec.Body.String())
		}
		assertNoDatabaseErrorLeak(t, rec.Body.String())
		assertUnchanged(t, db, source, dest, 500, 0)
	})

	t.Run("inactive_destination", func(t *testing.T) {
		source := createTestAccount(t, db, 500, 1)
		dest := createTestAccount(t, db, 0, 2)
		rec := doTransfer(t, mux, source.Number, dest.Number, 10, "USD", "")
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d body=%s", rec.Code, rec.Body.String())
		}
		assertNoDatabaseErrorLeak(t, rec.Body.String())
		assertUnchanged(t, db, source, dest, 500, 0)
	})
}

func TestTransferSameSourceAndDestination(t *testing.T) {
	db, mux := setup(t)
	account := createTestAccount(t, db, 500, 1)

	rec := doTransfer(t, mux, account.Number, account.Number, 25, "USD", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertNoDatabaseErrorLeak(t, rec.Body.String())
	if got := getBalance(t, db, account.ID); !almostEqual(got, 500) {
		t.Fatalf("balance changed: got %.2f, want 500", got)
	}
	if n := countTransfers(t, db, account.ID, account.ID); n != 0 {
		t.Fatalf("expected no transfer rows, got %d", n)
	}
}

func TestTransferCurrencyMismatch(t *testing.T) {
	db, mux := setup(t)
	source := createTestAccount(t, db, 500, 1)
	dest := createTestAccount(t, db, 0, 1)

	rec := doTransfer(t, mux, source.Number, dest.Number, 10, "EUR", "")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertUnchanged(t, db, source, dest, 500, 0)
}

func TestTransferInvalidRequestData(t *testing.T) {
	db, mux := setup(t)
	source := createTestAccount(t, db, 500, 1)
	dest := createTestAccount(t, db, 0, 1)
	valid := fmt.Sprintf(`{"source_number":%q,"destination_number":%q,"amount":10,"currency":"USD"}`, source.Number, dest.Number)

	cases := []struct {
		name string
		body string
		key  string
	}{
		{name: "malformed_json", body: `{not-json`},
		{name: "missing_source", body: fmt.Sprintf(`{"destination_number":%q,"amount":10,"currency":"USD"}`, dest.Number)},
		{name: "missing_destination", body: fmt.Sprintf(`{"source_number":%q,"amount":10,"currency":"USD"}`, source.Number)},
		{name: "missing_amount", body: fmt.Sprintf(`{"source_number":%q,"destination_number":%q,"currency":"USD"}`, source.Number, dest.Number)},
		{name: "amount_not_a_number", body: fmt.Sprintf(`{"source_number":%q,"destination_number":%q,"amount":"ten","currency":"USD"}`, source.Number, dest.Number)},
		{name: "invalid_currency", body: fmt.Sprintf(`{"source_number":%q,"destination_number":%q,"amount":10,"currency":"US"}`, source.Number, dest.Number)},
		{name: "idempotency_key_too_long", body: valid, key: strings.Repeat("k", 256)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := post(mux, "/api/transactions/transfer", []byte(tc.body), tc.key)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
			}
			assertNoDatabaseErrorLeak(t, rec.Body.String())
			assertUnchanged(t, db, source, dest, 500, 0)
		})
	}
}

// A database failure after both balances were updated must undo the whole
// transfer, hide the database error, and leave the idempotency key unused.
func TestTransferRollback(t *testing.T) {
	db, mux := setup(t)
	source := createTestAccount(t, db, 400, 1)
	dest := createTestAccount(t, db, 50, 1)
	key := uniqueValue("rollback")

	restore := failTransactionInserts(t, db, source.ID)
	rec := doTransfer(t, mux, source.Number, dest.Number, 100, "USD", key)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertNoDatabaseErrorLeak(t, rec.Body.String())
	if strings.Contains(rec.Body.String(), "simulated failure") {
		t.Fatalf("response leaked the database error: %s", rec.Body.String())
	}
	assertUnchanged(t, db, source, dest, 400, 50)

	restore()
	retry := doTransfer(t, mux, source.Number, dest.Number, 100, "USD", key)
	if retry.Code != http.StatusCreated {
		t.Fatalf("retry after failure expected 201, got %d body=%s", retry.Code, retry.Body.String())
	}
	if got := getBalance(t, db, source.ID); !almostEqual(got, 300) {
		t.Fatalf("source balance: got %.2f, want 300", got)
	}
	if got := getBalance(t, db, dest.ID); !almostEqual(got, 150) {
		t.Fatalf("destination balance: got %.2f, want 150", got)
	}
	if n := countTransfers(t, db, source.ID, dest.ID); n != 1 {
		t.Fatalf("expected a single transfer row, got %d", n)
	}
}

func TestTransferDuplicateIdempotencyKey(t *testing.T) {
	db, mux := setup(t)
	source := createTestAccount(t, db, 1000, 1)
	dest := createTestAccount(t, db, 0, 1)
	key := uniqueValue("idem")

	first := doTransfer(t, mux, source.Number, dest.Number, 100, "USD", key)
	if first.Code != http.StatusCreated {
		t.Fatalf("first request expected 201, got %d body=%s", first.Code, first.Body.String())
	}
	second := doTransfer(t, mux, source.Number, dest.Number, 100, "USD", key)
	if second.Code != http.StatusCreated {
		t.Fatalf("replay expected 201, got %d body=%s", second.Code, second.Body.String())
	}
	assertNoDatabaseErrorLeak(t, second.Body.String())
	if a, b := decodeTransfer(t, first).Data, decodeTransfer(t, second).Data; a != b {
		t.Fatalf("replay returned a different result: first=%+v replay=%+v", a, b)
	}

	if got := getBalance(t, db, source.ID); !almostEqual(got, 900) {
		t.Fatalf("source balance: got %.2f, want 900", got)
	}
	if got := getBalance(t, db, dest.ID); !almostEqual(got, 100) {
		t.Fatalf("destination balance: got %.2f, want 100", got)
	}
	if n := countTransfers(t, db, source.ID, dest.ID); n != 1 {
		t.Fatalf("expected a single transfer row, got %d", n)
	}
}

// A replay returns the stored result even when the same request would now be
// rejected, here because the first transfer emptied the source account.
func TestTransferIdempotentReplayIgnoresCurrentState(t *testing.T) {
	db, mux := setup(t)
	source := createTestAccount(t, db, 100, 1)
	dest := createTestAccount(t, db, 0, 1)
	key := uniqueValue("replay")

	first := doTransfer(t, mux, source.Number, dest.Number, 100, "USD", key)
	if first.Code != http.StatusCreated {
		t.Fatalf("first request expected 201, got %d body=%s", first.Code, first.Body.String())
	}
	replay := doTransfer(t, mux, source.Number, dest.Number, 100, "USD", key)
	if replay.Code != http.StatusCreated {
		t.Fatalf("replay expected 201, got %d body=%s", replay.Code, replay.Body.String())
	}
	if a, b := decodeTransfer(t, first).Data, decodeTransfer(t, replay).Data; a != b {
		t.Fatalf("replay returned a different result: first=%+v replay=%+v", a, b)
	}
	if n := countTransfers(t, db, source.ID, dest.ID); n != 1 {
		t.Fatalf("expected a single transfer row, got %d", n)
	}
}

func TestTransferIdempotencyKeyReusedForDifferentRequest(t *testing.T) {
	db, mux := setup(t)
	source := createTestAccount(t, db, 1000, 1)
	dest := createTestAccount(t, db, 0, 1)
	key := uniqueValue("reuse")

	first := doTransfer(t, mux, source.Number, dest.Number, 100, "USD", key)
	if first.Code != http.StatusCreated {
		t.Fatalf("first request expected 201, got %d body=%s", first.Code, first.Body.String())
	}
	second := doTransfer(t, mux, source.Number, dest.Number, 250, "USD", key)
	if second.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for a reused key, got %d body=%s", second.Code, second.Body.String())
	}
	assertNoDatabaseErrorLeak(t, second.Body.String())

	if got := getBalance(t, db, source.ID); !almostEqual(got, 900) {
		t.Fatalf("source balance: got %.2f, want 900", got)
	}
	if n := countTransfers(t, db, source.ID, dest.ID); n != 1 {
		t.Fatalf("expected a single transfer row, got %d", n)
	}
}

func TestConcurrentTransfers(t *testing.T) {
	db, mux := setup(t)
	source := createTestAccount(t, db, 1000, 1)
	dest := createTestAccount(t, db, 0, 1)

	const workers = 20
	const amount = 100.0
	codes := runConcurrently(workers, func(int) *httptest.ResponseRecorder {
		return postTransfer(mux, source.Number, dest.Number, amount, "USD", "")
	})

	sourceBal := getBalance(t, db, source.ID)
	destBal := getBalance(t, db, dest.ID)
	created := countTransfers(t, db, source.ID, dest.ID)

	if sourceBal < 0 {
		t.Fatalf("source balance became negative: %.2f", sourceBal)
	}
	if !almostEqual(sourceBal+destBal, 1000) {
		t.Fatalf("money was not conserved: source=%.2f dest=%.2f", sourceBal, destBal)
	}
	if created != 10 {
		t.Fatalf("expected 10 successful transfers of 100 from 1000, got %d (source=%.2f dest=%.2f codes=%v)", created, sourceBal, destBal, codes)
	}
	if !almostEqual(sourceBal, 0) || !almostEqual(destBal, 1000) {
		t.Fatalf("expected source=0 dest=1000, got source=%.2f dest=%.2f", sourceBal, destBal)
	}
	if ok, rejected := countCode(codes, http.StatusCreated), countCode(codes, http.StatusUnprocessableEntity); ok != 10 || rejected != 10 {
		t.Fatalf("expected 10 accepted and 10 rejected requests, got codes=%v", codes)
	}
}

// The example from the brief: 600 and 500 leave a balance of 1,000 at the same
// time. Exactly one of them may succeed.
func TestConcurrentTransfersCannotOverdraw(t *testing.T) {
	db, mux := setup(t)
	amounts := []float64{600, 500}

	for round := 0; round < 10; round++ {
		source := createTestAccount(t, db, 1000, 1)
		dest := createTestAccount(t, db, 0, 1)

		codes := runConcurrently(len(amounts), func(i int) *httptest.ResponseRecorder {
			return postTransfer(mux, source.Number, dest.Number, amounts[i], "USD", "")
		})
		if countCode(codes, http.StatusCreated) != 1 || countCode(codes, http.StatusUnprocessableEntity) != 1 {
			t.Fatalf("round %d: expected one accepted and one rejected transfer, got codes=%v", round, codes)
		}

		sourceBal := getBalance(t, db, source.ID)
		destBal := getBalance(t, db, dest.ID)
		if !almostEqual(sourceBal+destBal, 1000) {
			t.Fatalf("round %d: money was not conserved: source=%.2f dest=%.2f", round, sourceBal, destBal)
		}
		if !almostEqual(destBal, 600) && !almostEqual(destBal, 500) {
			t.Fatalf("round %d: unexpected final state: source=%.2f dest=%.2f", round, sourceBal, destBal)
		}
	}
}

// Transfers in opposite directions lock the same two rows. They must all
// complete instead of deadlocking.
func TestConcurrentOpposingTransfers(t *testing.T) {
	db, mux := setup(t)
	a := createTestAccount(t, db, 1000, 1)
	b := createTestAccount(t, db, 1000, 1)

	const workers = 20
	codes := runConcurrently(workers, func(i int) *httptest.ResponseRecorder {
		if i%2 == 0 {
			return postTransfer(mux, a.Number, b.Number, 10, "USD", "")
		}
		return postTransfer(mux, b.Number, a.Number, 10, "USD", "")
	})
	if countCode(codes, http.StatusCreated) != workers {
		t.Fatalf("expected every transfer to succeed, got codes=%v", codes)
	}
	if got := getBalance(t, db, a.ID); !almostEqual(got, 1000) {
		t.Fatalf("account a balance: got %.2f, want 1000", got)
	}
	if got := getBalance(t, db, b.ID); !almostEqual(got, 1000) {
		t.Fatalf("account b balance: got %.2f, want 1000", got)
	}
}

// Deposits and transfers touching the same account must not overwrite each
// other's balance updates.
func TestConcurrentDepositsAndTransfers(t *testing.T) {
	db, mux := setup(t)
	source := createTestAccount(t, db, 1000, 1)
	dest := createTestAccount(t, db, 0, 1)
	deposit := []byte(fmt.Sprintf(`{"account_number":%q,"amount":100,"currency":"USD"}`, source.Number))

	const workers = 20
	codes := runConcurrently(workers, func(i int) *httptest.ResponseRecorder {
		if i%2 == 0 {
			return post(mux, "/api/transactions/deposit", deposit, "")
		}
		return postTransfer(mux, source.Number, dest.Number, 100, "USD", "")
	})
	if countCode(codes, http.StatusCreated) != workers {
		t.Fatalf("expected every request to succeed, got codes=%v", codes)
	}
	if got := getBalance(t, db, source.ID); !almostEqual(got, 1000) {
		t.Fatalf("source balance: got %.2f, want 1000", got)
	}
	if got := getBalance(t, db, dest.ID); !almostEqual(got, 1000) {
		t.Fatalf("destination balance: got %.2f, want 1000", got)
	}
}

func TestConcurrentSameIdempotencyKey(t *testing.T) {
	db, mux := setup(t)
	source := createTestAccount(t, db, 1000, 1)
	dest := createTestAccount(t, db, 0, 1)
	key := uniqueValue("concurrent-idem")

	const workers = 10
	ids := make([]int64, workers)
	codes := runConcurrently(workers, func(i int) *httptest.ResponseRecorder {
		rec := postTransfer(mux, source.Number, dest.Number, 100, "USD", key)
		var body transferResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		ids[i] = body.Data.TransactionId
		return rec
	})

	sourceBal := getBalance(t, db, source.ID)
	destBal := getBalance(t, db, dest.ID)
	created := countTransfers(t, db, source.ID, dest.ID)

	if created != 1 {
		t.Fatalf("expected one transfer for the same idempotency key, got %d", created)
	}
	if !almostEqual(sourceBal, 900) || !almostEqual(destBal, 100) {
		t.Fatalf("expected source=900 dest=100, got source=%.2f dest=%.2f", sourceBal, destBal)
	}
	if countCode(codes, http.StatusCreated) != workers {
		t.Fatalf("expected every duplicate to get the stored result, got codes=%v", codes)
	}
	for _, id := range ids {
		if id == 0 || id != ids[0] {
			t.Fatalf("duplicates returned different transactions: %v", ids)
		}
	}
}

func TestAccountTransactionHistory(t *testing.T) {
	db, mux := setup(t)
	source := createTestAccount(t, db, 1000, 1)
	dest := createTestAccount(t, db, 0, 1)

	doTransfer(t, mux, source.Number, dest.Number, 10, "USD", "")
	latest := decodeTransfer(t, doTransfer(t, mux, source.Number, dest.Number, 20, "USD", "")).Data.TransactionId

	for _, account := range []testAccount{source, dest} {
		rec := get(mux, "/api/transactions/account/"+account.Number+"?limit=1")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		var page struct {
			Items []struct {
				TransactionId int64 `json:"transaction_id"`
			} `json:"items"`
			Metadata struct {
				TotalItems int64 `json:"total_items"`
			} `json:"metadata"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
			t.Fatalf("decode history: %v", err)
		}
		if page.Metadata.TotalItems != 2 || len(page.Items) != 1 || page.Items[0].TransactionId != latest {
			t.Fatalf("expected the newest of 2 transactions first, got %s", rec.Body.String())
		}
	}

	for _, path := range []string{
		"/api/transactions?limit=abc",
		"/api/transactions?limit=1000",
		"/api/transactions/account/" + source.Number + "?current_page=x",
	} {
		if rec := get(mux, path); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func setup(t *testing.T) (*gorm.DB, *http.ServeMux) {
	t.Helper()
	loadTestEnv()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("DATABASE_URL or TEST_DATABASE_URL is not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Skipf("PostgreSQL is not available: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("PostgreSQL is not available: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	if err := sqlDB.Ping(); err != nil {
		t.Skipf("PostgreSQL is not available: %v", err)
	}
	if err := db.Exec("SELECT 1 FROM accounts LIMIT 1").Error; err != nil {
		t.Skip("database schema is not applied; run the SQL files in _database first")
	}
	mux := http.NewServeMux()
	controllers.RegisterRoutes(mux, db)
	return db, mux
}

func loadTestEnv() {
	for _, p := range []string{".env", "../.env", "../../.env"} {
		if err := godotenv.Load(p); err == nil {
			return
		}
	}
}

func createTestAccount(t *testing.T, db *gorm.DB, balance float64, status int) testAccount {
	t.Helper()
	suffix := uniqueValue("")
	ident := "ID" + suffix
	if len(ident) > 30 {
		ident = ident[:30]
	}
	email := fmt.Sprintf("t%s@example.com", suffix)
	phone := fmt.Sprintf("5%09d", time.Now().UnixNano()%1000000000)
	var customerID int64
	if err := db.Raw(`
		INSERT INTO customers (customer_type, customer_status, customer_name, identification_number, email, phone, address)
		VALUES (1, 1, ?, ?, ?, ?, 'Test Address')
		RETURNING customer_id
	`, "Test User "+suffix, ident, email, phone).Scan(&customerID).Error; err != nil {
		t.Fatalf("insert customer: %v", err)
	}
	number := fmt.Sprintf("%010d", time.Now().UnixNano()%10000000000)
	iban := fmt.Sprintf("US%s", suffix)
	if len(iban) > 31 {
		iban = iban[:31]
	}
	var accountID int64
	if err := db.Raw(`
		INSERT INTO accounts (customer_id, account_type, account_status, account_number, iban, holder_name, balance, currency)
		VALUES (?, 1, ?, ?, ?, 'Test User', ?, 'USD')
		RETURNING account_id
	`, customerID, status, number, iban, balance).Scan(&accountID).Error; err != nil {
		t.Fatalf("insert account: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM idempotency_keys WHERE transaction_id IN (
			SELECT transaction_id FROM transactions WHERE source_id = ? OR destination_id = ?
		)`, accountID, accountID)
		db.Exec(`DELETE FROM transactions WHERE source_id = ? OR destination_id = ?`, accountID, accountID)
		db.Exec(`DELETE FROM accounts WHERE account_id = ?`, accountID)
		db.Exec(`DELETE FROM customers WHERE customer_id = ?`, customerID)
	})
	return testAccount{ID: accountID, Number: number, CustomerID: customerID}
}

func doTransfer(t *testing.T, mux http.Handler, sourceNumber, destNumber string, amount float64, currency, idempotencyKey string) *httptest.ResponseRecorder {
	t.Helper()
	return postTransfer(mux, sourceNumber, destNumber, amount, currency, idempotencyKey)
}

func postTransfer(mux http.Handler, sourceNumber, destNumber string, amount float64, currency, idempotencyKey string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]any{
		"source_number":      sourceNumber,
		"destination_number": destNumber,
		"amount":             amount,
		"currency":           currency,
	})
	return post(mux, "/api/transactions/transfer", body, idempotencyKey)
}

func post(mux http.Handler, path string, body []byte, idempotencyKey string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func get(mux http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

type transferResponse struct {
	Data struct {
		TransactionId int64
		Code          string
		Amount        float64
		BalanceBefore float64
		BalanceAfter  float64
	} `json:"data"`
}

func decodeTransfer(t *testing.T, rec *httptest.ResponseRecorder) transferResponse {
	t.Helper()
	var body transferResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rec.Body.String())
	}
	if body.Data.TransactionId == 0 {
		t.Fatalf("response has no transaction: %s", rec.Body.String())
	}
	return body
}

// runConcurrently releases all workers at the same moment and returns their
// HTTP status codes.
func runConcurrently(workers int, request func(i int) *httptest.ResponseRecorder) []int {
	codes := make([]int, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-start
			codes[i] = request(i).Code
		}()
	}
	close(start)
	wg.Wait()
	return codes
}

func countCode(codes []int, want int) int {
	n := 0
	for _, code := range codes {
		if code == want {
			n++
		}
	}
	return n
}

// failTransactionInserts makes the database reject transaction rows for one
// source account, which fails a transfer after both balances were updated.
// The returned function removes the failure.
func failTransactionInserts(t *testing.T, db *gorm.DB, sourceID int64) (restore func()) {
	t.Helper()
	name := fmt.Sprintf("test_fail_transaction_%d", sourceID)
	restore = func() {
		db.Exec(fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON transactions`, name))
		db.Exec(fmt.Sprintf(`DROP FUNCTION IF EXISTS %s()`, name))
	}
	t.Cleanup(restore)
	for _, statement := range []string{
		fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'simulated failure'; END $$ LANGUAGE plpgsql`, name),
		fmt.Sprintf(`CREATE TRIGGER %s BEFORE INSERT ON transactions FOR EACH ROW WHEN (NEW.source_id = %d) EXECUTE FUNCTION %s()`, name, sourceID, name),
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("install failing trigger: %v", err)
		}
	}
	return restore
}

func getBalance(t *testing.T, db *gorm.DB, accountID int64) float64 {
	t.Helper()
	var balance float64
	if err := db.Raw(`SELECT balance FROM accounts WHERE account_id = ?`, accountID).Scan(&balance).Error; err != nil {
		t.Fatalf("read balance: %v", err)
	}
	return balance
}

func countTransfers(t *testing.T, db *gorm.DB, sourceID, destID int64) int64 {
	t.Helper()
	var count int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM transactions
		WHERE source_id = ? AND destination_id = ? AND transaction_type_id = 3
	`, sourceID, destID).Scan(&count).Error; err != nil {
		t.Fatalf("count transfers: %v", err)
	}
	return count
}

func assertUnchanged(t *testing.T, db *gorm.DB, source, dest testAccount, sourceWant, destWant float64) {
	t.Helper()
	if got := getBalance(t, db, source.ID); !almostEqual(got, sourceWant) {
		t.Fatalf("source balance changed: got %.2f, want %.2f", got, sourceWant)
	}
	if got := getBalance(t, db, dest.ID); !almostEqual(got, destWant) {
		t.Fatalf("destination balance changed: got %.2f, want %.2f", got, destWant)
	}
	if n := countTransfers(t, db, source.ID, dest.ID); n != 0 {
		t.Fatalf("expected no transfer rows, got %d", n)
	}
}

func assertNoDatabaseErrorLeak(t *testing.T, body string) {
	t.Helper()
	lower := strings.ToLower(body)
	for _, token := range []string{"sqlstate", "pq:", "violates", "gorm", "relation ", "duplicate key"} {
		if strings.Contains(lower, token) {
			t.Fatalf("response leaked internal database details: %s", body)
		}
	}
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}

func uniqueValue(prefix string) string {
	n := time.Now().UnixNano() + testIDSeq.Add(1)
	if prefix == "" {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%s%d", prefix, n)
}
