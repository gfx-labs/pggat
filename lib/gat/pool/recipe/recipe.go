package recipe

type Recipe struct {
	options Options
}

func NewRecipe(options Options) *Recipe {
	return &Recipe{
		options: options,
	}
}

func (T *Recipe) Dial() {

}
