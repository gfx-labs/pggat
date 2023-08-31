package packets

import "pggat2/lib/fed"

const (
	TypeAuthentication           fed.Type = 'R'
	TypeBackendKeyData           fed.Type = 'K'
	TypeBind                     fed.Type = 'B'
	TypeBindComplete             fed.Type = '2'
	TypeClose                    fed.Type = 'C'
	TypeCloseComplete            fed.Type = '3'
	TypeCommandComplete          fed.Type = 'C'
	TypeCopyData                 fed.Type = 'd'
	TypeCopyDone                 fed.Type = 'c'
	TypeCopyFail                 fed.Type = 'f'
	TypeCopyInResponse           fed.Type = 'G'
	TypeCopyOutResponse          fed.Type = 'H'
	TypeCopyBothResponse         fed.Type = 'W'
	TypeDataRow                  fed.Type = 'D'
	TypeDescribe                 fed.Type = 'D'
	TypeEmptyQueryResponse       fed.Type = 'I'
	TypeErrorResponse            fed.Type = 'E'
	TypeExecute                  fed.Type = 'E'
	TypeFlush                    fed.Type = 'H'
	TypeFunctionCall             fed.Type = 'F'
	TypeFunctionCallResponse     fed.Type = 'V'
	TypeAuthenticationResponse   fed.Type = 'p'
	TypeNegotiateProtocolVersion fed.Type = 'v'
	TypeNoData                   fed.Type = 'n'
	TypeNoticeResponse           fed.Type = 'N'
	TypeNotificationResponse     fed.Type = 'A'
	TypeParameterDescription     fed.Type = 't'
	TypeParameterStatus          fed.Type = 'S'
	TypeParse                    fed.Type = 'P'
	TypeParseComplete            fed.Type = '1'
	TypePortalSuspended          fed.Type = 's'
	TypeQuery                    fed.Type = 'Q'
	TypeReadyForQuery            fed.Type = 'Z'
	TypeRowDescription           fed.Type = 'T'
	TypeSync                     fed.Type = 'S'
	TypeTerminate                fed.Type = 'X'
)
