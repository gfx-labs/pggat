package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const PrestHost = "https://psql-prest.staging.gfx.town"
const PostgresHost = "postgres://dev_rw:pGf63Aq0M5ck@pggat-dev.gfx.town:6432/prest"
const ThreadCount = 1000

type col struct {
	V int `json:"v"`
}

func spamPrest() error {
	c := col{
		V: int(rand.Int31()),
	}
	cstr, err := json.Marshal(c)
	if err != nil {
		return err
	}
	v, err := http.Post(PrestHost+"/prest/public/test", "application/json", bytes.NewReader(cstr))
	if err != nil {
		return err
	}
	var body []byte
	body, err = io.ReadAll(v.Body)
	if err != nil {
		return err
	}
	var rc col
	err = json.Unmarshal(body, &rc)
	if err != nil {
		return fmt.Errorf("error unmarshaling '%s': %w", string(body), err)
	}
	if rc != c {
		return fmt.Errorf("mismatch!!! %#v vs %#v", c, rc)
	}
	return nil
}

func spamPostgres(conn *pgx.Conn) error {
	rows, err := conn.Query(context.Background(), "INSERT INTO test (v) VALUES ($1);", rand.Int31())
	defer rows.Close()
	return err
}

var stats struct {
	max   time.Duration
	total time.Duration
	count int
	sync.Mutex
}

func spammer(spam func() error) error {
	for {
		start := time.Now()
		ticker := time.NewTicker(1 * time.Second)
		errch := make(chan error, 1)
		go func() {
			errch <- spam()
		}()
	a:
		for {
			select {
			case <-ticker.C:
				wait := time.Now().Sub(start)
				stats.Lock()
				if wait > stats.max {
					stats.max = wait
				}
				stats.Unlock()
			case err := <-errch:
				if err != nil {
					panic(err)
				}
				stats.Lock()
				wait := time.Now().Sub(start)
				stats.total += wait
				stats.count += 1
				if wait > stats.max {
					stats.max = wait
				}
				stats.Unlock()
				break a
			}
		}
	}
}

func postgresSpammer() error {
	c, err := pgx.Connect(context.Background(), PostgresHost)
	if err != nil {
		return err
	}
	return spammer(func() error {
		return spamPostgres(c)
	})
}

func prestSpammer() error {
	return spammer(spamPrest)
}

func main() {
	for i := 0; i < ThreadCount; i++ {
		go func() {
			err := postgresSpammer()
			if err != nil {
				panic(err)
			}
		}()
	}
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			stats.Lock()
			log.Printf("avg %f max %f", stats.total.Seconds()/float64(stats.count), stats.max.Seconds())
			stats.Unlock()
		}
	}
}
