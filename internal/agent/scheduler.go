package agent

import (
	"context"
	"log"

	"github.com/robfig/cron/v3"
)

// Scheduler bertanggung jawab untuk men-schedule dan manage multiple agents
type Scheduler struct {
	cron   *cron.Cron
	agents []Agent
}

// NewScheduler membuat instance scheduler baru
func NewScheduler() *Scheduler {
	return &Scheduler{
		cron:   cron.New(),
		agents: make([]Agent, 0),
	}
}

// RegisterAgent mendaftarkan agent baru ke scheduler
// Agent yang memiliki schedule akan otomatis dijadwalkan
func (s *Scheduler) RegisterAgent(agent Agent) {
	s.agents = append(s.agents, agent)

	schedule := agent.GetSchedule()
	if schedule != "" {
		_, err := s.cron.AddFunc(schedule, func() {
			log.Printf("🤖 [%s] Starting scheduled job...", agent.GetName())
			if err := agent.Execute(context.Background()); err != nil {
				log.Printf("❌ [%s] Job failed: %v", agent.GetName(), err)
			} else {
				log.Printf("✅ [%s] Job completed successfully", agent.GetName())
			}
		})

		if err != nil {
			log.Printf("⚠️ Failed to schedule agent %s: %v", agent.GetName(), err)
		} else {
			log.Printf("📅 [%s] Scheduled with cron: %s", agent.GetName(), schedule)
		}
	} else {
		log.Printf("📝 [%s] Registered as on-demand agent (no schedule)", agent.GetName())
	}
}

// Start menjalankan scheduler
func (s *Scheduler) Start() {
	s.cron.Start()
	log.Printf("🚀 Agent Scheduler started with %d registered agents", len(s.agents))
}

// Stop menghentikan scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
	log.Println("🛑 Agent Scheduler stopped")
}

// RunAgentByName menjalankan agent tertentu secara manual (on-demand)
// Berguna untuk testing atau trigger manual
func (s *Scheduler) RunAgentByName(ctx context.Context, name string) error {
	for _, agent := range s.agents {
		if agent.GetName() == name {
			log.Printf("🎯 [%s] Running on-demand execution...", name)
			return agent.Execute(ctx)
		}
	}
	log.Printf("⚠️ Agent with name '%s' not found", name)
	return nil
}

// GetRegisteredAgents mengembalikan daftar semua agent yang terdaftar
func (s *Scheduler) GetRegisteredAgents() []string {
	names := make([]string, len(s.agents))
	for i, agent := range s.agents {
		names[i] = agent.GetName()
	}
	return names
}
