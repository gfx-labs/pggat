package eqp

type Close interface {
	Done()
	close()
}

type ClosePortal struct {
	target string
	portal Portal
}

func (T ClosePortal) Done() {
	T.portal.Done()
}

func (ClosePortal) close() {}

var _ Close = ClosePortal{}

type ClosePreparedStatement struct {
	target            string
	preparedStatement PreparedStatement
}

func (T ClosePreparedStatement) Done() {
	T.preparedStatement.Done()
}

func (ClosePreparedStatement) close() {}

var _ Close = ClosePreparedStatement{}
