package packets

import "pggat2/lib/zap"

func ReadQuery(in zap.ReadablePacket) (string, bool) {
	if in.ReadType() != Query {
		return "", false
	}
	query, ok := in.ReadString()
	if !ok {
		return "", false
	}
	return query, true
}

func WriteQuery(out *zap.Packet, query string) {
	out.WriteType(Query)
	out.WriteString(query)
}
