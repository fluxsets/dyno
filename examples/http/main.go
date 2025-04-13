package main

import (
	"context"
	"github.com/fluxsets/dyno"
	"github.com/fluxsets/dyno/server/http"
	"log"
	gohttp "net/http"
)

func main() {
	cli := dyno.NewCLI(func(ctx context.Context, do dyno.Dyno) error {
		do.Hooks().OnStart(func(ctx context.Context) error {
			do.Logger().Info("pre start")
			return nil
		})
		router := http.NewRouter()
		router.HandleFunc("/hello", func(rw gohttp.ResponseWriter, r *gohttp.Request) {
			rw.Write([]byte("hello"))
		})
		if err := do.Deploy(http.NewServer(":9090", router.ServeHTTP)); err != nil {
			return err
		}

		return nil
	}, dyno.BindOption())
	err := cli.Run()
	if err != nil {
		log.Fatal(err)
	}
}
