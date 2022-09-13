package www

import (
	"html/template"
	"io"

	"gfx.cafe/gfx/pggat/lib/gat/gatling"
)

const adminTmplString = `


`

var adminTmpl = template.Must(template.New("admin_page").Parse(adminTmplString))

type AdminPage struct {
	G *gatling.Gatling
}

func NewAdminPage(g *gatling.Gatling) *AdminPage {
	return &AdminPage{
		G: g,
	}
}

func (p *AdminPage) Renderer(w io.Writer) error {
	return adminTmpl.Execute(w, p)
}

func (p *AdminPage) test() {
}
