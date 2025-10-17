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
		err := backend.cron.Add("enqueue_battles", "* * * * *", func() {
			fmt.Println("Running cron job: enqueue_battles")
			err := backend.battleboards.TempTest()
			if err != nil {
				fmt.Println("Error with cron job enqueue_battles:", err)
			}
			fmt.Println("Finished cron job: enqueue_battles")
		})
		if err != nil {
			return err
		}

		err = backend.battleboards.TempTest()
		if err != nil {
			return err
		}

		return e.Next()
	})

	backend.Run()
}
