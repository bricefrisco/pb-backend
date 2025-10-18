package main

import (
	"fmt"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/cron"
	sCron "github.com/robfig/cron/v3"
	"log"
)

type Backend struct {
	app          *pocketbase.PocketBase
	cron         *cron.Cron
	battleboards *Battleboards
}

func NewBackend() *Backend {
	app := pocketbase.New()
	return &Backend{
		app:          app,
		cron:         cron.New(),
		battleboards: NewBattleboards(app),
	}
}

func (b *Backend) Run() {
	b.cron.Start()
	if err := b.app.Start(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	backend := NewBackend()

	backend.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		c := sCron.New(
			sCron.WithSeconds(),
			sCron.WithChain(
				sCron.SkipIfStillRunning(sCron.DefaultLogger),
			),
		)

		_, err := c.AddFunc("30 * * * * *", func() {
			fmt.Println("Scheduler started")
			err := backend.battleboards.FetchNewBattles()
			if err != nil {
				fmt.Println("Error with cron job (FetchNewBattles):", err)
			}

			err = backend.battleboards.EnqueueNewBattles()
			if err != nil {
				fmt.Println("Error with cron job (EnqueueNewBattles):", err)
			}
			fmt.Println("Scheduler finished")
		})

		c.Start()

		if err != nil {
			fmt.Println("Failed to schedule cron job:", err)
		}

		go func() {
			backend.battleboards.ProcessQueue()
		}()

		return e.Next()
	})

	fmt.Println("Starting backend...")
	backend.Run()
}
