package main

import (
	"fmt"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/cron"
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
		err := backend.cron.Add("fetch_new_battles", "* * * * *", func() {
			fmt.Println("Running cron job: fetch_new_battles")
			err := backend.battleboards.FetchNewBattles()
			if err != nil {
				fmt.Println("Error with cron job enqueue_battles:", err)
			}
			fmt.Println("Finished cron job: fetch_new_battles")
		})

		if err != nil {
			fmt.Println("Failed to add cron job:", err)
		}

		err = backend.cron.Add("enqueue_new_battles", "* * * * *", func() {
			fmt.Println("Running cron job: enqueue_new_battles")
			err = backend.battleboards.EnqueueNewBattles()
			if err != nil {
				fmt.Println("Error with cron job enqueue_new_battles:", err)
			}
			fmt.Println("Finished cron job: enqueue_new_battles")
		})

		go func() {
			backend.battleboards.ProcessQueue()
		}()

		return e.Next()
	})

	fmt.Println("Starting backend...")
	backend.Run()
}
