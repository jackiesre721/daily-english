package handler

import (
	"daily-english/internal/model"
	"daily-english/internal/repository"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ImportURL fetches a web page, extracts article content, and saves it.
func (h *ImportHandler) ImportURL(c *gin.Context) {
	var input struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供 URL"})
		return
	}

	url := strings.TrimSpace(input.URL)
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL 必须以 http:// 或 https:// 开头"})
		return
	}

	// Fetch the page
	client := &http.Client{Timeout: 30 * time.Second}
	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法访问该网页: " + err.Error()})
		return
	}
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法访问该网页: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("网页返回状态码 %d", resp.StatusCode)})
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024)) // 2MB limit
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取网页内容失败"})
		return
	}

	// Parse and extract
	title, articleHTML, plainText := extractArticle(string(body))
	if plainText == "" && articleHTML == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法从网页提取正文内容"})
		return
	}

	// Use URL-based title if none found
	if title == "" {
		title = extractTitleFromURL(url)
	}

	contentHash := repository.ContentHash(url)
	existing, _ := h.importSourceRepo.FindByContentHash(contentHash)
	if existing != nil && existing.Status == "completed" {
		c.JSON(http.StatusConflict, gin.H{"error": "该网页已导入过", "source_id": existing.ID})
		return
	}

	sourceID := uuid.New().String()[:12]
	source := &model.ImportSource{
		ID:          sourceID,
		Name:        title,
		FileName:    url,
		ScenarioID:  "ai_imported",
		Status:      "completed",
		ContentHash: contentHash,
		CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
	}
	if err := h.importSourceRepo.Create(source); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存导入记录失败"})
		return
	}

	articleID := "art_" + sourceID
	article := &model.Article{
		ID:             articleID,
		ScenarioID:     "ai_imported",
		TitleZh:        title,
		TitleEn:        title,
		Type:           "web_import",
		Difficulty:     0,
		Source:         "web: " + url,
		ImportSourceID: sourceID,
		OriginalText:   plainText,
		HTMLContent:    articleHTML,
	}

	if _, err := h.articleRepo.Import([]model.Article{*article}); err != nil {
		source.Status = "failed"
		source.ErrorMessage = "save article: " + err.Error()
		h.importSourceRepo.Update(source) // best effort
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文章失败"})
		return
	}

	source.ArticleCount = 1
	if err := h.importSourceRepo.Update(source); err != nil {
		fmt.Printf("warning: failed to update source %s: %v\n", sourceID, err)
	}
	h.ensureScenario()

	c.JSON(http.StatusOK, gin.H{"status": "ok", "article": article, "source": source})
}

// extractArticle parses HTML and returns (title, articleHTML, plainText).
func extractArticle(htmlStr string) (string, string, string) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", "", ""
	}

	var title string
	var body *html.Node

	// Find <title> and <body>
	var findNodes func(*html.Node)
	findNodes = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "title" && title == "" {
				title = textContent(n)
			}
			if n.Data == "body" && body == nil {
				body = n
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findNodes(c)
		}
	}
	findNodes(doc)

	if body == nil {
		return title, "", ""
	}

	// Find the best content root: <article>, <main>, or [role=main]
	contentRoot := findContentRoot(body)
	if contentRoot == nil {
		contentRoot = body
	}

	// Clean and extract
	cleanNode(contentRoot, 0)

	articleHTML := renderNode(contentRoot)
	plainText := textContent(contentRoot)

	return title, articleHTML, plainText
}

// Tags to remove entirely (including children)
var removeTags = map[string]bool{
	"script": true, "style": true, "noscript": true, "iframe": true,
	"nav": true, "header": true, "footer": true, "aside": true,
	"form": true, "svg": true, "button": true, "input": true,
	"textarea": true, "select": true, "template": true,
}

// Tags to keep as content
var contentTags = map[string]bool{
	"p": true, "br": true, "hr": true, "img": true, "image": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"ul": true, "ol": true, "li": true,
	"blockquote": true, "pre": true, "code": true,
	"figure": true, "figcaption": true, "picture": true, "source": true,
	"em": true, "strong": true, "b": true, "i": true, "u": true, "s": true,
	"span": true, "a": true, "div": true, "section": true, "article": true,
	"main": true, "table": true, "thead": true, "tbody": true, "tr": true,
	"td": true, "th": true, "mark": true, "small": true, "sub": true, "sup": true,
	"time": true, "dl": true, "dt": true, "dd": true,
	"details": true, "summary": true, "cite": true, "q": true,
}

// findContentRoot finds the best content container
func findContentRoot(body *html.Node) *html.Node {
	// Prefer <article>
	var article, main, roleMain *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "article" && article == nil {
				article = n
			}
			if n.Data == "main" && main == nil {
				main = n
			}
			for _, attr := range n.Attr {
				if attr.Key == "role" && attr.Val == "main" && roleMain == nil {
					roleMain = n
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(body)

	if article != nil {
		return article
	}
	if main != nil {
		return main
	}
	if roleMain != nil {
		return roleMain
	}
	return body
}

// cleanNode removes unwanted tags and cleans attributes
func cleanNode(n *html.Node, depth int) {
	if depth > 100 {
		return
	}

	var next *html.Node
	for c := n.FirstChild; c != nil; c = next {
		next = c.NextSibling

		if c.Type == html.ElementNode {
			// Remove unwanted tags
			if removeTags[c.Data] {
				n.RemoveChild(c)
				continue
			}
			// Remove if not a known content tag
			if !contentTags[c.Data] {
				// Unwrap: move children up
				var childNext *html.Node
				for gc := c.FirstChild; gc != nil; gc = childNext {
					childNext = gc.NextSibling
					c.RemoveChild(gc)
					n.InsertBefore(gc, c)
				}
				n.RemoveChild(c)
				continue
			}
			// Clean attributes — keep only essential ones
			cleanAttributes(c)
			// Recurse
			cleanNode(c, depth+1)
		} else if c.Type == html.CommentNode {
			n.RemoveChild(c)
		}
	}
}

// cleanAttributes removes all attributes except essential ones
func cleanAttributes(n *html.Node) {
	var keep []html.Attribute
	for _, attr := range n.Attr {
		switch n.Data {
		case "img":
			if attr.Key == "src" || attr.Key == "alt" || attr.Key == "width" || attr.Key == "height" {
				keep = append(keep, attr)
			}
			// Fix relative URLs - keep as is, browser will handle
		case "a":
			// Remove href from links but keep text
			// Don't keep any attributes for <a>
		case "source":
			if attr.Key == "srcset" || attr.Key == "src" || attr.Key == "type" || attr.Key == "media" {
				keep = append(keep, attr)
			}
		}
	}
	n.Attr = keep
}

// renderNode renders an HTML node back to string
func renderNode(n *html.Node) string {
	var b strings.Builder
	var render func(*html.Node)
	render = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			return
		}
		if n.Type == html.ElementNode {
			// Self-closing tags
			if n.Data == "img" || n.Data == "br" || n.Data == "hr" {
				b.WriteString("<" + n.Data)
				for _, attr := range n.Attr {
					b.WriteString(fmt.Sprintf(` %s="%s"`, attr.Key, html.EscapeString(attr.Val)))
				}
				if n.Data == "img" {
					b.WriteString(">")
				} else {
					b.WriteString(">")
				}
				return
			}

			b.WriteString("<" + n.Data)
			for _, attr := range n.Attr {
				b.WriteString(fmt.Sprintf(` %s="%s"`, attr.Key, html.EscapeString(attr.Val)))
			}
			b.WriteString(">")

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				render(c)
			}

			b.WriteString("</" + n.Data + ">")
		}
	}
	render(n)
	return b.String()
}

// textContent extracts plain text from a node
func textContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
			// Add space after block-level elements
			if c.Type == html.ElementNode {
				switch c.Data {
				case "p", "div", "br", "h1", "h2", "h3", "h4", "h5", "h6",
					"li", "blockquote", "tr", "hr", "figcaption", "figure":
					b.WriteString("\n")
				}
			}
		}
	}
	walk(n)
	return strings.TrimSpace(b.String())
}

// extractTitleFromURL derives a title from the URL path
func extractTitleFromURL(rawURL string) string {
	parts := strings.Split(rawURL, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		p := strings.TrimSpace(parts[i])
		if p != "" && !strings.HasPrefix(p, "?") && !strings.HasPrefix(p, "#") && p != "http:" && p != "https:" {
			// Remove trailing query/hash
			if idx := strings.IndexAny(p, "?#"); idx >= 0 {
				p = p[:idx]
			}
			// Replace dashes/underscores with spaces
			p = strings.ReplaceAll(p, "-", " ")
			p = strings.ReplaceAll(p, "_", " ")
			return p
		}
	}
	return "Web Article"
}
