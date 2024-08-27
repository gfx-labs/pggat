package packets

// automatically generated. do not edit

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/util/slices"

	"errors"
)

var (
	ErrUnexpectedPacket = errors.New("unexpected packet")
	ErrInvalidFormat    = errors.New("invalid packet format")
)

const (
	TypeAuthentication           = 'R'
	TypeBackendKeyData           = 'K'
	TypeBind                     = 'B'
	TypeBindComplete             = '2'
	TypeClose                    = 'C'
	TypeCloseComplete            = '3'
	TypeCommandComplete          = 'C'
	TypeCopyBothResponse         = 'W'
	TypeCopyData                 = 'd'
	TypeCopyDone                 = 'c'
	TypeCopyFail                 = 'f'
	TypeCopyInResponse           = 'G'
	TypeCopyOutResponse          = 'H'
	TypeDataRow                  = 'D'
	TypeDescribe                 = 'D'
	TypeEmptyQueryResponse       = 'I'
	TypeExecute                  = 'E'
	TypeFlush                    = 'H'
	TypeFunctionCall             = 'F'
	TypeFunctionCallResponse     = 'V'
	TypeGSSResponse              = 'p'
	TypeMarkiplierResponse       = 'E'
	TypeNegotiateProtocolVersion = 'v'
	TypeNoData                   = 'n'
	TypeNoticeResponse           = 'N'
	TypeNotificationResponse     = 'A'
	TypeParameterDescription     = 't'
	TypeParameterStatus          = 'S'
	TypeParse                    = 'P'
	TypeParseComplete            = '1'
	TypePasswordMessage          = 'p'
	TypePortalSuspended          = 's'
	TypeQuery                    = 'Q'
	TypeReadyForQuery            = 'Z'
	TypeRowDescription           = 'T'
	TypeSASLInitialResponse      = 'p'
	TypeSASLResponse             = 'p'
	TypeSync                     = 'S'
	TypeTerminate                = 'X'
)

type AuthenticationPayloadCleartextPassword struct{}

func (*AuthenticationPayloadCleartextPassword) AuthenticationPayloadMode() int32 {
	return 3
}

func (T *AuthenticationPayloadCleartextPassword) Length() (length int) {

	return
}

func (T *AuthenticationPayloadCleartextPassword) ReadFrom(decoder *fed.Decoder) (err error) {

	return
}

func (T *AuthenticationPayloadCleartextPassword) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

type AuthenticationPayloadGSS struct{}

func (*AuthenticationPayloadGSS) AuthenticationPayloadMode() int32 {
	return 7
}

func (T *AuthenticationPayloadGSS) Length() (length int) {

	return
}

func (T *AuthenticationPayloadGSS) ReadFrom(decoder *fed.Decoder) (err error) {

	return
}

func (T *AuthenticationPayloadGSS) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

type AuthenticationPayloadGSSContinue []uint8

func (*AuthenticationPayloadGSSContinue) AuthenticationPayloadMode() int32 {
	return 8
}

func (T *AuthenticationPayloadGSSContinue) Length() (length int) {
	for _, temp1 := range *T {
		_ = temp1

		length += 1

	}

	return
}

func (T *AuthenticationPayloadGSSContinue) ReadFrom(decoder *fed.Decoder) (err error) {
	(*T) = (*T)[:0]

	for {
		if decoder.Position() >= decoder.Length() {
			break
		}

		(*T) = slices.Resize((*T), len((*T))+1)

		*(*uint8)(&((*T)[len((*T))-1])), err = decoder.Uint8()
		if err != nil {
			return
		}

	}

	return
}

func (T *AuthenticationPayloadGSSContinue) WriteTo(encoder *fed.Encoder) (err error) {
	for _, temp2 := range *T {
		err = encoder.Uint8(uint8(temp2))
		if err != nil {
			return
		}

	}

	return
}

type AuthenticationPayloadKerberosV5 struct{}

func (*AuthenticationPayloadKerberosV5) AuthenticationPayloadMode() int32 {
	return 2
}

func (T *AuthenticationPayloadKerberosV5) Length() (length int) {

	return
}

func (T *AuthenticationPayloadKerberosV5) ReadFrom(decoder *fed.Decoder) (err error) {

	return
}

func (T *AuthenticationPayloadKerberosV5) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

type AuthenticationPayloadMD5Password [4]uint8

func (*AuthenticationPayloadMD5Password) AuthenticationPayloadMode() int32 {
	return 5
}

func (T *AuthenticationPayloadMD5Password) Length() (length int) {
	for _, temp3 := range *T {
		_ = temp3

		length += 1

	}

	return
}

func (T *AuthenticationPayloadMD5Password) ReadFrom(decoder *fed.Decoder) (err error) {
	for temp4 := 0; temp4 < 4; temp4++ {
		*(*uint8)(&((*T)[temp4])), err = decoder.Uint8()
		if err != nil {
			return
		}

	}

	return
}

func (T *AuthenticationPayloadMD5Password) WriteTo(encoder *fed.Encoder) (err error) {
	for _, temp5 := range *T {
		err = encoder.Uint8(uint8(temp5))
		if err != nil {
			return
		}

	}

	return
}

type AuthenticationPayloadOk struct{}

func (*AuthenticationPayloadOk) AuthenticationPayloadMode() int32 {
	return 0
}

func (T *AuthenticationPayloadOk) Length() (length int) {

	return
}

func (T *AuthenticationPayloadOk) ReadFrom(decoder *fed.Decoder) (err error) {

	return
}

func (T *AuthenticationPayloadOk) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

type AuthenticationPayloadSASLMethod struct {
	Method string
}

type AuthenticationPayloadSASL []AuthenticationPayloadSASLMethod

func (*AuthenticationPayloadSASL) AuthenticationPayloadMode() int32 {
	return 10
}

func (T *AuthenticationPayloadSASL) Length() (length int) {
	for _, temp6 := range *T {
		_ = temp6

		length += len(temp6.Method) + 1

	}

	var temp7 string
	_ = temp7

	length += len(temp7) + 1

	return
}

func (T *AuthenticationPayloadSASL) ReadFrom(decoder *fed.Decoder) (err error) {
	(*T) = (*T)[:0]

	for {
		(*T) = slices.Resize((*T), len((*T))+1)

		*(*string)(&((*T)[len((*T))-1].Method)), err = decoder.String()
		if err != nil {
			return
		}
		if (*T)[len((*T))-1].Method == *new(string) {
			(*T) = (*T)[:len((*T))-1]
			break
		}
	}

	return
}

func (T *AuthenticationPayloadSASL) WriteTo(encoder *fed.Encoder) (err error) {
	for _, temp8 := range *T {
		err = encoder.String(string(temp8.Method))
		if err != nil {
			return
		}

	}

	var temp9 string

	err = encoder.String(string(temp9))
	if err != nil {
		return
	}

	return
}

type AuthenticationPayloadSASLContinue []uint8

func (*AuthenticationPayloadSASLContinue) AuthenticationPayloadMode() int32 {
	return 11
}

func (T *AuthenticationPayloadSASLContinue) Length() (length int) {
	for _, temp10 := range *T {
		_ = temp10

		length += 1

	}

	return
}

func (T *AuthenticationPayloadSASLContinue) ReadFrom(decoder *fed.Decoder) (err error) {
	(*T) = (*T)[:0]

	for {
		if decoder.Position() >= decoder.Length() {
			break
		}

		(*T) = slices.Resize((*T), len((*T))+1)

		*(*uint8)(&((*T)[len((*T))-1])), err = decoder.Uint8()
		if err != nil {
			return
		}

	}

	return
}

func (T *AuthenticationPayloadSASLContinue) WriteTo(encoder *fed.Encoder) (err error) {
	for _, temp11 := range *T {
		err = encoder.Uint8(uint8(temp11))
		if err != nil {
			return
		}

	}

	return
}

type AuthenticationPayloadSASLFinal []uint8

func (*AuthenticationPayloadSASLFinal) AuthenticationPayloadMode() int32 {
	return 12
}

func (T *AuthenticationPayloadSASLFinal) Length() (length int) {
	for _, temp12 := range *T {
		_ = temp12

		length += 1

	}

	return
}

func (T *AuthenticationPayloadSASLFinal) ReadFrom(decoder *fed.Decoder) (err error) {
	(*T) = (*T)[:0]

	for {
		if decoder.Position() >= decoder.Length() {
			break
		}

		(*T) = slices.Resize((*T), len((*T))+1)

		*(*uint8)(&((*T)[len((*T))-1])), err = decoder.Uint8()
		if err != nil {
			return
		}

	}

	return
}

func (T *AuthenticationPayloadSASLFinal) WriteTo(encoder *fed.Encoder) (err error) {
	for _, temp13 := range *T {
		err = encoder.Uint8(uint8(temp13))
		if err != nil {
			return
		}

	}

	return
}

type AuthenticationPayloadSSPI struct{}

func (*AuthenticationPayloadSSPI) AuthenticationPayloadMode() int32 {
	return 9
}

func (T *AuthenticationPayloadSSPI) Length() (length int) {

	return
}

func (T *AuthenticationPayloadSSPI) ReadFrom(decoder *fed.Decoder) (err error) {

	return
}

func (T *AuthenticationPayloadSSPI) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

type AuthenticationPayloadMode interface {
	AuthenticationPayloadMode() int32

	Length() int
	ReadFrom(decoder *fed.Decoder) error
	WriteTo(encoder *fed.Encoder) error
}
type AuthenticationPayload struct {
	Mode AuthenticationPayloadMode
}

type Authentication AuthenticationPayload

func (T *Authentication) Type() fed.Type {
	return TypeAuthentication
}

func (T *Authentication) Length() (length int) {
	length += 4

	length += (*T).Mode.Length()

	return
}

func (T *Authentication) TypeName() string {
	return "Authentication"
}

func (T *Authentication) String() string {
	return T.TypeName()
}

func (T *Authentication) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	var temp14 int32

	*(*int32)(&(temp14)), err = decoder.Int32()
	if err != nil {
		return
	}

	switch temp14 {
	case 3:
		(*T).Mode = new(AuthenticationPayloadCleartextPassword)
	case 7:
		(*T).Mode = new(AuthenticationPayloadGSS)
	case 8:
		(*T).Mode = new(AuthenticationPayloadGSSContinue)
	case 2:
		(*T).Mode = new(AuthenticationPayloadKerberosV5)
	case 5:
		(*T).Mode = new(AuthenticationPayloadMD5Password)
	case 0:
		(*T).Mode = new(AuthenticationPayloadOk)
	case 10:
		(*T).Mode = new(AuthenticationPayloadSASL)
	case 11:
		(*T).Mode = new(AuthenticationPayloadSASLContinue)
	case 12:
		(*T).Mode = new(AuthenticationPayloadSASLFinal)
	case 9:
		(*T).Mode = new(AuthenticationPayloadSSPI)
	default:
		err = ErrInvalidFormat
		return
	}

	err = (*T).Mode.ReadFrom(decoder)
	if err != nil {
		return
	}

	return
}

func (T *Authentication) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.Int32(int32((*T).Mode.AuthenticationPayloadMode()))
	if err != nil {
		return
	}

	err = (*T).Mode.WriteTo(encoder)
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*Authentication)(nil)

type BackendKeyDataPayload struct {
	ProcessID int32
	SecretKey int32
}

type BackendKeyData BackendKeyDataPayload

func (T *BackendKeyData) Type() fed.Type {
	return TypeBackendKeyData
}

func (T *BackendKeyData) Length() (length int) {
	length += 4

	length += 4

	return
}

func (T *BackendKeyData) TypeName() string {
	return "BackendKeyData"
}

func (T *BackendKeyData) String() string {
	return T.TypeName()
}

func (T *BackendKeyData) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*int32)(&((*T).ProcessID)), err = decoder.Int32()
	if err != nil {
		return
	}
	*(*int32)(&((*T).SecretKey)), err = decoder.Int32()
	if err != nil {
		return
	}

	return
}

func (T *BackendKeyData) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.Int32(int32((*T).ProcessID))
	if err != nil {
		return
	}

	err = encoder.Int32(int32((*T).SecretKey))
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*BackendKeyData)(nil)

type BindPayload struct {
	Destination       string
	Source            string
	FormatCodes       []int16
	Parameters        [][]uint8
	ResultFormatCodes []int16
}

type Bind BindPayload

func (T *Bind) Type() fed.Type {
	return TypeBind
}

func (T *Bind) Length() (length int) {
	length += len((*T).Destination) + 1

	length += len((*T).Source) + 1

	temp15 := uint16(len((*T).FormatCodes))
	_ = temp15

	length += 2

	for _, temp16 := range (*T).FormatCodes {
		_ = temp16

		length += 2

	}

	temp17 := uint16(len((*T).Parameters))
	_ = temp17

	length += 2

	for _, temp18 := range (*T).Parameters {
		_ = temp18

		temp19 := int32(len(temp18))
		_ = temp19

		length += 4

		for _, temp20 := range temp18 {
			_ = temp20

			length += 1

		}

	}

	temp21 := uint16(len((*T).ResultFormatCodes))
	_ = temp21

	length += 2

	for _, temp22 := range (*T).ResultFormatCodes {
		_ = temp22

		length += 2

	}

	return
}

func (T *Bind) TypeName() string {
	return "Bind"
}

func (T *Bind) String() string {
	return T.TypeName()
}

func (T *Bind) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*string)(&((*T).Destination)), err = decoder.String()
	if err != nil {
		return
	}
	*(*string)(&((*T).Source)), err = decoder.String()
	if err != nil {
		return
	}
	var temp23 uint16
	*(*uint16)(&(temp23)), err = decoder.Uint16()
	if err != nil {
		return
	}

	(*T).FormatCodes = slices.Resize((*T).FormatCodes, int(temp23))

	for temp24 := 0; temp24 < int(temp23); temp24++ {
		*(*int16)(&((*T).FormatCodes[temp24])), err = decoder.Int16()
		if err != nil {
			return
		}

	}

	var temp25 uint16
	*(*uint16)(&(temp25)), err = decoder.Uint16()
	if err != nil {
		return
	}

	(*T).Parameters = slices.Resize((*T).Parameters, int(temp25))

	for temp26 := 0; temp26 < int(temp25); temp26++ {
		var temp27 int32
		*(*int32)(&(temp27)), err = decoder.Int32()
		if err != nil {
			return
		}

		if temp27 == -1 {
			(*T).Parameters[temp26] = nil
		} else {
			if (*T).Parameters[temp26] == nil {
				(*T).Parameters[temp26] = make([]uint8, int(temp27))
			} else {
				(*T).Parameters[temp26] = slices.Resize((*T).Parameters[temp26], int(temp27))
			}

			for temp28 := 0; temp28 < int(temp27); temp28++ {
				*(*uint8)(&((*T).Parameters[temp26][temp28])), err = decoder.Uint8()
				if err != nil {
					return
				}

			}
		}

	}

	var temp29 uint16
	*(*uint16)(&(temp29)), err = decoder.Uint16()
	if err != nil {
		return
	}

	(*T).ResultFormatCodes = slices.Resize((*T).ResultFormatCodes, int(temp29))

	for temp30 := 0; temp30 < int(temp29); temp30++ {
		*(*int16)(&((*T).ResultFormatCodes[temp30])), err = decoder.Int16()
		if err != nil {
			return
		}

	}

	return
}

func (T *Bind) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.String(string((*T).Destination))
	if err != nil {
		return
	}

	err = encoder.String(string((*T).Source))
	if err != nil {
		return
	}

	temp31 := uint16(len((*T).FormatCodes))

	err = encoder.Uint16(uint16(temp31))
	if err != nil {
		return
	}

	for _, temp32 := range (*T).FormatCodes {
		err = encoder.Int16(int16(temp32))
		if err != nil {
			return
		}

	}

	temp33 := uint16(len((*T).Parameters))

	err = encoder.Uint16(uint16(temp33))
	if err != nil {
		return
	}

	for _, temp34 := range (*T).Parameters {
		temp35 := int32(len(temp34))

		if temp34 == nil {
			temp35 = -1
		}

		err = encoder.Int32(int32(temp35))
		if err != nil {
			return
		}

		for _, temp36 := range temp34 {
			err = encoder.Uint8(uint8(temp36))
			if err != nil {
				return
			}

		}

	}

	temp37 := uint16(len((*T).ResultFormatCodes))

	err = encoder.Uint16(uint16(temp37))
	if err != nil {
		return
	}

	for _, temp38 := range (*T).ResultFormatCodes {
		err = encoder.Int16(int16(temp38))
		if err != nil {
			return
		}

	}

	return
}

var _ fed.Packet = (*Bind)(nil)

type BindComplete struct{}

func (T *BindComplete) Type() fed.Type {
	return TypeBindComplete
}

func (T *BindComplete) Length() (length int) {

	return
}

func (T *BindComplete) TypeName() string {
	return "BindComplete"
}

func (T *BindComplete) String() string {
	return T.TypeName()
}

func (T *BindComplete) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	return
}

func (T *BindComplete) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

var _ fed.Packet = (*BindComplete)(nil)

type ClosePayload struct {
	Which uint8
	Name  string
}

type Close ClosePayload

func (T *Close) Type() fed.Type {
	return TypeClose
}

func (T *Close) Length() (length int) {
	length += 1

	length += len((*T).Name) + 1

	return
}

func (T *Close) TypeName() string {
	return "Close"
}

func (T *Close) String() string {
	return T.TypeName()
}

func (T *Close) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*uint8)(&((*T).Which)), err = decoder.Uint8()
	if err != nil {
		return
	}
	*(*string)(&((*T).Name)), err = decoder.String()
	if err != nil {
		return
	}

	return
}

func (T *Close) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.Uint8(uint8((*T).Which))
	if err != nil {
		return
	}

	err = encoder.String(string((*T).Name))
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*Close)(nil)

type CloseComplete struct{}

func (T *CloseComplete) Type() fed.Type {
	return TypeCloseComplete
}

func (T *CloseComplete) Length() (length int) {

	return
}

func (T *CloseComplete) TypeName() string {
	return "CloseComplete"
}

func (T *CloseComplete) String() string {
	return T.TypeName()
}

func (T *CloseComplete) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	return
}

func (T *CloseComplete) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

var _ fed.Packet = (*CloseComplete)(nil)

type CommandComplete string

func (T *CommandComplete) Type() fed.Type {
	return TypeCommandComplete
}

func (T *CommandComplete) Length() (length int) {
	length += len((*T)) + 1

	return
}

func (T *CommandComplete) TypeName() string {
	return "CommandComplete"
}

func (T *CommandComplete) String() string {
	return T.TypeName()
}

func (T *CommandComplete) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*string)(&(*T)), err = decoder.String()
	if err != nil {
		return
	}

	return
}

func (T *CommandComplete) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.String(string((*T)))
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*CommandComplete)(nil)

type CopyBothResponsePayload struct {
	Mode              int8
	ColumnFormatCodes []int16
}

type CopyBothResponse CopyBothResponsePayload

func (T *CopyBothResponse) Type() fed.Type {
	return TypeCopyBothResponse
}

func (T *CopyBothResponse) Length() (length int) {
	length += 1

	temp39 := uint16(len((*T).ColumnFormatCodes))
	_ = temp39

	length += 2

	for _, temp40 := range (*T).ColumnFormatCodes {
		_ = temp40

		length += 2

	}

	return
}

func (T *CopyBothResponse) TypeName() string {
	return "CopyBothResponse"
}

func (T *CopyBothResponse) String() string {
	return T.TypeName()
}

func (T *CopyBothResponse) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*int8)(&((*T).Mode)), err = decoder.Int8()
	if err != nil {
		return
	}
	var temp41 uint16
	*(*uint16)(&(temp41)), err = decoder.Uint16()
	if err != nil {
		return
	}

	(*T).ColumnFormatCodes = slices.Resize((*T).ColumnFormatCodes, int(temp41))

	for temp42 := 0; temp42 < int(temp41); temp42++ {
		*(*int16)(&((*T).ColumnFormatCodes[temp42])), err = decoder.Int16()
		if err != nil {
			return
		}

	}

	return
}

func (T *CopyBothResponse) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.Int8(int8((*T).Mode))
	if err != nil {
		return
	}

	temp43 := uint16(len((*T).ColumnFormatCodes))

	err = encoder.Uint16(uint16(temp43))
	if err != nil {
		return
	}

	for _, temp44 := range (*T).ColumnFormatCodes {
		err = encoder.Int16(int16(temp44))
		if err != nil {
			return
		}

	}

	return
}

var _ fed.Packet = (*CopyBothResponse)(nil)

type CopyData []uint8

func (T *CopyData) Type() fed.Type {
	return TypeCopyData
}

func (T *CopyData) Length() (length int) {
	for _, temp45 := range *T {
		_ = temp45

		length += 1

	}

	return
}

func (T *CopyData) TypeName() string {
	return "CopyData"
}

func (T *CopyData) String() string {
	return T.TypeName()
}

func (T *CopyData) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	(*T) = (*T)[:0]

	for {
		if decoder.Position() >= decoder.Length() {
			break
		}

		(*T) = slices.Resize((*T), len((*T))+1)

		*(*uint8)(&((*T)[len((*T))-1])), err = decoder.Uint8()
		if err != nil {
			return
		}

	}

	return
}

func (T *CopyData) WriteTo(encoder *fed.Encoder) (err error) {
	for _, temp46 := range *T {
		err = encoder.Uint8(uint8(temp46))
		if err != nil {
			return
		}

	}

	return
}

var _ fed.Packet = (*CopyData)(nil)

type CopyDone struct{}

func (T *CopyDone) Type() fed.Type {
	return TypeCopyDone
}

func (T *CopyDone) Length() (length int) {

	return
}

func (T *CopyDone) TypeName() string {
	return "CopyDone"
}

func (T *CopyDone) String() string {
	return T.TypeName()
}

func (T *CopyDone) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	return
}

func (T *CopyDone) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

var _ fed.Packet = (*CopyDone)(nil)

type CopyFail string

func (T *CopyFail) Type() fed.Type {
	return TypeCopyFail
}

func (T *CopyFail) Length() (length int) {
	length += len((*T)) + 1

	return
}

func (T *CopyFail) TypeName() string {
	return "CopyFail"
}

func (T *CopyFail) String() string {
	return T.TypeName()
}

func (T *CopyFail) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*string)(&(*T)), err = decoder.String()
	if err != nil {
		return
	}

	return
}

func (T *CopyFail) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.String(string((*T)))
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*CopyFail)(nil)

type CopyInResponsePayload struct {
	Mode              int8
	ColumnFormatCodes []int16
}

type CopyInResponse CopyInResponsePayload

func (T *CopyInResponse) Type() fed.Type {
	return TypeCopyInResponse
}

func (T *CopyInResponse) Length() (length int) {
	length += 1

	temp47 := uint16(len((*T).ColumnFormatCodes))
	_ = temp47

	length += 2

	for _, temp48 := range (*T).ColumnFormatCodes {
		_ = temp48

		length += 2

	}

	return
}

func (T *CopyInResponse) TypeName() string {
	return "CopyInResponse"
}

func (T *CopyInResponse) String() string {
	return T.TypeName()
}

func (T *CopyInResponse) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*int8)(&((*T).Mode)), err = decoder.Int8()
	if err != nil {
		return
	}
	var temp49 uint16
	*(*uint16)(&(temp49)), err = decoder.Uint16()
	if err != nil {
		return
	}

	(*T).ColumnFormatCodes = slices.Resize((*T).ColumnFormatCodes, int(temp49))

	for temp50 := 0; temp50 < int(temp49); temp50++ {
		*(*int16)(&((*T).ColumnFormatCodes[temp50])), err = decoder.Int16()
		if err != nil {
			return
		}

	}

	return
}

func (T *CopyInResponse) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.Int8(int8((*T).Mode))
	if err != nil {
		return
	}

	temp51 := uint16(len((*T).ColumnFormatCodes))

	err = encoder.Uint16(uint16(temp51))
	if err != nil {
		return
	}

	for _, temp52 := range (*T).ColumnFormatCodes {
		err = encoder.Int16(int16(temp52))
		if err != nil {
			return
		}

	}

	return
}

var _ fed.Packet = (*CopyInResponse)(nil)

type CopyOutResponsePayload struct {
	Mode              int8
	ColumnFormatCodes []int16
}

type CopyOutResponse CopyOutResponsePayload

func (T *CopyOutResponse) Type() fed.Type {
	return TypeCopyOutResponse
}

func (T *CopyOutResponse) Length() (length int) {
	length += 1

	temp53 := uint16(len((*T).ColumnFormatCodes))
	_ = temp53

	length += 2

	for _, temp54 := range (*T).ColumnFormatCodes {
		_ = temp54

		length += 2

	}

	return
}

func (T *CopyOutResponse) TypeName() string {
	return "CopyOutResponse"
}

func (T *CopyOutResponse) String() string {
	return T.TypeName()
}

func (T *CopyOutResponse) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*int8)(&((*T).Mode)), err = decoder.Int8()
	if err != nil {
		return
	}
	var temp55 uint16
	*(*uint16)(&(temp55)), err = decoder.Uint16()
	if err != nil {
		return
	}

	(*T).ColumnFormatCodes = slices.Resize((*T).ColumnFormatCodes, int(temp55))

	for temp56 := 0; temp56 < int(temp55); temp56++ {
		*(*int16)(&((*T).ColumnFormatCodes[temp56])), err = decoder.Int16()
		if err != nil {
			return
		}

	}

	return
}

func (T *CopyOutResponse) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.Int8(int8((*T).Mode))
	if err != nil {
		return
	}

	temp57 := uint16(len((*T).ColumnFormatCodes))

	err = encoder.Uint16(uint16(temp57))
	if err != nil {
		return
	}

	for _, temp58 := range (*T).ColumnFormatCodes {
		err = encoder.Int16(int16(temp58))
		if err != nil {
			return
		}

	}

	return
}

var _ fed.Packet = (*CopyOutResponse)(nil)

type DataRow [][]uint8

func (T *DataRow) Type() fed.Type {
	return TypeDataRow
}

func (T *DataRow) Length() (length int) {
	temp59 := uint16(len((*T)))
	_ = temp59

	length += 2

	for _, temp60 := range *T {
		_ = temp60

		temp61 := int32(len(temp60))
		_ = temp61

		length += 4

		for _, temp62 := range temp60 {
			_ = temp62

			length += 1

		}

	}

	return
}

func (T *DataRow) TypeName() string {
	return "DataRow"
}

func (T *DataRow) String() string {
	return T.TypeName()
}

func (T *DataRow) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	var temp63 uint16
	*(*uint16)(&(temp63)), err = decoder.Uint16()
	if err != nil {
		return
	}

	(*T) = slices.Resize((*T), int(temp63))

	for temp64 := 0; temp64 < int(temp63); temp64++ {
		var temp65 int32
		*(*int32)(&(temp65)), err = decoder.Int32()
		if err != nil {
			return
		}

		if temp65 == -1 {
			(*T)[temp64] = nil
		} else {
			if (*T)[temp64] == nil {
				(*T)[temp64] = make([]uint8, int(temp65))
			} else {
				(*T)[temp64] = slices.Resize((*T)[temp64], int(temp65))
			}

			for temp66 := 0; temp66 < int(temp65); temp66++ {
				*(*uint8)(&((*T)[temp64][temp66])), err = decoder.Uint8()
				if err != nil {
					return
				}

			}
		}

	}

	return
}

func (T *DataRow) WriteTo(encoder *fed.Encoder) (err error) {
	temp67 := uint16(len((*T)))

	err = encoder.Uint16(uint16(temp67))
	if err != nil {
		return
	}

	for _, temp68 := range *T {
		temp69 := int32(len(temp68))

		if temp68 == nil {
			temp69 = -1
		}

		err = encoder.Int32(int32(temp69))
		if err != nil {
			return
		}

		for _, temp70 := range temp68 {
			err = encoder.Uint8(uint8(temp70))
			if err != nil {
				return
			}

		}

	}

	return
}

var _ fed.Packet = (*DataRow)(nil)

type DescribePayload struct {
	Which uint8
	Name  string
}

type Describe DescribePayload

func (T *Describe) Type() fed.Type {
	return TypeDescribe
}

func (T *Describe) Length() (length int) {
	length += 1

	length += len((*T).Name) + 1

	return
}

func (T *Describe) TypeName() string {
	return "Describe"
}

func (T *Describe) String() string {
	return T.TypeName()
}

func (T *Describe) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*uint8)(&((*T).Which)), err = decoder.Uint8()
	if err != nil {
		return
	}
	*(*string)(&((*T).Name)), err = decoder.String()
	if err != nil {
		return
	}

	return
}

func (T *Describe) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.Uint8(uint8((*T).Which))
	if err != nil {
		return
	}

	err = encoder.String(string((*T).Name))
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*Describe)(nil)

type EmptyQueryResponse struct{}

func (T *EmptyQueryResponse) Type() fed.Type {
	return TypeEmptyQueryResponse
}

func (T *EmptyQueryResponse) Length() (length int) {

	return
}

func (T *EmptyQueryResponse) TypeName() string {
	return "EmptyQueryResponse"
}

func (T *EmptyQueryResponse) String() string {
	return T.TypeName()
}

func (T *EmptyQueryResponse) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	return
}

func (T *EmptyQueryResponse) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

var _ fed.Packet = (*EmptyQueryResponse)(nil)

type ExecutePayload struct {
	Target  string
	MaxRows uint32
}

type Execute ExecutePayload

func (T *Execute) Type() fed.Type {
	return TypeExecute
}

func (T *Execute) Length() (length int) {
	length += len((*T).Target) + 1

	length += 4

	return
}

func (T *Execute) TypeName() string {
	return "Execute"
}

func (T *Execute) String() string {
	return T.TypeName()
}

func (T *Execute) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*string)(&((*T).Target)), err = decoder.String()
	if err != nil {
		return
	}
	*(*uint32)(&((*T).MaxRows)), err = decoder.Uint32()
	if err != nil {
		return
	}

	return
}

func (T *Execute) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.String(string((*T).Target))
	if err != nil {
		return
	}

	err = encoder.Uint32(uint32((*T).MaxRows))
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*Execute)(nil)

type Flush struct{}

func (T *Flush) Type() fed.Type {
	return TypeFlush
}

func (T *Flush) Length() (length int) {

	return
}

func (T *Flush) TypeName() string {
	return "Flush"
}

func (T *Flush) String() string {
	return T.TypeName()
}

func (T *Flush) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	return
}

func (T *Flush) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

var _ fed.Packet = (*Flush)(nil)

type FunctionCallPayload struct {
	ObjectID            int32
	ArgumentFormatCodes []int16
	Arguments           [][]uint8
	ResultFormatCode    int16
}

type FunctionCall FunctionCallPayload

func (T *FunctionCall) Type() fed.Type {
	return TypeFunctionCall
}

func (T *FunctionCall) Length() (length int) {
	length += 4

	temp71 := uint16(len((*T).ArgumentFormatCodes))
	_ = temp71

	length += 2

	for _, temp72 := range (*T).ArgumentFormatCodes {
		_ = temp72

		length += 2

	}

	temp73 := uint16(len((*T).Arguments))
	_ = temp73

	length += 2

	for _, temp74 := range (*T).Arguments {
		_ = temp74

		temp75 := int32(len(temp74))
		_ = temp75

		length += 4

		for _, temp76 := range temp74 {
			_ = temp76

			length += 1

		}

	}

	length += 2

	return
}

func (T *FunctionCall) TypeName() string {
	return "FunctionCall"
}

func (T *FunctionCall) String() string {
	return T.TypeName()
}

func (T *FunctionCall) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*int32)(&((*T).ObjectID)), err = decoder.Int32()
	if err != nil {
		return
	}
	var temp77 uint16
	*(*uint16)(&(temp77)), err = decoder.Uint16()
	if err != nil {
		return
	}

	(*T).ArgumentFormatCodes = slices.Resize((*T).ArgumentFormatCodes, int(temp77))

	for temp78 := 0; temp78 < int(temp77); temp78++ {
		*(*int16)(&((*T).ArgumentFormatCodes[temp78])), err = decoder.Int16()
		if err != nil {
			return
		}

	}

	var temp79 uint16
	*(*uint16)(&(temp79)), err = decoder.Uint16()
	if err != nil {
		return
	}

	(*T).Arguments = slices.Resize((*T).Arguments, int(temp79))

	for temp80 := 0; temp80 < int(temp79); temp80++ {
		var temp81 int32
		*(*int32)(&(temp81)), err = decoder.Int32()
		if err != nil {
			return
		}

		if temp81 == -1 {
			(*T).Arguments[temp80] = nil
		} else {
			if (*T).Arguments[temp80] == nil {
				(*T).Arguments[temp80] = make([]uint8, int(temp81))
			} else {
				(*T).Arguments[temp80] = slices.Resize((*T).Arguments[temp80], int(temp81))
			}

			for temp82 := 0; temp82 < int(temp81); temp82++ {
				*(*uint8)(&((*T).Arguments[temp80][temp82])), err = decoder.Uint8()
				if err != nil {
					return
				}

			}
		}

	}

	*(*int16)(&((*T).ResultFormatCode)), err = decoder.Int16()
	if err != nil {
		return
	}

	return
}

func (T *FunctionCall) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.Int32(int32((*T).ObjectID))
	if err != nil {
		return
	}

	temp83 := uint16(len((*T).ArgumentFormatCodes))

	err = encoder.Uint16(uint16(temp83))
	if err != nil {
		return
	}

	for _, temp84 := range (*T).ArgumentFormatCodes {
		err = encoder.Int16(int16(temp84))
		if err != nil {
			return
		}

	}

	temp85 := uint16(len((*T).Arguments))

	err = encoder.Uint16(uint16(temp85))
	if err != nil {
		return
	}

	for _, temp86 := range (*T).Arguments {
		temp87 := int32(len(temp86))

		if temp86 == nil {
			temp87 = -1
		}

		err = encoder.Int32(int32(temp87))
		if err != nil {
			return
		}

		for _, temp88 := range temp86 {
			err = encoder.Uint8(uint8(temp88))
			if err != nil {
				return
			}

		}

	}

	err = encoder.Int16(int16((*T).ResultFormatCode))
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*FunctionCall)(nil)

type FunctionCallResponse []uint8

func (T *FunctionCallResponse) Type() fed.Type {
	return TypeFunctionCallResponse
}

func (T *FunctionCallResponse) Length() (length int) {
	temp89 := int32(len((*T)))
	_ = temp89

	length += 4

	for _, temp90 := range *T {
		_ = temp90

		length += 1

	}

	return
}

func (T *FunctionCallResponse) TypeName() string {
	return "FunctionCallResponse"
}

func (T *FunctionCallResponse) String() string {
	return T.TypeName()
}

func (T *FunctionCallResponse) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	var temp91 int32
	*(*int32)(&(temp91)), err = decoder.Int32()
	if err != nil {
		return
	}

	if temp91 == -1 {
		(*T) = nil
	} else {
		if (*T) == nil {
			(*T) = make([]uint8, int(temp91))
		} else {
			(*T) = slices.Resize((*T), int(temp91))
		}

		for temp92 := 0; temp92 < int(temp91); temp92++ {
			*(*uint8)(&((*T)[temp92])), err = decoder.Uint8()
			if err != nil {
				return
			}

		}
	}

	return
}

func (T *FunctionCallResponse) WriteTo(encoder *fed.Encoder) (err error) {
	temp93 := int32(len((*T)))

	if (*T) == nil {
		temp93 = -1
	}

	err = encoder.Int32(int32(temp93))
	if err != nil {
		return
	}

	for _, temp94 := range *T {
		err = encoder.Uint8(uint8(temp94))
		if err != nil {
			return
		}

	}

	return
}

var _ fed.Packet = (*FunctionCallResponse)(nil)

type GSSResponse []uint8

func (T *GSSResponse) Type() fed.Type {
	return TypeGSSResponse
}

func (T *GSSResponse) Length() (length int) {
	for _, temp95 := range *T {
		_ = temp95

		length += 1

	}

	return
}

func (T *GSSResponse) TypeName() string {
	return "GSSResponse"
}

func (T *GSSResponse) String() string {
	return T.TypeName()
}

func (T *GSSResponse) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	(*T) = (*T)[:0]

	for {
		if decoder.Position() >= decoder.Length() {
			break
		}

		(*T) = slices.Resize((*T), len((*T))+1)

		*(*uint8)(&((*T)[len((*T))-1])), err = decoder.Uint8()
		if err != nil {
			return
		}

	}

	return
}

func (T *GSSResponse) WriteTo(encoder *fed.Encoder) (err error) {
	for _, temp96 := range *T {
		err = encoder.Uint8(uint8(temp96))
		if err != nil {
			return
		}

	}

	return
}

var _ fed.Packet = (*GSSResponse)(nil)

type MarkiplierResponseField struct {
	Code  uint8
	Value string
}

type MarkiplierResponse []MarkiplierResponseField

func (T *MarkiplierResponse) Type() fed.Type {
	return TypeMarkiplierResponse
}

func (T *MarkiplierResponse) Length() (length int) {
	for _, temp97 := range *T {
		_ = temp97

		length += 1

		length += len(temp97.Value) + 1

	}

	var temp98 uint8
	_ = temp98

	length += 1

	return
}

func (T *MarkiplierResponse) TypeName() string {
	return "MarkiplierResponse"
}

func (T *MarkiplierResponse) String() string {
	return T.TypeName()
}

func (T *MarkiplierResponse) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	(*T) = (*T)[:0]

	for {
		(*T) = slices.Resize((*T), len((*T))+1)

		*(*uint8)(&((*T)[len((*T))-1].Code)), err = decoder.Uint8()
		if err != nil {
			return
		}
		if (*T)[len((*T))-1].Code == *new(uint8) {
			(*T) = (*T)[:len((*T))-1]
			break
		}
		*(*string)(&((*T)[len((*T))-1].Value)), err = decoder.String()
		if err != nil {
			return
		}
	}

	return
}

func (T *MarkiplierResponse) WriteTo(encoder *fed.Encoder) (err error) {
	for _, temp99 := range *T {
		err = encoder.Uint8(uint8(temp99.Code))
		if err != nil {
			return
		}

		err = encoder.String(string(temp99.Value))
		if err != nil {
			return
		}

	}

	var temp100 uint8

	err = encoder.Uint8(uint8(temp100))
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*MarkiplierResponse)(nil)

type NegotiateProtocolVersionPayload struct {
	MinorProtocolVersion        int32
	UnrecognizedProtocolOptions []string
}

type NegotiateProtocolVersion NegotiateProtocolVersionPayload

func (T *NegotiateProtocolVersion) Type() fed.Type {
	return TypeNegotiateProtocolVersion
}

func (T *NegotiateProtocolVersion) Length() (length int) {
	length += 4

	temp101 := uint32(len((*T).UnrecognizedProtocolOptions))
	_ = temp101

	length += 4

	for _, temp102 := range (*T).UnrecognizedProtocolOptions {
		_ = temp102

		length += len(temp102) + 1

	}

	return
}

func (T *NegotiateProtocolVersion) TypeName() string {
	return "NegotiateProtocolVersion"
}

func (T *NegotiateProtocolVersion) String() string {
	return T.TypeName()
}

func (T *NegotiateProtocolVersion) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*int32)(&((*T).MinorProtocolVersion)), err = decoder.Int32()
	if err != nil {
		return
	}
	var temp103 uint32
	*(*uint32)(&(temp103)), err = decoder.Uint32()
	if err != nil {
		return
	}

	(*T).UnrecognizedProtocolOptions = slices.Resize((*T).UnrecognizedProtocolOptions, int(temp103))

	for temp104 := 0; temp104 < int(temp103); temp104++ {
		*(*string)(&((*T).UnrecognizedProtocolOptions[temp104])), err = decoder.String()
		if err != nil {
			return
		}

	}

	return
}

func (T *NegotiateProtocolVersion) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.Int32(int32((*T).MinorProtocolVersion))
	if err != nil {
		return
	}

	temp105 := uint32(len((*T).UnrecognizedProtocolOptions))

	err = encoder.Uint32(uint32(temp105))
	if err != nil {
		return
	}

	for _, temp106 := range (*T).UnrecognizedProtocolOptions {
		err = encoder.String(string(temp106))
		if err != nil {
			return
		}

	}

	return
}

var _ fed.Packet = (*NegotiateProtocolVersion)(nil)

type NoData struct{}

func (T *NoData) Type() fed.Type {
	return TypeNoData
}

func (T *NoData) Length() (length int) {

	return
}

func (T *NoData) TypeName() string {
	return "NoData"
}

func (T *NoData) String() string {
	return T.TypeName()
}

func (T *NoData) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	return
}

func (T *NoData) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

var _ fed.Packet = (*NoData)(nil)

type NoticeResponseField struct {
	Code  uint8
	Value string
}

type NoticeResponse []NoticeResponseField

func (T *NoticeResponse) Type() fed.Type {
	return TypeNoticeResponse
}

func (T *NoticeResponse) Length() (length int) {
	for _, temp107 := range *T {
		_ = temp107

		length += 1

		length += len(temp107.Value) + 1

	}

	var temp108 uint8
	_ = temp108

	length += 1

	return
}

func (T *NoticeResponse) TypeName() string {
	return "NoticeResponse"
}

func (T *NoticeResponse) String() string {
	return T.TypeName()
}

func (T *NoticeResponse) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	(*T) = (*T)[:0]

	for {
		(*T) = slices.Resize((*T), len((*T))+1)

		*(*uint8)(&((*T)[len((*T))-1].Code)), err = decoder.Uint8()
		if err != nil {
			return
		}
		if (*T)[len((*T))-1].Code == *new(uint8) {
			(*T) = (*T)[:len((*T))-1]
			break
		}
		*(*string)(&((*T)[len((*T))-1].Value)), err = decoder.String()
		if err != nil {
			return
		}
	}

	return
}

func (T *NoticeResponse) WriteTo(encoder *fed.Encoder) (err error) {
	for _, temp109 := range *T {
		err = encoder.Uint8(uint8(temp109.Code))
		if err != nil {
			return
		}

		err = encoder.String(string(temp109.Value))
		if err != nil {
			return
		}

	}

	var temp110 uint8

	err = encoder.Uint8(uint8(temp110))
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*NoticeResponse)(nil)

type NotificationResponsePayload struct {
	ProcessID int32
	Channel   string
	Payload   string
}

type NotificationResponse NotificationResponsePayload

func (T *NotificationResponse) Type() fed.Type {
	return TypeNotificationResponse
}

func (T *NotificationResponse) Length() (length int) {
	length += 4

	length += len((*T).Channel) + 1

	length += len((*T).Payload) + 1

	return
}

func (T *NotificationResponse) TypeName() string {
	return "NotificationResponse"
}

func (T *NotificationResponse) String() string {
	return T.TypeName()
}

func (T *NotificationResponse) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*int32)(&((*T).ProcessID)), err = decoder.Int32()
	if err != nil {
		return
	}
	*(*string)(&((*T).Channel)), err = decoder.String()
	if err != nil {
		return
	}
	*(*string)(&((*T).Payload)), err = decoder.String()
	if err != nil {
		return
	}

	return
}

func (T *NotificationResponse) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.Int32(int32((*T).ProcessID))
	if err != nil {
		return
	}

	err = encoder.String(string((*T).Channel))
	if err != nil {
		return
	}

	err = encoder.String(string((*T).Payload))
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*NotificationResponse)(nil)

type ParameterDescription []int32

func (T *ParameterDescription) Type() fed.Type {
	return TypeParameterDescription
}

func (T *ParameterDescription) Length() (length int) {
	temp111 := uint16(len((*T)))
	_ = temp111

	length += 2

	for _, temp112 := range *T {
		_ = temp112

		length += 4

	}

	return
}

func (T *ParameterDescription) TypeName() string {
	return "ParameterDescription"
}

func (T *ParameterDescription) String() string {
	return T.TypeName()
}

func (T *ParameterDescription) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	var temp113 uint16
	*(*uint16)(&(temp113)), err = decoder.Uint16()
	if err != nil {
		return
	}

	(*T) = slices.Resize((*T), int(temp113))

	for temp114 := 0; temp114 < int(temp113); temp114++ {
		*(*int32)(&((*T)[temp114])), err = decoder.Int32()
		if err != nil {
			return
		}

	}

	return
}

func (T *ParameterDescription) WriteTo(encoder *fed.Encoder) (err error) {
	temp115 := uint16(len((*T)))

	err = encoder.Uint16(uint16(temp115))
	if err != nil {
		return
	}

	for _, temp116 := range *T {
		err = encoder.Int32(int32(temp116))
		if err != nil {
			return
		}

	}

	return
}

var _ fed.Packet = (*ParameterDescription)(nil)

type ParameterStatusPayload struct {
	Key   string
	Value string
}

type ParameterStatus ParameterStatusPayload

func (T *ParameterStatus) Type() fed.Type {
	return TypeParameterStatus
}

func (T *ParameterStatus) Length() (length int) {
	length += len((*T).Key) + 1

	length += len((*T).Value) + 1

	return
}

func (T *ParameterStatus) TypeName() string {
	return "ParameterStatus"
}

func (T *ParameterStatus) String() string {
	return T.TypeName()
}

func (T *ParameterStatus) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*string)(&((*T).Key)), err = decoder.String()
	if err != nil {
		return
	}
	*(*string)(&((*T).Value)), err = decoder.String()
	if err != nil {
		return
	}

	return
}

func (T *ParameterStatus) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.String(string((*T).Key))
	if err != nil {
		return
	}

	err = encoder.String(string((*T).Value))
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*ParameterStatus)(nil)

type ParsePayload struct {
	Destination        string
	Query              string
	ParameterDataTypes []int32
}

type Parse ParsePayload

func (T *Parse) Type() fed.Type {
	return TypeParse
}

func (T *Parse) Length() (length int) {
	length += len((*T).Destination) + 1

	length += len((*T).Query) + 1

	temp117 := uint16(len((*T).ParameterDataTypes))
	_ = temp117

	length += 2

	for _, temp118 := range (*T).ParameterDataTypes {
		_ = temp118

		length += 4

	}

	return
}

func (T *Parse) TypeName() string {
	return "Parse"
}

func (T *Parse) String() string {
	return T.TypeName()
}

func (T *Parse) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*string)(&((*T).Destination)), err = decoder.String()
	if err != nil {
		return
	}
	*(*string)(&((*T).Query)), err = decoder.String()
	if err != nil {
		return
	}
	var temp119 uint16
	*(*uint16)(&(temp119)), err = decoder.Uint16()
	if err != nil {
		return
	}

	(*T).ParameterDataTypes = slices.Resize((*T).ParameterDataTypes, int(temp119))

	for temp120 := 0; temp120 < int(temp119); temp120++ {
		*(*int32)(&((*T).ParameterDataTypes[temp120])), err = decoder.Int32()
		if err != nil {
			return
		}

	}

	return
}

func (T *Parse) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.String(string((*T).Destination))
	if err != nil {
		return
	}

	err = encoder.String(string((*T).Query))
	if err != nil {
		return
	}

	temp121 := uint16(len((*T).ParameterDataTypes))

	err = encoder.Uint16(uint16(temp121))
	if err != nil {
		return
	}

	for _, temp122 := range (*T).ParameterDataTypes {
		err = encoder.Int32(int32(temp122))
		if err != nil {
			return
		}

	}

	return
}

var _ fed.Packet = (*Parse)(nil)

type ParseComplete struct{}

func (T *ParseComplete) Type() fed.Type {
	return TypeParseComplete
}

func (T *ParseComplete) Length() (length int) {

	return
}

func (T *ParseComplete) TypeName() string {
	return "ParseComplete"
}

func (T *ParseComplete) String() string {
	return T.TypeName()
}

func (T *ParseComplete) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	return
}

func (T *ParseComplete) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

var _ fed.Packet = (*ParseComplete)(nil)

type PasswordMessage string

func (T *PasswordMessage) Type() fed.Type {
	return TypePasswordMessage
}

func (T *PasswordMessage) Length() (length int) {
	length += len((*T)) + 1

	return
}

func (T *PasswordMessage) TypeName() string {
	return "PasswordMessage"
}

func (T *PasswordMessage) String() string {
	return T.TypeName()
}

func (T *PasswordMessage) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*string)(&(*T)), err = decoder.String()
	if err != nil {
		return
	}

	return
}

func (T *PasswordMessage) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.String(string((*T)))
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*PasswordMessage)(nil)

type PortalSuspended struct{}

func (T *PortalSuspended) Type() fed.Type {
	return TypePortalSuspended
}

func (T *PortalSuspended) Length() (length int) {

	return
}

func (T *PortalSuspended) TypeName() string {
	return "PortalSuspended"
}

func (T *PortalSuspended) String() string {
	return T.TypeName()
}

func (T *PortalSuspended) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	return
}

func (T *PortalSuspended) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

var _ fed.Packet = (*PortalSuspended)(nil)

type Query string

func (T *Query) Type() fed.Type {
	return TypeQuery
}

func (T *Query) Length() (length int) {
	length += len((*T)) + 1

	return
}

func (T *Query) TypeName() string {
	return "Query"
}

func (T *Query) String() string {
	return T.TypeName()
}

func (T *Query) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*string)(&(*T)), err = decoder.String()
	if err != nil {
		return
	}

	return
}

func (T *Query) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.String(string((*T)))
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*Query)(nil)

type ReadyForQuery uint8

func (T *ReadyForQuery) Type() fed.Type {
	return TypeReadyForQuery
}

func (T *ReadyForQuery) Length() (length int) {
	length += 1

	return
}

func (T *ReadyForQuery) TypeName() string {
	return "ReadyForQuery"
}

func (T *ReadyForQuery) String() string {
	return T.TypeName()
}

func (T *ReadyForQuery) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*uint8)(&(*T)), err = decoder.Uint8()
	if err != nil {
		return
	}

	return
}

func (T *ReadyForQuery) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.Uint8(uint8((*T)))
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*ReadyForQuery)(nil)

type RowDescriptionRow struct {
	Name                  string
	TableID               int32
	ColumnAttributeNumber int16
	FieldDataType         int32
	DataTypeSize          int16
	TypeModifier          int32
	FormatCode            int16
}

type RowDescription []RowDescriptionRow

func (T *RowDescription) Type() fed.Type {
	return TypeRowDescription
}

func (T *RowDescription) Length() (length int) {
	temp123 := uint16(len((*T)))
	_ = temp123

	length += 2

	for _, temp124 := range *T {
		_ = temp124

		length += len(temp124.Name) + 1

		length += 4

		length += 2

		length += 4

		length += 2

		length += 4

		length += 2

	}

	return
}

func (T *RowDescription) TypeName() string {
	return "RowDescription"
}

func (T *RowDescription) String() string {
	return T.TypeName()
}

func (T *RowDescription) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	var temp125 uint16
	*(*uint16)(&(temp125)), err = decoder.Uint16()
	if err != nil {
		return
	}

	(*T) = slices.Resize((*T), int(temp125))

	for temp126 := 0; temp126 < int(temp125); temp126++ {
		*(*string)(&((*T)[temp126].Name)), err = decoder.String()
		if err != nil {
			return
		}
		*(*int32)(&((*T)[temp126].TableID)), err = decoder.Int32()
		if err != nil {
			return
		}
		*(*int16)(&((*T)[temp126].ColumnAttributeNumber)), err = decoder.Int16()
		if err != nil {
			return
		}
		*(*int32)(&((*T)[temp126].FieldDataType)), err = decoder.Int32()
		if err != nil {
			return
		}
		*(*int16)(&((*T)[temp126].DataTypeSize)), err = decoder.Int16()
		if err != nil {
			return
		}
		*(*int32)(&((*T)[temp126].TypeModifier)), err = decoder.Int32()
		if err != nil {
			return
		}
		*(*int16)(&((*T)[temp126].FormatCode)), err = decoder.Int16()
		if err != nil {
			return
		}

	}

	return
}

func (T *RowDescription) WriteTo(encoder *fed.Encoder) (err error) {
	temp127 := uint16(len((*T)))

	err = encoder.Uint16(uint16(temp127))
	if err != nil {
		return
	}

	for _, temp128 := range *T {
		err = encoder.String(string(temp128.Name))
		if err != nil {
			return
		}

		err = encoder.Int32(int32(temp128.TableID))
		if err != nil {
			return
		}

		err = encoder.Int16(int16(temp128.ColumnAttributeNumber))
		if err != nil {
			return
		}

		err = encoder.Int32(int32(temp128.FieldDataType))
		if err != nil {
			return
		}

		err = encoder.Int16(int16(temp128.DataTypeSize))
		if err != nil {
			return
		}

		err = encoder.Int32(int32(temp128.TypeModifier))
		if err != nil {
			return
		}

		err = encoder.Int16(int16(temp128.FormatCode))
		if err != nil {
			return
		}

	}

	return
}

var _ fed.Packet = (*RowDescription)(nil)

type SASLInitialResponsePayload struct {
	Mechanism             string
	InitialClientResponse []uint8
}

type SASLInitialResponse SASLInitialResponsePayload

func (T *SASLInitialResponse) Type() fed.Type {
	return TypeSASLInitialResponse
}

func (T *SASLInitialResponse) Length() (length int) {
	length += len((*T).Mechanism) + 1

	temp129 := int32(len((*T).InitialClientResponse))
	_ = temp129

	length += 4

	for _, temp130 := range (*T).InitialClientResponse {
		_ = temp130

		length += 1

	}

	return
}

func (T *SASLInitialResponse) TypeName() string {
	return "SASLInitialResponse"
}

func (T *SASLInitialResponse) String() string {
	return T.TypeName()
}

func (T *SASLInitialResponse) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	*(*string)(&((*T).Mechanism)), err = decoder.String()
	if err != nil {
		return
	}
	var temp131 int32
	*(*int32)(&(temp131)), err = decoder.Int32()
	if err != nil {
		return
	}

	if temp131 == -1 {
		(*T).InitialClientResponse = nil
	} else {
		if (*T).InitialClientResponse == nil {
			(*T).InitialClientResponse = make([]uint8, int(temp131))
		} else {
			(*T).InitialClientResponse = slices.Resize((*T).InitialClientResponse, int(temp131))
		}

		for temp132 := 0; temp132 < int(temp131); temp132++ {
			*(*uint8)(&((*T).InitialClientResponse[temp132])), err = decoder.Uint8()
			if err != nil {
				return
			}

		}
	}

	return
}

func (T *SASLInitialResponse) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.String(string((*T).Mechanism))
	if err != nil {
		return
	}

	temp133 := int32(len((*T).InitialClientResponse))

	if (*T).InitialClientResponse == nil {
		temp133 = -1
	}

	err = encoder.Int32(int32(temp133))
	if err != nil {
		return
	}

	for _, temp134 := range (*T).InitialClientResponse {
		err = encoder.Uint8(uint8(temp134))
		if err != nil {
			return
		}

	}

	return
}

var _ fed.Packet = (*SASLInitialResponse)(nil)

type SASLResponse []uint8

func (T *SASLResponse) Type() fed.Type {
	return TypeSASLResponse
}

func (T *SASLResponse) Length() (length int) {
	for _, temp135 := range *T {
		_ = temp135

		length += 1

	}

	return
}

func (T *SASLResponse) TypeName() string {
	return "SASLResponse"
}

func (T *SASLResponse) String() string {
	return T.TypeName()
}

func (T *SASLResponse) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	(*T) = (*T)[:0]

	for {
		if decoder.Position() >= decoder.Length() {
			break
		}

		(*T) = slices.Resize((*T), len((*T))+1)

		*(*uint8)(&((*T)[len((*T))-1])), err = decoder.Uint8()
		if err != nil {
			return
		}

	}

	return
}

func (T *SASLResponse) WriteTo(encoder *fed.Encoder) (err error) {
	for _, temp136 := range *T {
		err = encoder.Uint8(uint8(temp136))
		if err != nil {
			return
		}

	}

	return
}

var _ fed.Packet = (*SASLResponse)(nil)

type StartupPayloadControlPayloadCancelKey struct {
	ProcessID int32
	SecretKey int32
}

type StartupPayloadControlPayloadCancel StartupPayloadControlPayloadCancelKey

func (*StartupPayloadControlPayloadCancel) StartupPayloadControlPayloadMode() int16 {
	return 5678
}

func (T *StartupPayloadControlPayloadCancel) Length() (length int) {
	length += 4

	length += 4

	return
}

func (T *StartupPayloadControlPayloadCancel) ReadFrom(decoder *fed.Decoder) (err error) {
	*(*int32)(&((*T).ProcessID)), err = decoder.Int32()
	if err != nil {
		return
	}
	*(*int32)(&((*T).SecretKey)), err = decoder.Int32()
	if err != nil {
		return
	}

	return
}

func (T *StartupPayloadControlPayloadCancel) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.Int32(int32((*T).ProcessID))
	if err != nil {
		return
	}

	err = encoder.Int32(int32((*T).SecretKey))
	if err != nil {
		return
	}

	return
}

type StartupPayloadControlPayloadGSSAPI struct{}

func (*StartupPayloadControlPayloadGSSAPI) StartupPayloadControlPayloadMode() int16 {
	return 5680
}

func (T *StartupPayloadControlPayloadGSSAPI) Length() (length int) {

	return
}

func (T *StartupPayloadControlPayloadGSSAPI) ReadFrom(decoder *fed.Decoder) (err error) {

	return
}

func (T *StartupPayloadControlPayloadGSSAPI) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

type StartupPayloadControlPayloadSSL struct{}

func (*StartupPayloadControlPayloadSSL) StartupPayloadControlPayloadMode() int16 {
	return 5679
}

func (T *StartupPayloadControlPayloadSSL) Length() (length int) {

	return
}

func (T *StartupPayloadControlPayloadSSL) ReadFrom(decoder *fed.Decoder) (err error) {

	return
}

func (T *StartupPayloadControlPayloadSSL) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

type StartupPayloadControlPayloadMode interface {
	StartupPayloadControlPayloadMode() int16

	Length() int
	ReadFrom(decoder *fed.Decoder) error
	WriteTo(encoder *fed.Encoder) error
}
type StartupPayloadControlPayload struct {
	Mode StartupPayloadControlPayloadMode
}

type StartupPayloadControl StartupPayloadControlPayload

func (*StartupPayloadControl) StartupPayloadMode() int16 {
	return 1234
}

func (T *StartupPayloadControl) Length() (length int) {
	length += 2

	length += (*T).Mode.Length()

	return
}

func (T *StartupPayloadControl) ReadFrom(decoder *fed.Decoder) (err error) {
	var temp137 int16

	*(*int16)(&(temp137)), err = decoder.Int16()
	if err != nil {
		return
	}

	switch temp137 {
	case 5678:
		(*T).Mode = new(StartupPayloadControlPayloadCancel)
	case 5680:
		(*T).Mode = new(StartupPayloadControlPayloadGSSAPI)
	case 5679:
		(*T).Mode = new(StartupPayloadControlPayloadSSL)
	default:
		err = ErrInvalidFormat
		return
	}

	err = (*T).Mode.ReadFrom(decoder)
	if err != nil {
		return
	}

	return
}

func (T *StartupPayloadControl) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.Int16(int16((*T).Mode.StartupPayloadControlPayloadMode()))
	if err != nil {
		return
	}

	err = (*T).Mode.WriteTo(encoder)
	if err != nil {
		return
	}

	return
}

type StartupPayloadVersion3PayloadParameter struct {
	Key   string
	Value string
}

type StartupPayloadVersion3Payload struct {
	MinorVersion int16
	Parameters   []StartupPayloadVersion3PayloadParameter
}

type StartupPayloadVersion3 StartupPayloadVersion3Payload

func (*StartupPayloadVersion3) StartupPayloadMode() int16 {
	return 3
}

func (T *StartupPayloadVersion3) Length() (length int) {
	length += 2

	for _, temp138 := range (*T).Parameters {
		_ = temp138

		length += len(temp138.Key) + 1

		length += len(temp138.Value) + 1

	}

	var temp139 string
	_ = temp139

	length += len(temp139) + 1

	return
}

func (T *StartupPayloadVersion3) ReadFrom(decoder *fed.Decoder) (err error) {
	*(*int16)(&((*T).MinorVersion)), err = decoder.Int16()
	if err != nil {
		return
	}
	(*T).Parameters = (*T).Parameters[:0]

	for {
		(*T).Parameters = slices.Resize((*T).Parameters, len((*T).Parameters)+1)

		*(*string)(&((*T).Parameters[len((*T).Parameters)-1].Key)), err = decoder.String()
		if err != nil {
			return
		}
		if (*T).Parameters[len((*T).Parameters)-1].Key == *new(string) {
			(*T).Parameters = (*T).Parameters[:len((*T).Parameters)-1]
			break
		}
		*(*string)(&((*T).Parameters[len((*T).Parameters)-1].Value)), err = decoder.String()
		if err != nil {
			return
		}
	}

	return
}

func (T *StartupPayloadVersion3) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.Int16(int16((*T).MinorVersion))
	if err != nil {
		return
	}

	for _, temp140 := range (*T).Parameters {
		err = encoder.String(string(temp140.Key))
		if err != nil {
			return
		}

		err = encoder.String(string(temp140.Value))
		if err != nil {
			return
		}

	}

	var temp141 string

	err = encoder.String(string(temp141))
	if err != nil {
		return
	}

	return
}

type StartupPayloadMode interface {
	StartupPayloadMode() int16

	Length() int
	ReadFrom(decoder *fed.Decoder) error
	WriteTo(encoder *fed.Encoder) error
}
type StartupPayload struct {
	Mode StartupPayloadMode
}

type Startup StartupPayload

func (T *Startup) Type() fed.Type {
	return 0
}

func (T *Startup) Length() (length int) {
	length += 2

	length += (*T).Mode.Length()

	return
}

func (T *Startup) TypeName() string {
	return "Startup"
}

func (T *Startup) String() string {
	return T.TypeName()
}

func (T *Startup) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	var temp142 int16

	*(*int16)(&(temp142)), err = decoder.Int16()
	if err != nil {
		return
	}

	switch temp142 {
	case 1234:
		(*T).Mode = new(StartupPayloadControl)
	case 3:
		(*T).Mode = new(StartupPayloadVersion3)
	default:
		err = ErrInvalidFormat
		return
	}

	err = (*T).Mode.ReadFrom(decoder)
	if err != nil {
		return
	}

	return
}

func (T *Startup) WriteTo(encoder *fed.Encoder) (err error) {
	err = encoder.Int16(int16((*T).Mode.StartupPayloadMode()))
	if err != nil {
		return
	}

	err = (*T).Mode.WriteTo(encoder)
	if err != nil {
		return
	}

	return
}

var _ fed.Packet = (*Startup)(nil)

type Sync struct{}

func (T *Sync) Type() fed.Type {
	return TypeSync
}

func (T *Sync) Length() (length int) {

	return
}

func (T *Sync) TypeName() string {
	return "Sync"
}

func (T *Sync) String() string {
	return T.TypeName()
}

func (T *Sync) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	return
}

func (T *Sync) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

var _ fed.Packet = (*Sync)(nil)

type Terminate struct{}

func (T *Terminate) Type() fed.Type {
	return TypeTerminate
}

func (T *Terminate) Length() (length int) {

	return
}

func (T *Terminate) TypeName() string {
	return "Terminate"
}

func (T *Terminate) String() string {
	return T.TypeName()
}

func (T *Terminate) ReadFrom(decoder *fed.Decoder) (err error) {
	if decoder.Type() != T.Type() {
		return ErrUnexpectedPacket
	}

	return
}

func (T *Terminate) WriteTo(encoder *fed.Encoder) (err error) {

	return
}

var _ fed.Packet = (*Terminate)(nil)
