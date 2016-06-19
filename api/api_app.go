// Copyright © 2016 Thomas Rabaix <thomas.rabaix@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package api

import (
	"encoding/json"

	log "github.com/Sirupsen/logrus"
	"github.com/rande/goapp"
	"github.com/rande/pkgmirror"
	"goji.io"
	"goji.io/pat"
)

func ConfigureApp(config *pkgmirror.Config, l *goapp.Lifecycle) {

	l.Register(func(app *goapp.App) error {
		app.Set("pkgmirror.channel.state", func(app *goapp.App) interface{} {
			return make(chan pkgmirror.State)
		})

		app.Set("pkgmirror.sse.broker", func(app *goapp.App) interface{} {
			return pkgmirror.NewSseBroker()
		})

		return nil
	})

	l.Prepare(func(app *goapp.App) error {
		mux := app.Get("mux").(*goji.Mux)
		mux.HandleFuncC(pat.Get("/api/mirrors"), Api_GET_MirrorServices(app))
		mux.HandleFunc(pat.Get("/api/sse"), Api_GET_Sse(app))

		return nil
	})

	l.Run(func(app *goapp.App, state *goapp.GoroutineState) error {
		ch := app.Get("pkgmirror.channel.state").(chan pkgmirror.State)
		brk := app.Get("pkgmirror.sse.broker").(*pkgmirror.SseBroker)
		logger := app.Get("logger").(*log.Logger)

		logger.Info("Start the SSE Broker")
		// start the broken
		go brk.Listen()

		states := map[string]pkgmirror.State{}

		// send the current state
		brk.OnConnect(func() {
			for _, s := range states {
				data, _ := json.Marshal(&s)

				brk.Notifier <- data
			}
		})

		for {
			select {
			case s := <-ch:
				logger.WithFields(log.Fields{
					"id":      s.Id,
					"message": s.Message,
					"status":  s.Status,
				}).Debug("Receive message")

				states[s.Id] = s

				data, _ := json.Marshal(&s)

				brk.Notifier <- data
			}
		}
	})

}