package www

import (
	"io"
	"net/http"
)

type Index struct {
	Pages []*Page
}

type Page struct {
	Title    string
	Renderer func(io.Writer) error
}

func (p *Page) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := p.Renderer(w)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
}
