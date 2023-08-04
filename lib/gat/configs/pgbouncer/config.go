package pgbouncer

type Config struct {
	Databases map[string]string `ini:"databases"`
	Users     map[string]string `ini:"users"`
}

func Test() {

}
