package coordinator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"weaveforge/internal/agent/foreshadow"
	"weaveforge/internal/agent/setting"
	"weaveforge/internal/agent/style"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AgentHolders groups all agent references for the coordinator.
type AgentHolders struct {
	Setting    *setting.Agent
	Style      *style.Agent
	Foreshadow *foreshadow.Agent
}

// Coordinator orchestrates all agents and pushes results to the frontend.
type Coordinator struct {
	agents    AgentHolders
	intensity int // 0-10

	ctx      context.Context
	ctxMu    sync.RWMutex

	session   []SessionEvent
	sessionMu sync.Mutex
	sessionID string

	notifCh chan Notification
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// New creates a coordinator with the given agent holders.
func New(agents AgentHolders) *Coordinator {
	return &Coordinator{
		agents:    agents,
		intensity: 5,
		sessionID: uuid.New().String()[:8],
		notifCh:   make(chan Notification, 64),
		stopCh:    make(chan struct{}),
	}
}

// SetContext stores the Wails context for event emission.
func (c *Coordinator) SetContext(ctx context.Context) {
	c.ctxMu.Lock()
	c.ctx = ctx
	c.ctxMu.Unlock()
}

// Start begins processing notifications in the background.
func (c *Coordinator) Start() {
	c.wg.Add(1)
	go c.loop()
}

// Stop terminates the background loop.
func (c *Coordinator) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

// SetIntensity adjusts the assistant sensitivity (0 = off, 10 = max).
func (c *Coordinator) SetIntensity(level int) {
	if level < 0 {
		level = 0
	}
	if level > 10 {
		level = 10
	}
	c.intensity = level
}

// GetIntensity returns current intensity level.
func (c *Coordinator) GetIntensity() int {
	return c.intensity
}

// GetSessionHistory returns all recorded session events.
func (c *Coordinator) GetSessionHistory() []SessionEvent {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	cp := make([]SessionEvent, len(c.session))
	copy(cp, c.session)
	return cp
}

// ─── Event Bus ─────────────────────────────────────────────────────────

func (c *Coordinator) loop() {
	defer c.wg.Done()
	for {
		select {
		case n := <-c.notifCh:
			c.emitNotification(n)
		case <-c.stopCh:
			return
		}
	}
}

func (c *Coordinator) emitNotification(n Notification) {
	c.ctxMu.RLock()
	ctx := c.ctx
	c.ctxMu.RUnlock()
	if ctx == nil {
		return
	}
	n.Time = time.Now().Format("15:04:05")
	runtime.EventsEmit(ctx, "coordinator:notification", n)
}

func (c *Coordinator) recordEvent(tp, agent, content, action string) {
	c.sessionMu.Lock()
	c.session = append(c.session, SessionEvent{
		ID:         uuid.New().String(),
		Type:       tp,
		Agent:      agent,
		Content:    content,
		UserAction: action,
		Timestamp:  time.Now(),
	})
	// Keep last 500
	if len(c.session) > 500 {
		overflow := len(c.session) - 500
		c.session = c.session[overflow:]
	}
	c.sessionMu.Unlock()
}

// ─── Agent Orchestration ───────────────────────────────────────────────

// OnParagraphWritten runs agents asynchronously and pushes results.
func (c *Coordinator) OnParagraphWritten(chapterID, paragraphText string) {
	if c.intensity == 0 || paragraphText == "" {
		return
	}

	threshold := float64(c.intensity) / 10.0 // 0.0 - 1.0

	// Fire all checks concurrently
	var wg sync.WaitGroup
	ctx := context.Background()

	// a) Style deviation
	if c.agents.Style != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.checkStyleDeviation(ctx, paragraphText, threshold)
		}()
	}

	// b) Foreshadow detection
	if c.agents.Foreshadow != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.detectForeshadow(ctx, chapterID, paragraphText, threshold)
		}()
	}

	wg.Wait()
}

func (c *Coordinator) checkStyleDeviation(ctx context.Context, text string, threshold float64) {
	if c.intensity < 8 {
		return
	}
	profiles, _ := c.agents.Style.ListProfiles(ctx)
	if len(profiles) > 0 {
		c.notifCh <- Notification{
			ID:       uuid.New().String(),
			Agent:    "style",
			Title:    fmt.Sprintf("已学习 %d 个风格档案", len(profiles)),
			Content:  "继续写作或前往风格润色面板手动润色文本。",
			Severity: "info",
			Action:   "dismiss",
		}
	}
}

func (c *Coordinator) detectForeshadow(ctx context.Context, chapterID, text string, threshold float64) {
	if c.intensity < 4 {
		return
	}
	candidates, err := c.agents.Foreshadow.AutoDetectForeshadowing(ctx, text)
	if err != nil || len(candidates) == 0 {
		return
	}
	for _, cand := range candidates {
		if cand.Confidence < threshold {
			continue
		}
		c.notifCh <- Notification{
			ID:       uuid.New().String(),
			Agent:    "foreshadow",
			Title:    fmt.Sprintf("发现伏笔：%s", cand.Type),
			Content:  cand.Text,
			Severity: "info",
			Action:   "accept",
		}
	}
}

// RecordUserAction logs whether the user accepted or dismissed a suggestion.
func (c *Coordinator) RecordUserAction(notifID, action string) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	ev := SessionEvent{
		ID:         uuid.New().String(),
		Type:       "user_action",
		Agent:      "",
		Content:    notifID,
		UserAction: action,
		Timestamp:  time.Now(),
	}
	c.session = append(c.session, ev)
}
