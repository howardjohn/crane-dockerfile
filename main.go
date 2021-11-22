package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/pflag"
)

var (
	env        = pflag.StringToString("env", nil, "env")
	user       = pflag.String("user", "", "user")
	entrypoint = pflag.String("entrypoint", "", "entrypoint")
	base       = pflag.String("base", "", "base")
	dest       = pflag.String("dest", "", "dest")
	data       = pflag.String("data", "", "data")
)

func main() {
	pflag.Parse()

	if *dest == "" {
		log.Fatal("--dest required")
	}
	if *data == "" {
		log.Fatal("--data required")
	}

	updates := make(chan v1.Update)
	go func() {
		for {
			select {
			case u := <-updates:
				log.Println(u)
			}
		}
	}()

	t0 := time.Now()
	baseImage := empty.Image
	if *base != "" {
		ref, err := name.ParseReference(*base)
		if err != nil {
			log.Fatal(err)
		}
		bi, err := remote.Image(ref, remote.WithProgress(updates))
		if err != nil {
			log.Fatal(err)
		}
		baseImage = bi
	}

	cfgFile, err := baseImage.ConfigFile()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("base in ", time.Since(t0))

	cfg := cfgFile.Config
	for k, v := range *env {
		cfg.Env = append(cfg.Env, fmt.Sprintf("%v=%v", k, v))
	}
	if *user != "" {
		cfg.User = *user
	}
	if *entrypoint != "" {
		cfg.Entrypoint = []string{*entrypoint}
	}

	updated, err := mutate.Config(baseImage, cfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("config in ", time.Since(t0))

	l, err := tarball.LayerFromFile(*data, tarball.WithCompressedCaching)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("read layer in ", time.Since(t0))

	files, err := mutate.AppendLayers(updated, l)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("layer in ", time.Since(t0))

	destRef, err := name.ParseReference(*dest)
	if err != nil {
		log.Fatal(err)
	}

	if err := remote.Write(destRef, files); err != nil {
		log.Fatal(err)
	}

	log.Println("write in ", time.Since(t0))
}
