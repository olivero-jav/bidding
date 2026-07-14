package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"bidding/internal/auction/app"
	"bidding/internal/auction/domain"
)

// Handler adapts HTTP requests to the auction application service.
type Handler struct {
	svc *app.Service
}

func NewHandler(svc *app.Service) *Handler {
	return &Handler{svc: svc}
}

// Routes wires the auction endpoints onto a ServeMux (Go 1.22+ method routing).
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auctions", h.create)
	mux.HandleFunc("GET /api/auctions", h.list)
	mux.HandleFunc("GET /api/auctions/{id}", h.get)
	mux.HandleFunc("POST /api/auctions/{id}/bids", h.placeBid)
	return withCORS(mux)
}

// withCORS allows the Angular dev server to call the API cross-origin. Wide open
// for local dev; tighten to a specific origin before any real deployment.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- DTOs ---

type createAuctionRequest struct {
	SellerID     string    `json:"sellerId"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Category     string    `json:"category"`
	StartPrice   int64     `json:"startPrice"`
	MinIncrement int64     `json:"minIncrement"`
	Cap          *int64    `json:"cap"`
	EndsAt       time.Time `json:"endsAt"`
}

type auctionResponse struct {
	ID               string    `json:"id"`
	SellerID         string    `json:"sellerId"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Category         string    `json:"category"`
	StartPrice       int64     `json:"startPrice"`
	MinIncrement     int64     `json:"minIncrement"`
	Cap              *int64    `json:"cap"`
	EndsAt           time.Time `json:"endsAt"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	HighestBidAmount *int64    `json:"highestBidAmount"`
	HighestBidderID  *string   `json:"highestBidderId"`
}

func toResponse(a *domain.Auction) auctionResponse {
	var cap *int64
	if a.Cap != nil {
		v := int64(*a.Cap)
		cap = &v
	}
	var highestAmount *int64
	if a.HighestBidAmount != nil {
		v := int64(*a.HighestBidAmount)
		highestAmount = &v
	}
	var highestBidder *string
	if a.HighestBidderID != nil {
		v := a.HighestBidderID.String()
		highestBidder = &v
	}
	return auctionResponse{
		ID:               a.ID.String(),
		SellerID:         a.SellerID.String(),
		Title:            a.Title,
		Description:      a.Description,
		Category:         a.Category,
		StartPrice:       int64(a.StartPrice),
		MinIncrement:     int64(a.MinIncrement),
		Cap:              cap,
		EndsAt:           a.EndsAt,
		Status:           string(a.Status),
		CreatedAt:        a.CreatedAt,
		HighestBidAmount: highestAmount,
		HighestBidderID:  highestBidder,
	}
}

type placeBidRequest struct {
	BidderID string `json:"bidderId"`
	Amount   int64  `json:"amount"`
}

type bidResponse struct {
	ID        string    `json:"id"`
	AuctionID string    `json:"auctionId"`
	BidderID  string    `json:"bidderId"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"createdAt"`
}

// placeBidResponse returns both the recorded bid and the auction's new state so
// the client can render the winning bid and the updated current price at once.
type placeBidResponse struct {
	Bid     bidResponse     `json:"bid"`
	Auction auctionResponse `json:"auction"`
}

func toBidResponse(b *domain.Bid) bidResponse {
	return bidResponse{
		ID:        b.ID.String(),
		AuctionID: b.AuctionID.String(),
		BidderID:  b.BidderID.String(),
		Amount:    int64(b.Amount),
		CreatedAt: b.CreatedAt,
	}
}

// --- Handlers ---

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createAuctionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	sellerID, err := uuid.Parse(req.SellerID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "sellerId must be a valid UUID")
		return
	}

	in := app.CreateAuctionInput{
		SellerID:     sellerID,
		Title:        req.Title,
		Description:  req.Description,
		Category:     req.Category,
		StartPrice:   domain.Money(req.StartPrice),
		MinIncrement: domain.Money(req.MinIncrement),
		EndsAt:       req.EndsAt,
	}
	if req.Cap != nil {
		c := domain.Money(*req.Cap)
		in.Cap = &c
	}

	a, err := h.svc.CreateAuction(r.Context(), in)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(a))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	auctions, err := h.svc.ListAuctions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list auctions")
		return
	}
	out := make([]auctionResponse, 0, len(auctions))
	for _, a := range auctions {
		out = append(out, toResponse(a))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a valid UUID")
		return
	}
	a, err := h.svc.GetAuction(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(a))
}

func (h *Handler) placeBid(w http.ResponseWriter, r *http.Request) {
	auctionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a valid UUID")
		return
	}

	var req placeBidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	bidderID, err := uuid.Parse(req.BidderID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bidderId must be a valid UUID")
		return
	}

	result, err := h.svc.PlaceBid(r.Context(), app.PlaceBidInput{
		AuctionID: auctionID,
		BidderID:  bidderID,
		Amount:    domain.Money(req.Amount),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, placeBidResponse{
		Bid:     toBidResponse(result.Bid),
		Auction: toResponse(result.Auction),
	})
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeDomainError maps domain errors to HTTP status codes: malformed input and
// publication-invariant violations are 400, a missing auction is 404, a bid that
// conflicts with the auction's current state is 409, a lock the server could not
// grab in time is 503 (retryable), anything else 500.
func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrAuctionNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrBiddingUnavailable):
		writeError(w, http.StatusServiceUnavailable, err.Error())
	case isBidConflict(err):
		writeError(w, http.StatusConflict, err.Error())
	case isValidationError(err):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, context.Canceled):
		writeError(w, http.StatusRequestTimeout, "request canceled")
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// isValidationError reports malformed-input errors (400): the request itself is
// invalid regardless of any auction state.
func isValidationError(err error) bool {
	for _, ve := range []error{
		domain.ErrEmptyTitle,
		domain.ErrNonPositiveIncrement,
		domain.ErrNegativeStartPrice,
		domain.ErrEndsInPast,
		domain.ErrCapNotAboveStart,
		domain.ErrNonPositiveAmount,
	} {
		if errors.Is(err, ve) {
			return true
		}
	}
	return false
}

// isBidConflict reports bid rejections that depend on the auction's current
// state (409): valid request, but it lost the race or arrived too late.
func isBidConflict(err error) bool {
	for _, ce := range []error{
		domain.ErrAuctionNotActive,
		domain.ErrAuctionEnded,
		domain.ErrBidBelowMinimum,
	} {
		if errors.Is(err, ce) {
			return true
		}
	}
	return false
}
