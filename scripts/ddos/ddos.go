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

const PrestHost = "http://localhost:3000"
const PostgresHost = "postgres://dev_rw:pGf63Aq0M5ck@pggat-dev.gfx.town:6432/prest"
const ThreadCount = 4
const TestTime = 30 * time.Second

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
		return fmt.Errorf("mismatch!!! %#v vs %#v (raw '%s')", c, rc, string(body))
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
		func() {
			start := time.Now()
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			errch := make(chan error, 1)
			go func() {
				errch <- spam()
				close(errch)
			}()
		a:
			for {
				select {
				case now := <-ticker.C:
					wait := now.Sub(start)
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
		}()
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
	start := time.Now()
	for i := 0; i < ThreadCount; i++ {
		go func() {
			err := prestSpammer()
			if err != nil {
				panic(err)
			}
		}()
	}
	ticker := time.NewTicker(1 * time.Second)
	finish := time.After(TestTime)
	for {
		select {
		case now := <-ticker.C:
			stats.Lock()
			log.Printf("avg %f - max %f - %f/s", stats.total.Seconds()/float64(stats.count), stats.max.Seconds(), float64(stats.count)/now.Sub(start).Seconds())
			stats.Unlock()
		case now := <-finish:
			stats.Lock()
			log.Printf("TEST FINISHED: avg %f - max %f - %f/s - %d requests completed", stats.total.Seconds()/float64(stats.count), stats.max.Seconds(), float64(stats.count)/now.Sub(start).Seconds(), stats.count)
			stats.Unlock()
			return
		}
	}
}
