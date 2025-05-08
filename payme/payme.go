package payme

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/BaxtiyorUrolov/Tolov-tizimlari/payme/models"
	"github.com/BaxtiyorUrolov/Tolov-tizimlari/payme/storage"
	"github.com/jmoiron/sqlx"
)

// Handler handles payme webhook requests and transactions.
type Handler struct {
	store        storage.StorageI
	paymeKey     string // payme API key, provided by the user
	paymeBaseURL string // Base URL for payme API (e.g., test or production)
}

// NewHandler creates a new Handler instance with the provided database, payme key, and base URL.
func NewHandler(db *sqlx.DB, paymeKey, paymeBaseURL string) *Handler {
	if paymeBaseURL == "" {
		paymeBaseURL = "https://test.paycom.uz" // Default to test environment if not specified
	}
	return &Handler{
		store:        storage.NewStorage(db),
		paymeKey:     paymeKey,
		paymeBaseURL: paymeBaseURL,
	}
}

// CreatePaymeTransaction generates a payme transaction URL for the given user and amount.
func (h *Handler) CreatePaymeTransaction(userID, amount int, paymeMerchantID, returnURL string) (string, error) {
	payme := &models.Payme{
		ID:         fmt.Sprintf("%d-%d", userID, time.Now().Unix()),
		UserID:     userID,
		Amount:     amount * 100, // Convert from UZS to tiyin
		State:      1,
		CreateTime: time.Now().UTC(),
	}

	params := fmt.Sprintf("m=%s;ac.order_id=%s;a=%d;c=%s", paymeMerchantID, payme.ID, payme.Amount, returnURL)
	encoded := base64.StdEncoding.EncodeToString([]byte(params))
	return fmt.Sprintf("%s/%s", h.paymeBaseURL, encoded), nil
}

// HandlePaymeWebhook processes incoming payme webhook requests.
func (h *Handler) HandlePaymeWebhook(w http.ResponseWriter, r *http.Request) {
	var reqBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		fmt.Println("Error decoding request:", err)
		writeErrorResponse(w, -32504, getErrorMessageMap(-32504))
		return
	}

	// Authentication check
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Basic ") {
		writeUnauthorizedResponse(w, getErrorMessageMap(-32504))
		return
	}

	decodedAuth, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
	if err != nil || len(strings.SplitN(string(decodedAuth), ":", 2)) != 2 || strings.SplitN(string(decodedAuth), ":", 2)[1] != h.paymeKey {
		writeUnauthorizedResponse(w, getErrorMessageMap(-32504))
		return
	}

	method, ok := reqBody["method"].(string)
	if !ok {
		fmt.Println("Method not found in request")
		writeErrorResponse(w, -32504, getErrorMessageMap(-32504))
		return
	}

	switch method {
	case "CheckPerformTransaction":
		h.handleCheckPerformTransaction(w, reqBody)
	case "CreateTransaction":
		h.handleCreateTransaction(w, reqBody)
	case "PerformTransaction":
		h.handlePerformTransaction(w, reqBody)
	case "CancelTransaction":
		h.handleCancelTransaction(w, reqBody)
	case "CheckTransaction":
		h.handleCheckTransaction(w, reqBody)
	default:
		writeErrorResponse(w, -32504, getErrorMessageMap(-32504))
	}
}

func (h *Handler) handleCheckPerformTransaction(w http.ResponseWriter, req map[string]interface{}) {
	params, ok := req["params"].(map[string]interface{})
	if !ok {
		fmt.Println("Params not found")
		writeErrorResponse(w, -32504, getErrorMessageMap(-32504))
		return
	}

	amount, ok := params["amount"].(float64)
	if !ok || amount <= 0 {
		fmt.Println("Invalid amount")
		writeErrorResponse(w, -31001, getErrorMessageMap(-31001))
		return
	}

	account, ok := params["account"].(map[string]interface{})
	if !ok {
		fmt.Println("Account not found")
		writeErrorResponse(w, -31050, getErrorMessageMap(-31050))
		return
	}

	orderID, ok := account["order_id"].(string)
	if !ok {
		fmt.Println("Order ID not found")
		writeErrorResponse(w, -31050, getErrorMessageMap(-31050))
		return
	}

	allow, err := h.store.Payme().CheckPayme(context.Background(), orderID, int(amount))
	if err != nil || !allow {
		fmt.Println("Transaction check failed:", err)
		writeErrorResponse(w, -31050, getErrorMessageMap(-31050))
		return
	}

	writeSuccessResponse(w, map[string]interface{}{"allow": true})
}

func (h *Handler) handleCreateTransaction(w http.ResponseWriter, req map[string]interface{}) {
	params, ok := req["params"].(map[string]interface{})
	if !ok {
		fmt.Println("Params not found")
		writeErrorResponse(w, -32504, getErrorMessageMap(-32504))
		return
	}

	amount, ok := params["amount"].(float64)
	if !ok || amount <= 0 {
		writeErrorResponse(w, -31001, getErrorMessageMap(-31001))
		return
	}

	account, ok := params["account"].(map[string]interface{})
	if !ok {
		fmt.Println("Account not found")
		writeErrorResponse(w, -31050, getErrorMessageMap(-31050))
		return
	}

	orderID, ok := account["order_id"].(string)
	if !ok {
		fmt.Println("Order_id not found")
		writeErrorResponse(w, -31050, getErrorMessageMap(-31050))
		return
	}

	paymeID, ok := params["id"].(string)
	if !ok {
		fmt.Println("payme ID not found")
		writeErrorResponse(w, -32504, getErrorMessageMap(-32504))
		return
	}

	payment, err := h.store.Payme().GetPayme(context.Background(), orderID)
	if payment.PaymeTransactionID != "" && payment.PaymeTransactionID != paymeID {
		writeErrorResponse(w, -31099, getErrorMessageMap(-31099))
		return
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		fmt.Println("Error fetching payme transaction:", err)
		writeErrorResponse(w, -31008, getErrorMessageMap(-31008))
		return
	}

	payment, err = h.store.Payme().GetPayme(context.Background(), orderID)
	if err != nil {
		fmt.Println("Transaction not found:", err)
		writeErrorResponse(w, -31050, getErrorMessageMap(-31050))
		return
	}

	if payment.State != 1 {
		fmt.Println("Invalid transaction state")
		writeErrorResponse(w, -31050, getErrorMessageMap(-31050))
		return
	}

	if int(amount) != payment.Amount {
		writeErrorResponse(w, -31001, getErrorMessageMap(-31001))
		return
	}

	currentTime := time.Now().UTC().UnixMilli()
	createTime := payment.CreateTime.UnixMilli()

	if createTime > currentTime+1000 {
		writeErrorResponse(w, -31008, getErrorMessageMap(-31008))
		return
	}

	writeSuccessResponse(w, map[string]interface{}{
		"create_time": createTime,
		"transaction": paymeID,
		"state":       1,
	})
}

func (h *Handler) handlePerformTransaction(w http.ResponseWriter, req map[string]interface{}) {
	params, ok := req["params"].(map[string]interface{})
	if !ok {
		fmt.Println("Params not found")
		writeErrorResponse(w, -32504, getErrorMessageMap(-32504))
		return
	}

	paymeID, ok := params["id"].(string)
	if !ok {
		fmt.Println("payme ID not found")
		writeErrorResponse(w, -31050, getErrorMessageMap(-31050))
		return
	}

	payment, err := h.store.Payme().GetPayme(context.Background(), paymeID)
	if err != nil {
		fmt.Println("Transaction not found:", err)
		if err == sql.ErrNoRows {
			writeErrorResponse(w, -31003, getErrorMessageMap(-31003))
		} else {
			writeErrorResponse(w, -31008, getErrorMessageMap(-31008))
		}
		return
	}

	if payment.State != 1 {
		performTime := int64(0)
		if payment.PerformTime != nil {
			performTime = payment.PerformTime.UnixMilli()
		}
		writeSuccessResponse(w, map[string]interface{}{
			"transaction":  paymeID,
			"perform_time": performTime,
			"state":        payment.State,
		})
		return
	}

	performTime := time.Now().UTC()

	payment.State = 2
	payment.PerformTime = &performTime
	err = h.store.Payme().UpdatePayme(context.Background(), payment)
	if err != nil {
		fmt.Println("Error updating transaction:", err)
		writeErrorResponse(w, -31008, getErrorMessageMap(-31008))
		return
	}

	writeSuccessResponse(w, map[string]interface{}{
		"transaction":  paymeID,
		"perform_time": performTime.UnixMilli(),
		"state":        2,
	})
}

func (h *Handler) handleCancelTransaction(w http.ResponseWriter, req map[string]interface{}) {
	params, ok := req["params"].(map[string]interface{})
	if !ok {
		fmt.Println("Params not found")
		writeErrorResponse(w, -32504, getErrorMessageMap(-32504))
		return
	}

	paymeID, ok := params["id"].(string)
	if !ok {
		fmt.Println("payme ID not found")
		writeErrorResponse(w, -31050, getErrorMessageMap(-31050))
		return
	}

	payment, err := h.store.Payme().GetPayme(context.Background(), paymeID)
	if err != nil {
		fmt.Println("Transaction not found:", err)
		if err == sql.ErrNoRows {
			writeErrorResponse(w, -31003, getErrorMessageMap(-31003))
		} else {
			writeErrorResponse(w, -31008, getErrorMessageMap(-31008))
		}
		return
	}

	cancelTime := time.Now().UTC()

	if payment.State == -1 || payment.State == -2 {
		cancelTimeValue := int64(0)
		if payment.CancelTime != nil {
			cancelTimeValue = payment.CancelTime.UnixMilli()
		}
		writeSuccessResponse(w, map[string]interface{}{
			"transaction": paymeID,
			"cancel_time": cancelTimeValue,
			"state":       payment.State,
		})
		return
	}

	newState := -1
	if payment.State == 2 {
		newState = -2
	}

	payment.State = newState
	payment.CancelTime = &cancelTime
	err = h.store.Payme().UpdatePayme(context.Background(), payment)
	if err != nil {
		fmt.Println("Error updating transaction:", err)
		writeErrorResponse(w, -31008, getErrorMessageMap(-31008))
		return
	}

	writeSuccessResponse(w, map[string]interface{}{
		"transaction": paymeID,
		"cancel_time": cancelTime.UnixMilli(),
		"state":       newState,
	})
}

func (h *Handler) handleCheckTransaction(w http.ResponseWriter, req map[string]interface{}) {
	params, ok := req["params"].(map[string]interface{})
	if !ok {
		fmt.Println("Params not found")
		writeErrorResponse(w, -32504, getErrorMessageMap(-32504))
		return
	}

	paymeID, ok := params["id"].(string)
	if !ok {
		fmt.Println("payme ID not found")
		writeErrorResponse(w, -31050, getErrorMessageMap(-31050))
		return
	}

	payment, err := h.store.Payme().GetPayme(context.Background(), paymeID)
	if err != nil {
		fmt.Println("Transaction not found:", err)
		writeErrorResponse(w, -31003, getErrorMessageMap(-31003))
		return
	}

	var reason interface{}
	if payment.State == -1 {
		reason = 3
	} else if payment.State == -2 {
		reason = 5
	} else {
		reason = nil
	}

	performTime := int64(0)
	if payment.PerformTime != nil {
		performTime = payment.PerformTime.UnixMilli()
	}

	cancelTime := int64(0)
	if payment.CancelTime != nil {
		cancelTime = payment.CancelTime.UnixMilli()
	}

	writeSuccessResponse(w, map[string]interface{}{
		"create_time":  payment.CreateTime.UnixMilli(),
		"perform_time": performTime,
		"cancel_time":  cancelTime,
		"transaction":  paymeID,
		"state":        payment.State,
		"reason":       reason,
	})
}

func writeErrorResponse(w http.ResponseWriter, code int, message interface{}) {
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Println("Error encoding response:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func writeSuccessResponse(w http.ResponseWriter, result map[string]interface{}) {
	response := map[string]interface{}{
		"result": result,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Println("Error encoding response:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func writeUnauthorizedResponse(w http.ResponseWriter, message interface{}) {
	writeErrorResponse(w, -32504, message)
}

func getErrorMessageMap(code int) map[string]interface{} {
	return map[string]interface{}{
		"en": getErrorMessageEn(code),
		"ru": getErrorMessageRu(code),
		"uz": getErrorMessageUz(code),
	}
}

func getErrorMessageEn(code int) string {
	switch code {
	case -31001:
		return "Invalid amount"
	case -31003:
		return "Transaction not found"
	case -31008:
		return "Unable to perform this operation"
	case -31050:
		return "Invalid account or transaction not found"
	case -31099:
		return "Transaction already exists with different ID"
	case -32504:
		return "Invalid request format"
	default:
		return "Unknown error"
	}
}

func getErrorMessageRu(code int) string {
	switch code {
	case -31001:
		return "Неверная сумма"
	case -31003:
		return "Транзакция не найдена"
	case -31008:
		return "Невозможно выполнить данную операцию"
	case -31050:
		return "Неверный аккаунт или транзакция не найдена"
	case -31099:
		return "Транзакция уже существует с другим ID"
	case -32504:
		return "Неверный формат запроса"
	default:
		return "Неизвестная ошибка"
	}
}

func getErrorMessageUz(code int) string {
	switch code {
	case -31001:
		return "Noto‘g‘ri miqdor"
	case -31003:
		return "Tranzaksiya topilmadi"
	case -31008:
		return "Bu amalni bajarib bo‘lmaydi"
	case -31050:
		return "Noto‘g‘ri akkaunt yoki tranzaksiya topilmadi"
	case -31099:
		return "Tranzaksiya allaqachon boshqa ID bilan mavjud"
	case -32504:
		return "So‘rov formati noto‘g‘ri"
	default:
		return "Noma’lum xato"
	}
}
