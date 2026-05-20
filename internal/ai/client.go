package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"daily-english/internal/model"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Client struct {
	config     model.AIConfig
	httpClient *http.Client
}

func NewClient(config model.AIConfig) *Client {
	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: 300 * time.Second},
	}
}

// ── Multi-step AI pipeline ──

const maxChunkSize = 8000 // chars per chunk for AI processing

// splitIntoChunks splits text into chunks at paragraph boundaries.
func splitIntoChunks(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	lines := strings.Split(text, "\n")
	var current strings.Builder

	for _, line := range lines {
		if current.Len()+len(line)+1 > maxLen && current.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(line)
	}
	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}
	return chunks
}

// Step 1: Format a single chunk — returns formatted text, segments, and translation.
type formatChunkResult struct {
	OriginalText string           `json:"original_text"`
	Segments     []model.Segment  `json:"segments"`
}

func (c *Client) formatChunk(chunkText string, contentType string) (*formatChunkResult, error) {
	prompt := `You are an English learning content parser. Format the given text and return a JSON object:

{
  "original_text": "<formatted text>",
  "segments": [
    {"type": "dialogue", "speaker": "Name", "text": "words", "translation": "Chinese translation"},
    {"type": "narration", "text": "text", "translation": "Chinese translation"},
    {"type": "action", "text": "text", "translation": "Chinese translation"}
  ]
}

Content type: ` + contentType + `

FOR DIALOGUE:
- Format each spoken line: Speaker: "their words"
- Narration on separate lines without speaker prefix
- Blank line between scene changes
- Segments: "dialogue" (with speaker), "narration", "action"

FOR PASSAGE:
- Add paragraph breaks where topics change
- Segments: "narration" for each paragraph

CRITICAL:
- original_text MUST contain COMPLETE input text, NO omissions
- segments MUST cover ALL text
- Each segment MUST have a "translation" field with accurate Chinese translation of that segment's text
- Output ONLY valid JSON`

	content, err := c.callWithRetry(prompt, chunkText)
	if err != nil {
		return nil, err
	}
	content = stripCodeFence(content)

	var result formatChunkResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON: %v\nraw: %s", err, truncate(content, 300))
	}
	return &result, nil
}

// Step 1 full: Format article — detect type, then format (chunked if needed).
func (c *Client) FormatArticle(text string) (*model.Article, error) {
	article := &model.Article{}

	// Use first non-empty line as title
	firstLine := ""
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			firstLine = trimmed
			break
		}
	}
	if firstLine == "" {
		firstLine = "Article"
	}

	// First, detect content type with a small call
	detectPrompt := `Analyze this text and return ONLY a JSON object: {"type":"dialogue" or "passage","title_en":"` + firstLine + `","title_zh":"<Chinese translation of the title>","difficulty":1,"context_note":"<brief context in Chinese>"}
Rules: Multiple speakers conversing = "dialogue". Continuous essay = "passage". difficulty: 1 simple, 2 moderate, 3 advanced. Use the given title_en exactly as-is, do NOT change it. Only provide a Chinese translation for title_zh.`

	// Send first 1500 chars for type detection
	preview := text
	if len(preview) > 1500 {
		preview = preview[:1500]
	}

	fmt.Printf("[AI Step 1a] Detect content type\n")
	detectContent, err := c.callWithRetry(detectPrompt, preview)
	if err != nil {
		return nil, fmt.Errorf("type detection failed: %v", err)
	}
	detectContent = stripCodeFence(detectContent)

	type detectResult struct {
		Type        string `json:"type"`
		TitleEn     string `json:"title_en"`
		TitleZh     string `json:"title_zh"`
		Difficulty  int    `json:"difficulty"`
		ContextNote string `json:"context_note"`
	}
	var detected detectResult
	if err := json.Unmarshal([]byte(detectContent), &detected); err != nil {
		// Fallback
		article.Type = "passage"
		article.TitleEn = firstLine
		article.TitleZh = ""
	} else {
		article.Type = detected.Type
		article.TitleEn = firstLine // always use original first line
		article.TitleZh = detected.TitleZh
		article.Difficulty = detected.Difficulty
		article.ContextNote = detected.ContextNote
	}

	// Format text — split into chunks if needed
	chunks := splitIntoChunks(text, maxChunkSize)
	fmt.Printf("[AI Step 1b] Format %d chunk(s)\n", len(chunks))

	var allSegments []model.Segment
	var allOriginal strings.Builder

	for i, chunk := range chunks {
		fmt.Printf("[AI Step 1b] Formatting chunk %d/%d (%d chars)\n", i+1, len(chunks), len(chunk))
		result, err := c.formatChunk(chunk, article.Type)
		if err != nil {
			fmt.Printf("[AI] Warning: chunk %d failed: %v\n", i+1, err)
			// Fallback: use raw chunk
			allOriginal.WriteString(chunk)
			allOriginal.WriteString("\n\n")
			continue
		}

		if allOriginal.Len() > 0 {
			allOriginal.WriteString("\n\n")
		}
		allOriginal.WriteString(result.OriginalText)
		allSegments = append(allSegments, result.Segments...)
	}

	article.OriginalText = strings.TrimSpace(allOriginal.String())
	article.Segments = allSegments

	if article.OriginalText == "" {
		return nil, fmt.Errorf("formatting produced empty text")
	}
	return article, nil
}

// Step 2: Extract vocabulary from the formatted article (chunked if needed).
func (c *Client) ExtractWords(article *model.Article) ([]model.Word, error) {
	prompt := `You are an English vocabulary expert for Chinese learners. Given an English text, extract ALL meaningful vocabulary words. Return a JSON array:

[
  {
    "word_id": "w1",
    "text_in_article": "<exact word>",
    "english": "<the word>",
    "phonetic": "<IPA>",
    "chinese": "<Chinese meaning>",
    "explanation": "<usage explanation>",
    "memory_tip": "<mnemonic in Chinese>",
    "related_terms": ["word=meaning"],
    "recommended_for_study": true
  }
]

Rules:
- Extract ALL meaningful vocabulary words (skip only basic sight words like a/the/is/it/he/she)
- recommended_for_study: set true for words a Chinese learner should actively memorize (medium difficulty, high utility, common in daily life). Set false for words that are either very easy or very rare/specialized. Aim for 30-50% true.
- All Chinese in Simplified Chinese
- Output ONLY valid JSON array`

	chunks := splitIntoChunks(article.OriginalText, maxChunkSize)
	fmt.Printf("[AI Step 2] Extract words from %d chunk(s)\n", len(chunks))

	var allWords []model.Word
	seen := make(map[string]bool)
	wordCounter := 0

	for i, chunk := range chunks {
		fmt.Printf("[AI Step 2] Processing chunk %d/%d\n", i+1, len(chunks))
		content, err := c.callWithRetry(prompt, chunk)
		if err != nil {
			fmt.Printf("[AI] Warning: word extraction chunk %d failed: %v\n", i+1, err)
			continue
		}
		content = stripCodeFence(content)

		var words []model.Word
		if err := json.Unmarshal([]byte(content), &words); err != nil {
			fmt.Printf("[AI] Warning: chunk %d JSON parse failed: %v\n", i+1, err)
			continue
		}

		for _, w := range words {
			key := strings.ToLower(strings.TrimSpace(w.English))
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			wordCounter++
			w.WordID = fmt.Sprintf("w%d", wordCounter)
			allWords = append(allWords, w)
		}
	}

	return allWords, nil
}

// Step 3: Generate quizzes based on article and vocabulary.
func (c *Client) GenerateQuizzes(article *model.Article, words []model.Word) ([]model.Quiz, error) {
	prompt := `You are an English learning quiz generator. Given vocabulary words, generate exactly 3 quizzes. Return a JSON array:

[
  {"type": "fill_blank", "question": "<sentence with _____>", "hint": "<hint>", "options": ["correct", "wrong1", "wrong2", "wrong3"], "answer": 0, "target_word_id": "w1"},
  {"type": "understand", "question": "<comprehension question in Chinese>", "options": ["A", "B", "C", "D"], "answer": 0, "target_word_id": "w1"},
  {"type": "reorder", "words": ["w1", "w2", "w3"], "answer": "w1 w2 w3", "target_word_ids": ["w1"]}
]

Rules:
- Exactly 3 quizzes: 1 fill_blank, 1 understand, 1 reorder
- Use the provided word_ids
- Create fill_blank questions using sentences from the article context
- All Chinese content in Simplified Chinese
- Output ONLY valid JSON array`

	// Use truncated article text + full word list for quiz context
	articlePreview := article.OriginalText
	if len(articlePreview) > 2000 {
		articlePreview = articlePreview[:2000] + "..."
	}
	wordsJSON, _ := json.Marshal(words)
	input := fmt.Sprintf("Article context:\n%s\n\nVocabulary:\n%s", articlePreview, string(wordsJSON))

	fmt.Printf("[AI Step 3] Generate quizzes\n")

	content, err := c.callWithRetry(prompt, input)
	if err != nil {
		return nil, fmt.Errorf("step 3 quizzes failed: %v", err)
	}

	content = stripCodeFence(content)

	var quizzes []model.Quiz
	if err := json.Unmarshal([]byte(content), &quizzes); err != nil {
		return nil, fmt.Errorf("step 3 invalid JSON: %v\nraw: %s", err, truncate(content, 300))
	}
	return quizzes, nil
}

// OptimizeArticle runs the full 3-step AI pipeline on raw text.
// The progressFn callback is called after each step with step name.
func (c *Client) OptimizeArticle(text string, progressFn func(step string)) (*model.Article, error) {
	if c.config.BaseURL == "" || c.config.AuthToken == "" {
		return nil, fmt.Errorf("AI not configured")
	}

	// Step 1: Format
	if progressFn != nil {
		progressFn("format")
	}
	article, err := c.FormatArticle(text)
	if err != nil {
		return nil, err
	}

	// Step 2: Words
	if progressFn != nil {
		progressFn("words")
	}
	words, err := c.ExtractWords(article)
	if err != nil {
		// Words failed — still return article without words
		fmt.Printf("[AI] Warning: word extraction failed: %v\n", err)
	} else {
		article.Words = words
	}

	// Step 3: Quizzes
	if progressFn != nil {
		progressFn("quizzes")
	}
	if len(article.Words) > 0 {
		quizzes, err := c.GenerateQuizzes(article, article.Words)
		if err != nil {
			fmt.Printf("[AI] Warning: quiz generation failed: %v\n", err)
		} else {
			article.Quizzes = quizzes
		}
	}

	if progressFn != nil {
		progressFn("done")
	}
	return article, nil
}

// ── Shared call with retry on content safety ──

func (c *Client) callWithRetry(systemPrompt, userText string) (string, error) {
	content, err := c.call(systemPrompt, userText)
	if err != nil {
		if strings.Contains(err.Error(), `"code":"1301"`) || strings.Contains(err.Error(), "不安全或敏感") {
			fmt.Println("[AI] Content safety triggered, retrying...")
			content, err = c.call(systemPrompt, userText)
			if err != nil {
				return "", fmt.Errorf("content safety filter blocked: %v", err)
			}
		} else {
			return "", err
		}
	}
	return content, nil
}

// ── Word lookup (for inline vocabulary) ──

type WordLookupResult struct {
	Word        string `json:"word"`
	Phonetic    string `json:"phonetic"`
	Chinese     string `json:"chinese"`
	Explanation string `json:"explanation"`
}

func (c *Client) LookupWord(word string) (*WordLookupResult, error) {
	if c.config.BaseURL == "" || c.config.AuthToken == "" {
		return nil, fmt.Errorf("AI not configured")
	}
	prompt := `You are an English learning assistant for Chinese learners. Given an English word, return a JSON object with:
{"word":"<the word>","phonetic":"<IPA pronunciation>","chinese":"<Chinese meaning in Simplified Chinese>","explanation":"<brief usage explanation in Chinese, 1-2 sentences>"}
Output ONLY valid JSON. No markdown, no explanation.`

	content, err := c.call(prompt, word)
	if err != nil {
		return nil, err
	}
	content = stripCodeFence(content)

	var result WordLookupResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON from AI: %v", err)
	}
	return &result, nil
}

// ── Sentence translation ──

type SentenceTranslationResult struct {
	Translation string `json:"translation"`
}

func (c *Client) TranslateSentence(sentence string) (*SentenceTranslationResult, error) {
	if c.config.BaseURL == "" || c.config.AuthToken == "" {
		return nil, fmt.Errorf("AI not configured")
	}
	prompt := `You are a professional English-to-Chinese translator. Translate the given English sentence into natural, accurate Simplified Chinese. Return ONLY a JSON object:
{"translation":"<Chinese translation>"}
No markdown, no explanation, just the JSON.`

	content, err := c.call(prompt, sentence)
	if err != nil {
		return nil, err
	}
	content = stripCodeFence(content)

	var result SentenceTranslationResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON from AI: %v", err)
	}
	return &result, nil
}

// ── AI text analysis ──

type AnalyzedWord struct {
	Word        string `json:"word"`
	Phonetic    string `json:"phonetic"`
	Chinese     string `json:"chinese"`
	Explanation string `json:"explanation"`
}

type AnalyzedPhrase struct {
	Phrase      string `json:"phrase"`
	Chinese     string `json:"chinese"`
	Explanation string `json:"explanation"`
	Category    string `json:"category"` // phrase / sentence / expression
}

type AnalyzeResult struct {
	Summary  string          `json:"summary"`
	Translation string      `json:"translation"`
	Words    []AnalyzedWord  `json:"words"`
	Phrases  []AnalyzedPhrase `json:"phrases"`
}

func (c *Client) AnalyzeText(text string) (*AnalyzeResult, error) {
	if c.config.BaseURL == "" || c.config.AuthToken == "" {
		return nil, fmt.Errorf("AI not configured")
	}

	prompt := `You are an expert English learning assistant. A Chinese learner has selected some English text. Analyze it and return a JSON object.

Rules:
- Use Simplified Chinese for all explanations
- Return ONLY valid JSON, no markdown, no code fences
- Do NOT use pipe character "|" in any field value, use commas or parentheses instead
- Do NOT use markdown formatting (no bold, no headers, no backticks) in any field value
`

	wordCount := len(strings.Fields(text))
	if wordCount <= 40 {
		prompt += `The text is a single sentence. Return JSON with:
- "translation": string — natural Chinese translation of the full sentence
- "words": array of {word, phonetic, chinese, explanation} — pick 3-5 important words with short explanation
- "phrases": array of {phrase, chinese, explanation, category} — pick any useful expressions or collocations (category: "phrase" or "expression")`
	} else {
		prompt += `The text is a paragraph (multiple sentences). Return JSON with:
- "summary": string — brief paragraph summary in Chinese (1-2 sentences)
- "translation": string — Chinese translation of the full text
- "words": array of {word, phonetic, chinese, explanation} — pick up to 8 important words
- "phrases": array of {phrase, chinese, explanation, category} — extract useful phrases and common expressions (category: "phrase", "sentence", or "expression"). Include 3-8 items.`
	}

	result, err := c.call(prompt, text)
	if err != nil {
		return nil, err
	}
	result = stripCodeFence(result)

	var analyzeResult AnalyzeResult
	json.Unmarshal([]byte(result), &analyzeResult)
	return &analyzeResult, nil
}

// ── HTTP layer (OpenAI / Anthropic) ──

func isAnthropic(baseURL string) bool {
	lower := strings.ToLower(baseURL)
	return strings.Contains(lower, "anthropic") ||
		strings.Contains(lower, "/messages") ||
		strings.Contains(lower, "claude")
}

func (c *Client) call(systemPrompt, userText string) (string, error) {
	if isAnthropic(c.config.BaseURL) {
		return c.callAnthropic(systemPrompt, userText)
	}
	return c.callOpenAI(systemPrompt, userText)
}

func (c *Client) callOpenAI(systemPrompt, userText string) (string, error) {
	url := buildOpenAIURL(c.config.BaseURL)
	modelName := c.modelName("gpt-4")

	reqBody := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userText},
		},
		"temperature": 0.3,
		"max_tokens":  8192,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.AuthToken)

	return c.doRequest(req, url, func(respBody []byte) (string, error) {
		var chatResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(respBody, &chatResp); err != nil {
			return "", fmt.Errorf("parse response: %v", err)
		}
		if len(chatResp.Choices) == 0 {
			return "", fmt.Errorf("no choices in response")
		}
		return chatResp.Choices[0].Message.Content, nil
	})
}

func (c *Client) callAnthropic(systemPrompt, userText string) (string, error) {
	url := buildAnthropicURL(c.config.BaseURL)
	modelName := c.modelName("claude-3-5-sonnet-20241022")

	maxTokens := 8192
	reqBody := map[string]interface{}{
		"model":      modelName,
		"max_tokens": maxTokens,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": userText},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.AuthToken)
	req.Header.Set("anthropic-version", "2023-06-01")

	return c.doRequest(req, url, func(respBody []byte) (string, error) {
		var resp struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return "", fmt.Errorf("parse response: %v", err)
		}
		for _, block := range resp.Content {
			if block.Type == "text" {
				return block.Text, nil
			}
		}
		if len(resp.Content) > 0 {
			return resp.Content[0].Text, nil
		}
		return "", fmt.Errorf("no text content in response")
	})
}

func (c *Client) doRequest(req *http.Request, url string, parseFn func([]byte) (string, error)) (string, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("AI request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI API returned %d (URL: %s): %s", resp.StatusCode, url, truncate(string(respBody), 500))
	}

	return parseFn(respBody)
}

func (c *Client) modelName(fallback string) string {
	if c.config.Model != "" {
		return c.config.Model
	}
	return fallback
}

func stripCodeFence(s string) string {
	re := regexp.MustCompile("(?s)^\\s*```(?:json)?\\s*\n?(.*?)\\s*```\\s*$")
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}
	return s
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

func buildOpenAIURL(rawURL string) string {
	u := strings.TrimRight(rawURL, "/")
	if strings.HasSuffix(u, "/chat/completions") {
		return u
	}
	if strings.HasSuffix(u, "/v1") {
		return u + "/chat/completions"
	}
	return u + "/v1/chat/completions"
}

func buildAnthropicURL(rawURL string) string {
	u := strings.TrimRight(rawURL, "/")
	if strings.HasSuffix(u, "/messages") {
		return u
	}
	if strings.HasSuffix(u, "/v1") {
		return u + "/messages"
	}
	return u + "/v1/messages"
}
