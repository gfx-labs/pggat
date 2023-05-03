package auth

import "testing"

type TestCase struct {
	Username string
	Password string
	Salt     [4]byte
	Encoded  string
}

var Cases = []TestCase{
	{
		Username: "foo",
		Password: "bar",
		Salt:     [...]byte{49, 216, 227, 148},
		Encoded:  "md5042e94d42b7d6d5240214a5d2787d66c",
	},
	{
		Username: "foo",
		Password: "bar",
		Salt:     [...]byte{31, 184, 173, 138},
		Encoded:  "md51ad732d286c85df38d63c98d29a43b7d",
	},
	{
		Username: "postgres",
		Password: "password",
		Salt:     [...]byte{64, 94, 241, 253},
		Encoded:  "md5c9b2e85e17689ce9c02b6c45913c5e4f",
	},
	{
		Username: "postgres",
		Password: "password",
		Salt:     [...]byte{154, 100, 162, 40},
		Encoded:  "md5be1eace6d866b585ee20a3f2f12e9ab2",
	},
}

func TestEncodeMD5(t *testing.T) {
	for _, c := range Cases {
		encoded := EncodeMD5(c.Username, c.Password, c.Salt)
		if encoded != c.Encoded {
			t.Error("encoding failed! expected", c.Encoded, "but got", encoded)
		}
	}
}

func TestCheckMD5(t *testing.T) {
	for _, c := range Cases {
		if !CheckMD5(c.Username, c.Password, c.Salt, c.Encoded) {
			t.Error("check failed!")
		}
	}
}
