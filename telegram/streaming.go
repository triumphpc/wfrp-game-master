// Package telegram provides streaming message handling for long messages
package telegram

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Streamer handles sending long messages in chunks
type Streamer struct {
	bot         *Bot
	maxLength    int
	rateLimit    time.Duration
	mu           sync.Mutex
	queue        chan *streamJob
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// streamJob represents a streaming job
type streamJob struct {
	chatID  int64
	text     string
	replyTo  *int
	callback func(int, error)
}

// NewStreamer creates a new message streamer
func NewStreamer(bot *Bot) *Streamer {
	return &Streamer{
		bot:      bot,
		maxLength: 4096, // Telegram message limit
		rateLimit: 100 * time.Millisecond, // 10 messages per second
		queue:     make(chan *streamJob, 100),
		stopChan:  make(chan struct{}),
	}
}

// Start begins processing the streaming queue
func (s *Streamer) Start() {
	s.wg.Add(1)
	go s.processQueue()
}

// Stop gracefully stops the streamer
func (s *Streamer) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

// Stream sends a long message in chunks
func (s *Streamer) Stream(chatID int64, text string) error {
	resultChan := make(chan error, 1)
	job := &streamJob{
		chatID: chatID,
		text:    text,
		callback: func(part int, err error) {
			resultChan <- err
		},
	}

	select {
	case s.queue <- job:
	case <-time.After(5 * time.Second):
		return ErrQueueFull
	}

	return <-resultChan
}

// StreamReply sends a long reply in chunks
func (s *Streamer) StreamReply(messageID int, chatID int64, text string) error {
	resultChan := make(chan error, 1)
	job := &streamJob{
		chatID: chatID,
		text:    text,
		replyTo:  &messageID,
		callback: func(part int, err error) {
			resultChan <- err
		},
	}

	select {
	case s.queue <- job:
	case <-time.After(5 * time.Second):
		return ErrQueueFull
	}

	return <-resultChan
}

// processQueue handles streaming jobs
func (s *Streamer) processQueue() {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopChan:
			return
		case job, ok := <-s.queue:
			if !ok {
				return
			}
			s.processJob(job)
		}
	}
}

// processJob processes a single streaming job
func (s *Streamer) processJob(job *streamJob) {
	// Split text into chunks
	chunks := s.splitText(job.text)

	for i, chunk := range chunks {
		// Apply rate limiting
		if i > 0 {
			time.Sleep(s.rateLimit)
		}

		var err error
		if job.replyTo != nil {
			err = s.bot.SendReply(*job.replyTo, job.chatID, chunk)
		} else {
			err = s.bot.SendMessage(job.chatID, chunk)
		}

		if job.callback != nil {
			job.callback(i, err)
		}

		if err != nil {
			log.Printf("Failed to send chunk %d/%d: %v", i+1, len(chunks), err)
			return
		}
	}
}

// splitText splits text into chunks that fit within max length
func (s *Streamer) splitText(text string) []string {
	if len(text) <= s.maxLength {
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > 0 {
		if len(remaining) <= s.maxLength {
			chunks = append(chunks, remaining)
			break
		}

		// Try to split at sentence boundary
		chunk := remaining[:s.maxLength]
		cut := s.findBestSplitPoint(chunk)
		chunks = append(chunks, remaining[:cut])
		remaining = remaining[cut:]
	}

	return chunks
}

// findBestSplitPoint finds the best place to split text
func (s *Streamer) findBestSplitPoint(text string) int {
	// Priority order: period, newline, space
	splitPoints := []struct {
		pos  int
		char  rune
	}{
		{strings.LastIndex(text, "."), '.'},
		{strings.LastIndex(text, "\n"), '\n'},
		{strings.LastIndex(text, "?"), '?'},
		{strings.LastIndex(text, "!"), '!'},
		{strings.LastIndex(text, " "), ' '},
	}

	for _, sp := range splitPoints {
		if sp.pos > 0 && sp.pos < len(text)-1 {
			return sp.pos + 1
		}
	}

	// No good split point, use max length
	return len(text) - 100 // Leave some buffer
}

// StreamMarkdown sends a markdown-formatted message in chunks
func (s *Streamer) StreamMarkdown(chatID int64, markdown string) error {
	// For simplicity, just use regular streaming
	// Could be enhanced to preserve markdown formatting
	return s.Stream(chatID, markdown)
}

// StreamMarkdownReply sends a markdown-formatted reply in chunks
func (s *Streamer) StreamMarkdownReply(messageID int, chatID int64, markdown string) error {
	return s.StreamReply(messageID, chatID, markdown)
}

// Errors
var (
	ErrQueueFull = fmt.Errorf("streaming queue is full")
)
