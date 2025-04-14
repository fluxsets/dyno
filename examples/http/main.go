package main

import (
	"context"
	"github.com/fluxsets/dyno"
	"github.com/fluxsets/dyno/server/http"
	"log"
	gohttp "net/http"
)

type Config struct {
	Addr string `json:"addr"`
	A    string `json:"a"`
	B    string `json:"b"`
}

func main() {
	option := dyno.OptionFromFlags()
	option.Name = "http-example"
	option.Version = "v0.0.1"
	cli := dyno.NewCLI(option, func(ctx context.Context, do dyno.Dyno) error {
		opt := do.Option()
		logger := do.Logger()
		logger.Info("parsed option", "option", opt)
		config := &Config{}
		if err := do.Config().Unmarshal(config); err != nil {
			return err
		}
		logger.Info("parsed config", "config", config)
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
	})
	err := cli.Run()
	if err != nil {
		log.Fatal(err)
	}
}
